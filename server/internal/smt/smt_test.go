package smt

import (
	"math/big"
	"testing"
)

// hexToBig converts a hex string (with or without 0x prefix) to *big.Int.
func hexToBig(s string) *big.Int {
	if len(s) >= 2 && s[:2] == "0x" {
		s = s[2:]
	}
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("invalid hex: " + s)
	}
	return n
}

// Test vectors generated from TS project using gen-vectors.ts
var (
	serial1 = hexToBig("100048210DD2DF2E128096A9282B5EC5")
	serial2 = hexToBig("200048210DD2DF2E128096A9282B5EC5")
	serial3 = hexToBig("300048210DD2DF2E128096A9282B5EC5")
	serial4 = hexToBig("400048210DD2DF2E128096A9282B5EC5") // non-member

	// Expected Poseidon hash values
	expectedHash2_1_2     = hexToBig("0x15d5f9b94530aeb9c69c47e44e2fca2e73ea185df9de41e24c9b397124b283dc")
	expectedHash3_1_1_1   = hexToBig("0x387101fb2a91af1c1c3683c122e6ab8567332c135c897dbcf7ca755889b16c4b")
	expectedHash2_0_0     = hexToBig("0x63e0e78ba24a2b23621bae4e4ac184b0eed6eb5240a51120b44793982be151a7")

	// Expected roots after each insertion
	rootAfterAdd1 = hexToBig("0xa8745054a4d00e0d8760deb3f969867396672ff350bbe6e9887169f2b9e87ea1")
	rootAfterAdd2 = hexToBig("0x8674dee5a21dcde3c9f857ac77b6afb1110b2ce392dc012337254c6d647ff67e")
	rootAfterAdd3 = hexToBig("0x2a28d47e11d101db7c58da1fa2bb6bcf350490aed93c94b181f49957d078fd55")

	// Root after deleting serial2
	rootAfterDelete2 = hexToBig("0xc49d1e0a6ac1fcd07db35ab6ca24143b29dd4be85e94d1f52968551cf9adba6a")
)

func TestPoseidonHashCompatibility(t *testing.T) {
	h := NewPoseidonHasher()

	tests := []struct {
		name     string
		a, b     *big.Int
		c        *big.Int // nil for Hash2
		expected *big.Int
	}{
		{"Hash2(1, 2)", big.NewInt(1), big.NewInt(2), nil, expectedHash2_1_2},
		{"Hash3(1, 1, 1)", big.NewInt(1), big.NewInt(1), big.NewInt(1), expectedHash3_1_1_1},
		{"Hash2(0, 0)", big.NewInt(0), big.NewInt(0), nil, expectedHash2_0_0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got *big.Int
			if tc.c != nil {
				got = h.Hash3(tc.a, tc.b, tc.c)
			} else {
				got = h.Hash2(tc.a, tc.b)
			}
			if got.Cmp(tc.expected) != 0 {
				t.Errorf("got %s, want %s", got.Text(16), tc.expected.Text(16))
			}
		})
	}
}

func TestKeyToPath(t *testing.T) {
	depth := DefaultDepth

	// key=0 should give all zeros
	path0 := KeyToPath(big.NewInt(0), depth)
	for i := 0; i < depth; i++ {
		if path0[i] != 0 {
			t.Fatalf("path[%d] = %d, want 0 for key=0", i, path0[i])
		}
	}

	// key=1 should have bit 0 = 1, rest 0
	path1 := KeyToPath(big.NewInt(1), depth)
	if path1[0] != 1 {
		t.Fatal("path[0] should be 1 for key=1")
	}
	for i := 1; i < depth; i++ {
		if path1[i] != 0 {
			t.Fatalf("path[%d] = %d, want 0 for key=1", i, path1[i])
		}
	}

	// key=5 (binary 101) → path[0]=1, path[1]=0, path[2]=1 (LSB first)
	path5 := KeyToPath(big.NewInt(5), depth)
	if path5[0] != 1 || path5[1] != 0 || path5[2] != 1 {
		t.Fatalf("path for key=5: got [%d,%d,%d], want [1,0,1]", path5[0], path5[1], path5[2])
	}
}

func TestSMTAddAndRoot(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	// Add serial1
	if err := tree.Add(serial1, big.NewInt(1)); err != nil {
		t.Fatal(err)
	}
	if tree.Root.Cmp(rootAfterAdd1) != 0 {
		t.Errorf("root after add1: got %s, want %s", tree.Root.Text(16), rootAfterAdd1.Text(16))
	}

	// Add serial2
	if err := tree.Add(serial2, big.NewInt(1)); err != nil {
		t.Fatal(err)
	}
	if tree.Root.Cmp(rootAfterAdd2) != 0 {
		t.Errorf("root after add2: got %s, want %s", tree.Root.Text(16), rootAfterAdd2.Text(16))
	}

	// Add serial3
	if err := tree.Add(serial3, big.NewInt(1)); err != nil {
		t.Fatal(err)
	}
	if tree.Root.Cmp(rootAfterAdd3) != 0 {
		t.Errorf("root after add3: got %s, want %s", tree.Root.Text(16), rootAfterAdd3.Text(16))
	}

	// Count
	if tree.Count != 3 {
		t.Errorf("count: got %d, want 3", tree.Count)
	}
}

