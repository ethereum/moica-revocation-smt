package smt

import (
	"math/big"

	poseidon "github.com/zkmopro/go-poseidon-p256"
)

// Hasher defines the interface for hash functions used by the SMT.
type Hasher interface {
	// Hash2 hashes two field elements (used for branch nodes).
	Hash2(a, b *big.Int) *big.Int
	// Hash3 hashes three field elements (used for leaf nodes).
	Hash3(a, b, c *big.Int) *big.Int
}

// PoseidonHasher implements Hasher using Poseidon over the P-256 base field.
type PoseidonHasher struct{}

func NewPoseidonHasher() *PoseidonHasher {
	return &PoseidonHasher{}
}

func (h *PoseidonHasher) Hash2(a, b *big.Int) *big.Int {
	return poseidon.Hash2(a, b)
}

func (h *PoseidonHasher) Hash3(a, b, c *big.Int) *big.Int {
	return poseidon.Hash3(a, b, c)
}
