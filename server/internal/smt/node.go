package smt

import "math/big"

// ChildNodes represents the children of a node in the SMT.
// For branch nodes: [left, right] (2 elements)
// For leaf nodes: [key, value, entryMark=1] (3 elements)
type ChildNodes []*big.Int

// IsLeaf returns true if the node is a leaf (has 3 children with entryMark).
func (cn ChildNodes) IsLeaf() bool {
	return len(cn) == 3 && cn[2] != nil
}

// Clone creates a deep copy of child nodes.
func (cn ChildNodes) Clone() ChildNodes {
	result := make(ChildNodes, len(cn))
	for i, v := range cn {
		if v != nil {
			result[i] = new(big.Int).Set(v)
		}
	}
	return result
}