func TestSMTGet(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	tree.Add(serial1, big.NewInt(1))

	val := tree.Get(serial1)
	if val == nil || val.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("get serial1: got %v, want 1", val)
	}

	val = tree.Get(serial4)
	if val != nil {
		t.Errorf("get serial4 (non-member): got %v, want nil", val)
	}
}

func TestSMTDuplicateKey(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	tree.Add(serial1, big.NewInt(1))
	err := tree.Add(serial1, big.NewInt(1))
	if err == nil {
		t.Fatal("expected error for duplicate key")
	}
}

func TestSMTDelete(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	tree.Add(serial1, big.NewInt(1))
	tree.Add(serial2, big.NewInt(1))
	tree.Add(serial3, big.NewInt(1))

	// Delete serial2
	if err := tree.Delete(serial2); err != nil {
		t.Fatal(err)
	}

	if tree.Root.Cmp(rootAfterDelete2) != 0 {
		t.Errorf("root after delete: got %s, want %s", tree.Root.Text(16), rootAfterDelete2.Text(16))
	}

	if tree.Count != 2 {
		t.Errorf("count after delete: got %d, want 2", tree.Count)
	}

	// serial2 should no longer be found
	if val := tree.Get(serial2); val != nil {
		t.Errorf("get deleted key: got %v, want nil", val)
	}
}

func TestSMTDeleteNonExistent(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	err := tree.Delete(serial1)
	if err == nil {
		t.Fatal("expected error for deleting non-existent key")
	}
}

func TestMembershipProof(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	tree.Add(serial1, big.NewInt(1))
	tree.Add(serial2, big.NewInt(1))
	tree.Add(serial3, big.NewInt(1))

	proof := tree.CreateProof(serial1)

	if !proof.Membership {
		t.Fatal("expected membership=true")
	}
	if proof.Root.Cmp(rootAfterAdd3) != 0 {
		t.Errorf("proof root: got %s, want %s", proof.Root.Text(16), rootAfterAdd3.Text(16))
	}
	if len(proof.Entry) != 3 {
		t.Fatalf("entry length: got %d, want 3", len(proof.Entry))
	}
	if proof.Entry[0].Cmp(serial1) != 0 {
		t.Error("entry[0] should be the key")
	}
	if proof.Entry[1].Cmp(big.NewInt(1)) != 0 {
		t.Error("entry[1] should be 1")
	}
	if proof.MatchingEntry != nil {
		t.Error("matchingEntry should be nil for membership proof")
	}

	// Verify proof
	if !VerifyProof(h, proof, DefaultDepth) {
		t.Fatal("membership proof verification failed")
	}

	// Check siblings: 126 siblings, first 124 are zero, last 2 non-zero
	if len(proof.Siblings) != 126 {
		t.Fatalf("siblings length: got %d, want 126", len(proof.Siblings))
	}
	for i := 0; i < 124; i++ {
		if proof.Siblings[i].Sign() != 0 {
			t.Errorf("sibling[%d] should be zero", i)
		}
	}
	expectedSibling124 := hexToBig("0xb38420e2097e07feba845b3c061e15ad70d0e3e66ef91a8432aaeec4e1a0990d")
	expectedSibling125 := hexToBig("0xb914872470b21867929bc68b7a27e890ba5a245cbea9e4eddf1e6a6a1bc58f7a")
	if proof.Siblings[124].Cmp(expectedSibling124) != 0 {
		t.Errorf("sibling[124]: got %s, want %s", proof.Siblings[124].Text(16), expectedSibling124.Text(16))
	}
	if proof.Siblings[125].Cmp(expectedSibling125) != 0 {
		t.Errorf("sibling[125]: got %s, want %s", proof.Siblings[125].Text(16), expectedSibling125.Text(16))
	}
}

