package chain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/moven0831/moica-revocation-smt/server/internal/chain/contract"
)

// Issuer IDs are keccak256 hashes used as on-chain identifiers.
var (
	IssuerG2 = crypto.Keccak256Hash([]byte("MOICA-G2"))
	IssuerG3 = crypto.Keccak256Hash([]byte("MOICA-G3"))
)

// maxReplacementAttempts is the default number of fee-bumped resends the relayer
// tries when a transaction is not confirmed within the confirm timeout.
const maxReplacementAttempts = 3

// ErrFeeCeilingExceeded is returned when the live gas fee (or a required fee bump)
// would exceed the configured ceiling. Callers can detect it with errors.Is and
// skip posting rather than overpay during a gas spike.
var ErrFeeCeilingExceeded = errors.New("gas fee exceeds configured ceiling")

// EthBackend combines the interfaces needed for contract interaction and transaction mining.
type EthBackend interface {
	bind.ContractBackend
	bind.DeployBackend
}

// Relayer signs and sends setRoot transactions to SMTRootStorage.
type Relayer struct {
	client          *Client
	backend         EthBackend
	privateKey      *ecdsa.PrivateKey
	contractAddress common.Address
	chainID         *big.Int

	// maxFeeWei caps the EIP-1559 fee per gas. nil means no ceiling.
	maxFeeWei *big.Int
	// confirmTimeout bounds how long to wait for each send before attempting a
	// fee-bumped replacement. 0 means wait on the caller's context with no replacement.
	confirmTimeout time.Duration
	// maxReplacements is the number of fee-bumped resends attempted on timeout.
	maxReplacements int
}

// RelayerOption configures optional relayer behavior.
type RelayerOption func(*Relayer)

// WithMaxFeeGwei sets the maximum gas fee per gas the relayer will pay, in gwei.
// A value <= 0 leaves the ceiling disabled (no cap).
func WithMaxFeeGwei(gwei int) RelayerOption {
	return func(r *Relayer) {
		if gwei > 0 {
			r.maxFeeWei = new(big.Int).Mul(big.NewInt(int64(gwei)), big.NewInt(params.GWei))
		}
	}
}

// WithConfirmTimeout sets the per-transaction confirmation timeout. On timeout the
// relayer resends at the same nonce with a bumped fee. A value <= 0 disables
// replacement and waits on the caller's context.
func WithConfirmTimeout(d time.Duration) RelayerOption {
	return func(r *Relayer) {
		if d > 0 {
			r.confirmTimeout = d
		}
	}
}

// NewRelayer creates a relayer from a hex private key and contract address.
func NewRelayer(client *Client, privateKeyHex string, contractAddr string, opts ...RelayerOption) (*Relayer, error) {
	pk, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}

	r := &Relayer{
		client:          client,
		backend:         client.Eth(),
		privateKey:      pk,
		contractAddress: common.HexToAddress(contractAddr),
		chainID:         chainID,
		maxReplacements: maxReplacementAttempts,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r, nil
}

// TransactOpts returns a signed transactor for the relayer.
func (r *Relayer) TransactOpts(ctx context.Context) (*bind.TransactOpts, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(r.privateKey, r.chainID)
	if err != nil {
		return nil, err
	}
	opts.Context = ctx
	return opts, nil
}

// Address returns the relayer's Ethereum address.
func (r *Relayer) Address() common.Address {
	return crypto.PubkeyToAddress(r.privateKey.PublicKey)
}

// ContractAddress returns the contract address.
func (r *Relayer) ContractAddress() common.Address {
	return r.contractAddress
}

// VerifyContract checks that the contract is reachable and the on-chain relayer matches this relayer's address.
func (r *Relayer) VerifyContract(ctx context.Context) error {
	instance, err := contract.NewSMTRootStorage(r.contractAddress, r.backend)
	if err != nil {
		return fmt.Errorf("bind contract: %w", err)
	}

	onChainRelayer, err := instance.Relayer(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("query relayer(): %w", err)
	}

	expected := r.Address()
	if onChainRelayer != expected {
		return fmt.Errorf("relayer mismatch: contract has %s, expected %s", onChainRelayer.Hex(), expected.Hex())
	}

	return nil
}

// MaxPostDuration is the worst-case time a single PostRoot may take, given the
// confirm timeout and the fee-bumped resends. Callers use it to size deadlines.
func (r *Relayer) MaxPostDuration() time.Duration {
	return time.Duration(r.maxReplacements+1) * r.confirmTimeout
}

// PostRoot sends a setRoot transaction to the SMTRootStorage contract and waits for
// confirmation, bounding the fee by the configured ceiling and resending at the same
// nonce with a bumped fee if it is not mined within the confirm timeout.
func (r *Relayer) PostRoot(ctx context.Context, issuerID [32]byte, root *big.Int, crlNumber *big.Int) (*types.Transaction, error) {
	instance, err := contract.NewSMTRootStorage(r.contractAddress, r.backend)
	if err != nil {
		return nil, fmt.Errorf("bind contract: %w", err)
	}

	opts, err := r.TransactOpts(ctx)
	if err != nil {
		return nil, fmt.Errorf("transact opts: %w", err)
	}

	if err := r.applyGasBounds(ctx, opts); err != nil {
		return nil, err
	}
	return r.sendAndConfirm(ctx, opts, func(o *bind.TransactOpts) (*types.Transaction, error) {
		return instance.SetRoot(o, issuerID, root, crlNumber)
	})
}

