package chain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

// Client wraps an Ethereum JSON-RPC client.
type Client struct {
	eth *ethclient.Client
}

// NewClient connects to an Ethereum node.
func NewClient(rpcURL string) (*Client, error) {
	eth, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	return &Client{eth: eth}, nil
}

// ChainID returns the chain ID.
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	return c.eth.ChainID(ctx)
}

// Close disconnects the client.
func (c *Client) Close() {
	c.eth.Close()
}

// Eth returns the underlying ethclient for advanced usage.
func (c *Client) Eth() *ethclient.Client {
	return c.eth
}
