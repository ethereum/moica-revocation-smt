package snapshot

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

func TestSnapshotRoundTrip(t *testing.T) {
	h := smt.NewPoseidonHasher()
	tree := smt.New(h)

	serials := []string{
		"100048210DD2DF2E128096A9282B5EC5",
		"200048210DD2DF2E128096A9282B5EC5",
		"300048210DD2DF2E128096A9282B5EC5",
	}

	for _, s := range serials {
		key, _ := new(big.Int).SetString(s, 16)
		tree.Add(key, big.NewInt(1))
	}

	originalRoot := new(big.Int).Set(tree.Root)
	originalCount := tree.Count

	// Export
	var buf bytes.Buffer
	if err := Export(tree, 0, &buf); err != nil {
		t.Fatal("export:", err)
	}

	// Import
	restored, _, err := Import(h, &buf)
	if err != nil {
		t.Fatal("import:", err)
	}

	// Verify root and count match
	if restored.Root.Cmp(originalRoot) != 0 {
		t.Errorf("root mismatch: got %s, want %s", restored.Root.Text(16), originalRoot.Text(16))
	}
	if restored.Count != originalCount {
		t.Errorf("count mismatch: got %d, want %d", restored.Count, originalCount)
	}

	// Verify membership proof works on restored tree
	key, _ := new(big.Int).SetString(serials[0], 16)
	proof := restored.CreateProof(key)
	if !proof.Membership {
		t.Fatal("membership proof failed on restored tree")
	}
	if !smt.VerifyProof(h, proof, smt.DefaultDepth) {
		t.Fatal("proof verification failed on restored tree")
	}

	// Verify non-membership proof works on restored tree
	nonMember, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)
	nonProof := restored.CreateProof(nonMember)
	if nonProof.Membership {
		t.Fatal("non-member should not be member")
	}
	if !smt.VerifyProof(h, nonProof, smt.DefaultDepth) {
		t.Fatal("non-membership proof verification failed on restored tree")
	}
}

func TestSnapshotEmpty(t *testing.T) {
	h := smt.NewPoseidonHasher()
	tree := smt.New(h)

	var buf bytes.Buffer
	if err := Export(tree, 0, &buf); err != nil {
		t.Fatal("export empty:", err)
	}

	restored, _, err := Import(h, &buf)
	if err != nil {
		t.Fatal("import empty:", err)
	}

	if restored.Root.Sign() != 0 {
		t.Error("empty tree root should be zero")
	}
	if restored.Count != 0 {
		t.Error("empty tree count should be zero")
	}
}

func TestSnapshotCRLNumber(t *testing.T) {
	h := smt.NewPoseidonHasher()
	tree := smt.New(h)

	key, _ := new(big.Int).SetString("ABCDEF", 16)
	tree.Add(key, big.NewInt(1))

	var crlNum uint64 = 42

	var buf bytes.Buffer
	if err := Export(tree, crlNum, &buf); err != nil {
		t.Fatal("export:", err)
	}

	restored, gotCRL, err := Import(h, &buf)
	if err != nil {
		t.Fatal("import:", err)
	}

	if gotCRL != crlNum {
		t.Errorf("crlNumber: got %d, want %d", gotCRL, crlNum)
	}
	if restored.Root.Cmp(tree.Root) != 0 {
		t.Errorf("root mismatch")
	}
}
