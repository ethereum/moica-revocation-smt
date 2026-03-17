package smt

import "math/big"

// MerkleProof represents a membership or non-membership proof.
type MerkleProof struct {
	// Entry is [key] for non-membership or [key, value, 1] for membership.
	Entry []*big.Int
	// MatchingEntry is set for non-membership proofs when another entry
	// exists at the same path prefix. Format: [key, value, 1].
	MatchingEntry []*big.Int
	// Siblings are the sibling nodes along the path from leaf to root.
	Siblings []*big.Int
	// Root is the tree root at the time the proof was created.
	Root *big.Int
	// Membership is true if the key exists in the tree.
	Membership bool
}

// CreateProof generates a membership or non-membership proof for the given key.
func (t *SMT) CreateProof(key *big.Int) *MerkleProof {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entry, matchingEntry, siblings := t.retrieveEntry(key)

	membership := len(entry) >= 2 && entry[1] != nil

	proof := &MerkleProof{
		Entry:      make([]*big.Int, len(entry)),
		Siblings:   make([]*big.Int, len(siblings)),
		Root:       new(big.Int).Set(t.Root),
		Membership: membership,
	}

	for i, v := range entry {
		if v != nil {
			proof.Entry[i] = new(big.Int).Set(v)
		}
	}
	for i, v := range siblings {
		proof.Siblings[i] = new(big.Int).Set(v)
	}

	if matchingEntry != nil {
		proof.MatchingEntry = make([]*big.Int, len(matchingEntry))
		for i, v := range matchingEntry {
			if v != nil {
				proof.MatchingEntry[i] = new(big.Int).Set(v)
			}
		}
	}

	return proof
}

// VerifyProof verifies a membership or non-membership proof.
// The depth parameter specifies the tree depth for path computation.
func VerifyProof(h Hasher, proof *MerkleProof, depth int) bool {
	if proof.MatchingEntry == nil {
		// No matching entry: compute root directly from entry.
		path := KeyToPath(proof.Entry[0], depth)
		var node *big.Int
		if len(proof.Entry) >= 2 && proof.Entry[1] != nil {
			// Membership proof: node is hash of entry
			node = h.Hash3(proof.Entry[0], proof.Entry[1], proof.Entry[2])
		} else {
			// Non-membership proof with no matching entry: node is zero
			node = new(big.Int)
		}
		root := calculateRootStatic(h, node, path, proof.Siblings)
		return root.Cmp(proof.Root) == 0
	}

	// Non-membership proof with matching entry.
	matchingPath := KeyToPath(proof.MatchingEntry[0], depth)
	node := h.Hash3(proof.MatchingEntry[0], proof.MatchingEntry[1], proof.MatchingEntry[2])
	root := calculateRootStatic(h, node, matchingPath, proof.Siblings)

	if root.Cmp(proof.Root) == 0 {
		path := KeyToPath(proof.Entry[0], depth)
		firstCommon := GetFirstCommonBits(path, matchingPath)
		return len(proof.Siblings) <= firstCommon
	}

	return false
}

// calculateRootStatic computes root from node, path, and siblings without modifying tree state.
func calculateRootStatic(h Hasher, node *big.Int, path []byte, siblings []*big.Int) *big.Int {
	n := new(big.Int).Set(node)
	for i := len(siblings) - 1; i >= 0; i-- {
		if path[i] == 1 {
			n = h.Hash2(siblings[i], n)
		} else {
			n = h.Hash2(n, siblings[i])
		}
	}
	return n
}