// applyGasBounds sets EIP-1559 fees bounded by the configured ceiling and pins the
// nonce. It returns ErrFeeCeilingExceeded when base fee + tip already exceeds the ceiling.
func (r *Relayer) applyGasBounds(ctx context.Context, opts *bind.TransactOpts) error {
	tip, err := r.backend.SuggestGasTipCap(ctx)
	if err != nil {
		return fmt.Errorf("suggest gas tip: %w", err)
	}
	head, err := r.backend.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("fetch head: %w", err)
	}
	if head.BaseFee == nil {
		return fmt.Errorf("chain has no base fee (pre-EIP-1559 not supported)")
	}

	minFee := new(big.Int).Add(head.BaseFee, tip)
	if r.maxFeeWei != nil && minFee.Cmp(r.maxFeeWei) > 0 {
		return fmt.Errorf("%w: base fee + tip %s gwei > ceiling %s gwei",
			ErrFeeCeilingExceeded, weiToGwei(minFee), weiToGwei(r.maxFeeWei))
	}

	// 2*baseFee + tip leaves headroom for a few blocks of base-fee growth.
	feeCap := new(big.Int).Add(new(big.Int).Lsh(head.BaseFee, 1), tip)
	if r.maxFeeWei != nil && feeCap.Cmp(r.maxFeeWei) > 0 {
		feeCap = new(big.Int).Set(r.maxFeeWei)
	}
	opts.GasTipCap = tip
	opts.GasFeeCap = feeCap
	return r.pinNonce(ctx, opts)
}

// pinNonce fixes the transaction nonce so that every resend in sendAndConfirm
// reuses it, replacing the prior tx rather than queuing a new one behind it.
func (r *Relayer) pinNonce(ctx context.Context, opts *bind.TransactOpts) error {
	nonce, err := r.backend.PendingNonceAt(ctx, r.Address())
	if err != nil {
		return fmt.Errorf("fetch nonce: %w", err)
	}
	opts.Nonce = new(big.Int).SetUint64(nonce)
	return nil
}

// sendAndConfirm sends the transaction and waits for confirmation, resending at the
// same nonce with a bumped fee if it is not mined within confirmTimeout.
func (r *Relayer) sendAndConfirm(ctx context.Context, opts *bind.TransactOpts, send func(*bind.TransactOpts) (*types.Transaction, error)) (*types.Transaction, error) {
	for attempt := 0; ; attempt++ {
		tx, err := send(opts)
		if err != nil {
			return nil, fmt.Errorf("setRoot: %w", err)
		}

		waitCtx := ctx
		var cancel context.CancelFunc
		if r.confirmTimeout > 0 {
			waitCtx, cancel = context.WithTimeout(ctx, r.confirmTimeout)
		}
		receipt, err := bind.WaitMined(waitCtx, r.backend, tx)
		if cancel != nil {
			cancel()
		}

		if err == nil {
			if receipt.Status == 0 {
				return tx, fmt.Errorf("transaction reverted: %s", tx.Hash().Hex())
			}
			return tx, nil
		}

		// A per-attempt timeout (parent ctx still alive) is retryable; the parent
		// ctx expiring or being canceled is terminal.
		if ctx.Err() != nil {
			return tx, fmt.Errorf("wait mined: %w", err)
		}
		if attempt >= r.maxReplacements {
			return tx, fmt.Errorf("tx %s not confirmed after %d attempt(s) at nonce %v; still pending: %w",
				tx.Hash().Hex(), attempt+1, opts.Nonce, err)
		}
		if bumpErr := r.bumpFees(opts); bumpErr != nil {
			return tx, bumpErr
		}
	}
}

// bumpFees raises the transaction fee by ~13% (above the 10% replacement minimum),
// staying within the ceiling. It returns ErrFeeCeilingExceeded when a legal bump
// would exceed the ceiling, meaning the tx is stuck at the cap and we refuse to overpay.
func (r *Relayer) bumpFees(opts *bind.TransactOpts) error {
	// +13% (clears the node's 10% replacement threshold), with a 1-wei floor so
	// even tiny fees strictly increase and integer truncation can't stall a resend.
	bump := func(v *big.Int) *big.Int {
		inc := new(big.Int).Div(new(big.Int).Mul(v, big.NewInt(13)), big.NewInt(100))
		if inc.Sign() == 0 {
			inc = big.NewInt(1)
		}
		return new(big.Int).Add(v, inc)
	}

	nextFeeCap := bump(opts.GasFeeCap)
	if r.maxFeeWei != nil && nextFeeCap.Cmp(r.maxFeeWei) > 0 {
		return fmt.Errorf("%w: cannot bump fee cap above %s gwei (tx stuck at ceiling)",
			ErrFeeCeilingExceeded, weiToGwei(r.maxFeeWei))
	}
	opts.GasTipCap = bump(opts.GasTipCap)
	opts.GasFeeCap = nextFeeCap
	return nil
}

// weiToGwei formats a wei amount as a gwei string with two decimals, for log/error messages.
func weiToGwei(wei *big.Int) string {
	return new(big.Rat).SetFrac(wei, big.NewInt(params.GWei)).FloatString(2)
}
