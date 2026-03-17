package smt

import "math/big"

// DefaultDepth is the standard tree depth for 256-bit keys.
const DefaultDepth = 256

// KeyToPath converts a key (big.Int) to a path of the given depth (LSB-first).
// path[i] = key.Bit(i), matching the TS implementation:
//
//	key.toString(2).padStart(256, "0").split("").reverse().map(Number)
func KeyToPath(key *big.Int, depth int) []byte {
	path := make([]byte, depth)
	for i := 0; i < depth; i++ {
		path[i] = byte(key.Bit(i))
	}
	return path
}

// GetFirstCommonBits returns the number of first common bits between two paths.
func GetFirstCommonBits(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