func TestNonMembershipProof(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	tree.Add(serial1, big.NewInt(1))
	tree.Add(serial2, big.NewInt(1))
	tree.Add(serial3, big.NewInt(1))

	proof := tree.CreateProof(serial4)

	if proof.Membership {
		t.Fatal("expected membership=false")
	}
	if proof.Root.Cmp(rootAfterAdd3) != 0 {
		t.Errorf("proof root: got %s, want %s", proof.Root.Text(16), rootAfterAdd3.Text(16))
	}

	// Entry should be [key] only
	if len(proof.Entry) != 1 {
		t.Fatalf("entry length: got %d, want 1", len(proof.Entry))
	}
	if proof.Entry[0].Cmp(serial4) != 0 {
		t.Error("entry[0] should be the queried key")
	}

	// MatchingEntry should be serial2's leaf
	if proof.MatchingEntry == nil {
		t.Fatal("matchingEntry should not be nil")
	}
	if proof.MatchingEntry[0].Cmp(serial2) != 0 {
		t.Errorf("matchingEntry[0]: got %s, want %s", proof.MatchingEntry[0].Text(16), serial2.Text(16))
	}

	// Verify proof
	if !VerifyProof(h, proof, DefaultDepth) {
		t.Fatal("non-membership proof verification failed")
	}

	// Check siblings: 125 siblings, first 124 zero, last one non-zero
	if len(proof.Siblings) != 125 {
		t.Fatalf("siblings length: got %d, want 125", len(proof.Siblings))
	}
	expectedSibling124 := hexToBig("0x92be5fe55d1fc933569429dd6050f6eca78efd4d0a777bfae335138bdf65a3af")
	if proof.Siblings[124].Cmp(expectedSibling124) != 0 {
		t.Errorf("sibling[124]: got %s, want %s", proof.Siblings[124].Text(16), expectedSibling124.Text(16))
	}
}

func TestProofVerificationTamper(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	tree.Add(serial1, big.NewInt(1))
	tree.Add(serial2, big.NewInt(1))

	proof := tree.CreateProof(serial1)

	// Tamper with root
	tamperedProof := &MerkleProof{
		Entry:    proof.Entry,
		Siblings: proof.Siblings,
		Root:     new(big.Int).Add(proof.Root, big.NewInt(1)),
		Membership: proof.Membership,
	}
	if VerifyProof(h, tamperedProof, DefaultDepth) {
		t.Fatal("tampered proof should not verify")
	}
}

func TestEmptyTreeProof(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	proof := tree.CreateProof(serial1)

	if proof.Membership {
		t.Fatal("expected non-membership in empty tree")
	}
	if proof.Root.Sign() != 0 {
		t.Error("root should be zero for empty tree")
	}
	if len(proof.Siblings) != 0 {
		t.Error("siblings should be empty for empty tree")
	}

	if !VerifyProof(h, proof, DefaultDepth) {
		t.Fatal("empty tree non-membership proof should verify")
	}
}

func TestSingleEntryTree(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	tree.Add(serial1, big.NewInt(1))

	// The root of a single-entry tree is hash(key, value, 1)
	expectedRoot := h.Hash3(serial1, big.NewInt(1), big.NewInt(1))
	if tree.Root.Cmp(expectedRoot) != 0 {
		t.Errorf("single entry root: got %s, want %s", tree.Root.Text(16), expectedRoot.Text(16))
	}

	// Membership proof should have 0 siblings
	proof := tree.CreateProof(serial1)
	if !proof.Membership {
		t.Fatal("expected membership")
	}
	if len(proof.Siblings) != 0 {
		t.Errorf("siblings length: got %d, want 0", len(proof.Siblings))
	}
	if !VerifyProof(h, proof, DefaultDepth) {
		t.Fatal("single entry proof verification failed")
	}
}

func TestDeleteAllEntries(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	tree.Add(serial1, big.NewInt(1))
	tree.Add(serial2, big.NewInt(1))

	tree.Delete(serial1)
	tree.Delete(serial2)

	if tree.Root.Sign() != 0 {
		t.Error("root should be zero after deleting all entries")
	}
	if tree.Count != 0 {
		t.Errorf("count: got %d, want 0", tree.Count)
	}
}

func TestAddKeyExceedingDepth(t *testing.T) {
	h := NewPoseidonHasher()
	tree := New(h)

	// A 129-bit key should be rejected at depth 128
	bigKey := new(big.Int).Lsh(big.NewInt(1), 128) // 2^128
	err := tree.Add(bigKey, big.NewInt(1))
	if err == nil {
		t.Fatal("expected error for key exceeding tree depth")
	}
	if tree.Count != 0 {
		t.Errorf("count should be 0 after rejected add, got %d", tree.Count)
	}
}

func TestDeterministicRoot(t *testing.T) {
	h := NewPoseidonHasher()

	// Build tree in order 1,2,3
	tree1 := New(h)
	tree1.Add(serial1, big.NewInt(1))
	tree1.Add(serial2, big.NewInt(1))
	tree1.Add(serial3, big.NewInt(1))

	// Build tree in order 3,1,2
	tree2 := New(h)
	tree2.Add(serial3, big.NewInt(1))
	tree2.Add(serial1, big.NewInt(1))
	tree2.Add(serial2, big.NewInt(1))

	if tree1.Root.Cmp(tree2.Root) != 0 {
		t.Errorf("different insertion order should give same root: %s vs %s",
			tree1.Root.Text(16), tree2.Root.Text(16))
	}
}
