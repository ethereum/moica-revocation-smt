package snapshot

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

// Snapshot represents the serialized form of an SMT.
// Compatible with the TS tree-snapshot.json.gz format.
type Snapshot struct {
	Version int              `json:"version"`
	Root    string           `json:"root"`
	Count   int              `json:"count"`
	Depth     int              `json:"depth,omitempty"`
	CRLNumber uint64           `json:"crlNumber"`
	Nodes     []NodeEntry      `json:"nodes"`
}

// NodeEntry is a [nodeHash, [children...]] pair.
type NodeEntry struct {
	Hash     string   `json:"hash"`
	Children []string `json:"children"`
}

// bigToHex converts a big.Int to "0x"-prefixed hex string.
func bigToHex(n *big.Int) string {
	if n == nil || n.Sign() == 0 {
		return "0x0"
	}
	return "0x" + n.Text(16)
}

// hexToBig converts a "0x"-prefixed hex string to big.Int.
func hexToBig(s string) (*big.Int, error) {
	if len(s) >= 2 && s[:2] == "0x" {
		s = s[2:]
	}
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex: %s", s)
	}
	return n, nil
}

// Export writes the SMT as a gzip-compressed JSON snapshot.
func Export(tree *smt.SMT, crlNumber uint64, w io.Writer) error {
	nodes := tree.Nodes()

	snapshot := Snapshot{
		Version:   1,
		Root:      bigToHex(tree.Root),
		Count:     tree.Count,
		Depth:     tree.Depth,
		CRLNumber: crlNumber,
		Nodes:     make([]NodeEntry, 0, len(nodes)),
	}

	for hash, children := range nodes {
		hashBig, _ := new(big.Int).SetString(hash, 16)
		entry := NodeEntry{
			Hash:     bigToHex(hashBig),
			Children: make([]string, len(children)),
		}
		for i, c := range children {
			entry.Children[i] = bigToHex(c)
		}
		snapshot.Nodes = append(snapshot.Nodes, entry)
	}

	gw := gzip.NewWriter(w)
	defer gw.Close()

	enc := json.NewEncoder(gw)
	return enc.Encode(snapshot)
}

// ImportFile opens a gzip-compressed JSON snapshot file and reconstructs an SMT.
func ImportFile(h smt.Hasher, path string) (*smt.SMT, uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	return Import(h, f)
}

// Import reads a gzip-compressed JSON snapshot and reconstructs an SMT.
func Import(h smt.Hasher, r io.Reader) (*smt.SMT, uint64, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, 0, fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	var snapshot Snapshot
	if err := json.NewDecoder(gr).Decode(&snapshot); err != nil {
		return nil, 0, fmt.Errorf("json decode: %w", err)
	}

	root, err := hexToBig(snapshot.Root)
	if err != nil {
		return nil, 0, fmt.Errorf("parse root: %w", err)
	}

	depth := snapshot.Depth
	if depth == 0 {
		depth = smt.DefaultDepth
	}

	nodes := make(map[string]smt.ChildNodes, len(snapshot.Nodes))
	for _, entry := range snapshot.Nodes {
		hashBig, err := hexToBig(entry.Hash)
		if err != nil {
			return nil, 0, fmt.Errorf("parse node hash: %w", err)
		}

		children := make(smt.ChildNodes, len(entry.Children))
		for i, c := range entry.Children {
			children[i], err = hexToBig(c)
			if err != nil {
				return nil, 0, fmt.Errorf("parse child: %w", err)
			}
		}

		key := hashBig.Text(16)
		nodes[key] = children
	}

	tree := smt.NewWithDepth(h, depth)
	tree.SetNodes(nodes, root, snapshot.Count)

	return tree, snapshot.CRLNumber, nil
}
