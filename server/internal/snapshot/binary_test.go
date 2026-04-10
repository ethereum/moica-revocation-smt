package snapshot

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

func buildTestTree(t *testing.T) *smt.SMT {
	t.Helper()
	h := smt.NewPoseidonHasher()
	tree := smt.New(h)

	serials := []string{
		"100048210DD2DF2E128096A9282B5EC5",
		"200048210DD2DF2E128096A9282B5EC5",
		"300048210DD2DF2E128096A9282B5EC5",
	}
	for _, s := range serials {
		key, _ := new(big.Int).SetString(s, 16)
		if err := tree.Add(key, big.NewInt(1)); err != nil {
			t.Fatalf("add %s: %v", s, err)
		}
	}
	return tree
}

func TestBinaryRoundTrip(t *testing.T) {
	h := smt.NewPoseidonHasher()
	tree := buildTestTree(t)

	originalRoot := new(big.Int).Set(tree.Root)
	originalCount := tree.Count
	var crlNum uint64 = 42

	// Export
	var buf bytes.Buffer
	if err := ExportBinary(tree, crlNum, &buf); err != nil {
		t.Fatal("export:", err)
	}

	// Import
	restored, gotCRL, err := ImportBinary(h, &buf)
	if err != nil {
		t.Fatal("import:", err)
	}

	// Verify metadata
	if restored.Root.Cmp(originalRoot) != 0 {
		t.Errorf("root mismatch: got %s, want %s", restored.Root.Text(16), originalRoot.Text(16))
	}
	if restored.Count != originalCount {
		t.Errorf("count mismatch: got %d, want %d", restored.Count, originalCount)
	}
	if gotCRL != crlNum {
		t.Errorf("crlNumber: got %d, want %d", gotCRL, crlNum)
	}

	// Verify membership proof
	key, _ := new(big.Int).SetString("100048210DD2DF2E128096A9282B5EC5", 16)
	proof := restored.CreateProof(key)
	if !proof.Membership {
		t.Fatal("membership proof failed on restored tree")
	}
	if !smt.VerifyProof(h, proof, smt.DefaultDepth) {
		t.Fatal("proof verification failed on restored tree")
	}

	// Verify non-membership proof
	nonMember, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)
	nonProof := restored.CreateProof(nonMember)
	if nonProof.Membership {
		t.Fatal("non-member should not be member")
	}
	if !smt.VerifyProof(h, nonProof, smt.DefaultDepth) {
		t.Fatal("non-membership proof verification failed")
	}
}

func TestBinaryEmptyTree(t *testing.T) {
	h := smt.NewPoseidonHasher()
	tree := smt.New(h)

	var buf bytes.Buffer
	if err := ExportBinary(tree, 0, &buf); err != nil {
		t.Fatal("export empty:", err)
	}

	restored, _, err := ImportBinary(h, &buf)
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

func TestBinaryCrossFormat(t *testing.T) {
	h := smt.NewPoseidonHasher()
	tree := buildTestTree(t)

	// Export as JSON
	var jsonBuf bytes.Buffer
	if err := Export(tree, 99, &jsonBuf); err != nil {
		t.Fatal("json export:", err)
	}

	// Import from JSON
	jsonTree, _, err := Import(h, &jsonBuf)
	if err != nil {
		t.Fatal("json import:", err)
	}

	// Export as binary from the JSON-imported tree
	var binBuf bytes.Buffer
	if err := ExportBinary(jsonTree, 99, &binBuf); err != nil {
		t.Fatal("binary export:", err)
	}

	// Import from binary
	binTree, _, err := ImportBinary(h, &binBuf)
	if err != nil {
		t.Fatal("binary import:", err)
	}

	// Roots must match
	if jsonTree.Root.Cmp(binTree.Root) != 0 {
		t.Errorf("root mismatch: json=%s, binary=%s",
			jsonTree.Root.Text(16), binTree.Root.Text(16))
	}

	// Proof from binary-imported tree must match
	key, _ := new(big.Int).SetString("200048210DD2DF2E128096A9282B5EC5", 16)
	jsonProof := jsonTree.CreateProof(key)
	binProof := binTree.CreateProof(key)

	if jsonProof.Membership != binProof.Membership {
		t.Error("membership flag mismatch between formats")
	}
	if len(jsonProof.Siblings) != len(binProof.Siblings) {
		t.Errorf("siblings length mismatch: json=%d, binary=%d",
			len(jsonProof.Siblings), len(binProof.Siblings))
	}
}

func TestBinaryTruncated(t *testing.T) {
	h := smt.NewPoseidonHasher()
	tree := buildTestTree(t)

	var buf bytes.Buffer
	if err := ExportBinary(tree, 0, &buf); err != nil {
		t.Fatal("export:", err)
	}

	// Truncate to just the header + 1 byte (not enough for a full node)
	truncated := buf.Bytes()[:BinaryHeader+1]
	_, _, err := ImportBinary(h, bytes.NewReader(truncated))
	if err == nil {
		t.Fatal("expected error on truncated binary")
	}
}

func TestBinaryInvalidMagic(t *testing.T) {
	h := smt.NewPoseidonHasher()

	var buf [BinaryHeader]byte
	binary.BigEndian.PutUint16(buf[0:2], 0xDEAD) // wrong magic
	binary.BigEndian.PutUint16(buf[2:4], BinaryVersion)

	_, _, err := ImportBinary(h, bytes.NewReader(buf[:]))
	if err == nil {
		t.Fatal("expected error on invalid magic")
	}
}

func TestBinaryUnknownVersion(t *testing.T) {
	h := smt.NewPoseidonHasher()

	var buf [BinaryHeader]byte
	binary.BigEndian.PutUint16(buf[0:2], BinaryMagic)
	binary.BigEndian.PutUint16(buf[2:4], 99) // unsupported version

	_, _, err := ImportBinary(h, bytes.NewReader(buf[:]))
	if err == nil {
		t.Fatal("expected error on unknown version")
	}
}

func TestBinaryFileRoundTrip(t *testing.T) {
	h := smt.NewPoseidonHasher()
	tree := buildTestTree(t)

	dir := t.TempDir()

	// Test uncompressed
	rawPath := filepath.Join(dir, "test.bin")
	if err := ExportBinaryFile(tree, 42, rawPath); err != nil {
		t.Fatal("ExportBinaryFile:", err)
	}
	restored, crl, err := ImportBinaryFile(h, rawPath)
	if err != nil {
		t.Fatal("ImportBinaryFile:", err)
	}
	if restored.Root.Cmp(tree.Root) != 0 {
		t.Error("root mismatch (uncompressed)")
	}
	if crl != 42 {
		t.Errorf("crl: got %d, want 42", crl)
	}

	// Test gzip-compressed
	gzPath := filepath.Join(dir, "test.bin.gz")
	if err := ExportBinaryFile(tree, 42, gzPath); err != nil {
		t.Fatal("ExportBinaryFile gz:", err)
	}
	restored2, _, err := ImportBinaryFile(h, gzPath)
	if err != nil {
		t.Fatal("ImportBinaryFile gz:", err)
	}
	if restored2.Root.Cmp(tree.Root) != 0 {
		t.Error("root mismatch (compressed)")
	}

	// Verify no tmp files remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("stale temp file: %s", e.Name())
		}
	}
}
