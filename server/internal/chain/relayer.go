package chain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/moven0831/moica-revocation-smt/server/internal/chain/contract"
)

// Issuer IDs are keccak256 hashes used as on-chain identifiers.
var (
	IssuerG2 = crypto.Keccak256Hash([]byte("MOICA-G2"))
	IssuerG3 = crypto.Keccak256Hash([]byte("MOICA-G3"))
)

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
}

// NewRelayer creates a relayer from a hex private key and contract address.
func NewRelayer(client *Client, privateKeyHex string, contractAddr string) (*Relayer, error) {
	pk, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get chain ID: %w", err)
	}

	return &Relayer{
		client:          client,
		backend:         client.Eth(),
		privateKey:      pk,
		contractAddress: common.HexToAddress(contractAddr),
		chainID:         chainID,
	}, nil
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

// PostRoot sends a setRoot transaction to the SMTRootStorage contract and waits for confirmation.
func (r *Relayer) PostRoot(ctx context.Context, issuerID [32]byte, root *big.Int, crlNumber *big.Int) (*types.Transaction, error) {
	instance, err := contract.NewSMTRootStorage(r.contractAddress, r.backend)
	if err != nil {
		return nil, fmt.Errorf("bind contract: %w", err)
	}

	opts, err := r.TransactOpts(ctx)
	if err != nil {
		return nil, fmt.Errorf("transact opts: %w", err)
	}

	tx, err := instance.SetRoot(opts, issuerID, root, crlNumber)
	if err != nil {
		return nil, fmt.Errorf("setRoot: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, r.backend, tx)
	if err != nil {
		return tx, fmt.Errorf("wait mined: %w", err)
	}
	if receipt.Status == 0 {
		return tx, fmt.Errorf("transaction reverted: %s", tx.Hash().Hex())
	}

	return tx, nil
}
