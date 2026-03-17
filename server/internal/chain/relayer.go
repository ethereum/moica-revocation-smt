package chain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Relayer signs and sends setRoot transactions to SMTRootStorage.
type Relayer struct {
	client          *Client
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
