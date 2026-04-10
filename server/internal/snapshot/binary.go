package snapshot

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

// Binary format constants.
const (
	BinaryMagic   = 0x534D // "SM"
	BinaryVersion = 1
	BinaryHeader  = 52 // 2+2+4+32+4+8 bytes
)

// ExportBinary writes the SMT as an uncompressed binary snapshot.
// Caller is responsible for gzip wrapping if desired.
//
// Format:
//
//	HEADER (52 bytes):
//	  [0:2]   magic       uint16  0x534D
//	  [2:4]   version     uint16  1
//	  [4:8]   nodeCount   uint32
//	  [8:40]  rootHash    [32]byte
//	  [40:44] depth       uint32
//	  [44:52] crlNumber   uint64
//
//	PER NODE:
//	  [0:1]   type        uint8   0=branch, 1=leaf
//	  [1:33]  hash        [32]byte
//	  Branch: [33:65] left [32]byte, [65:97] right [32]byte
//	  Leaf:   [33:65] key [32]byte, [65:97] value [32]byte, [97:129] entryMark [32]byte
func ExportBinary(tree *smt.SMT, crlNumber uint64, w io.Writer) error {
	nodes := tree.NodesRaw()

	// Write header
	var hdr [BinaryHeader]byte
	binary.BigEndian.PutUint16(hdr[0:2], BinaryMagic)
	binary.BigEndian.PutUint16(hdr[2:4], BinaryVersion)
	binary.BigEndian.PutUint32(hdr[4:8], uint32(len(nodes)))
	root32 := bigTo32Bytes(tree.Root)
	copy(hdr[8:40], root32[:])
	binary.BigEndian.PutUint32(hdr[40:44], uint32(tree.Depth))
	binary.BigEndian.PutUint64(hdr[44:52], crlNumber)

	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write nodes
	for _, rn := range nodes {
		var typ byte
		if rn.IsLeaf {
			typ = 1
		}
		if err := writeByte(w, typ); err != nil {
			return fmt.Errorf("write node type: %w", err)
		}
		if _, err := w.Write(rn.Hash[:]); err != nil {
			return fmt.Errorf("write node hash: %w", err)
		}
		for _, child := range rn.Children {
			if _, err := w.Write(child[:]); err != nil {
				return fmt.Errorf("write child: %w", err)
			}
		}
	}

	return nil
}

// ImportBinary reads an uncompressed binary snapshot and reconstructs an SMT.
// Caller is responsible for gzip decompression if needed.
func ImportBinary(h smt.Hasher, r io.Reader) (*smt.SMT, uint64, error) {
	// Read header
	var hdr [BinaryHeader]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, 0, fmt.Errorf("read header: %w", err)
	}

	magic := binary.BigEndian.Uint16(hdr[0:2])
	if magic != BinaryMagic {
		return nil, 0, fmt.Errorf("invalid magic: 0x%04X (expected 0x%04X)", magic, BinaryMagic)
	}

	version := binary.BigEndian.Uint16(hdr[2:4])
	if version != BinaryVersion {
		return nil, 0, fmt.Errorf("unsupported version: %d (expected %d)", version, BinaryVersion)
	}

	nodeCount := binary.BigEndian.Uint32(hdr[4:8])
	var rootHash [32]byte
	copy(rootHash[:], hdr[8:40])
	root := new(big.Int).SetBytes(rootHash[:])

	depth := binary.BigEndian.Uint32(hdr[40:44])
	crlNumber := binary.BigEndian.Uint64(hdr[44:52])

	// Read nodes
	nodes := make(map[string]smt.ChildNodes, nodeCount)
	count := 0

	var typeBuf [1]byte
	var hashBuf [32]byte
	var childBuf [32]byte

	for i := uint32(0); i < nodeCount; i++ {
		if _, err := io.ReadFull(r, typeBuf[:]); err != nil {
			return nil, 0, fmt.Errorf("read node %d type: %w (expected %d nodes)", i, err, nodeCount)
		}

		if _, err := io.ReadFull(r, hashBuf[:]); err != nil {
			return nil, 0, fmt.Errorf("read node %d hash: %w", i, err)
		}

		isLeaf := typeBuf[0] == 1
		numChildren := 2
		if isLeaf {
			numChildren = 3
		}

		children := make(smt.ChildNodes, numChildren)
		for j := 0; j < numChildren; j++ {
			if _, err := io.ReadFull(r, childBuf[:]); err != nil {
				return nil, 0, fmt.Errorf("read node %d child %d: %w", i, j, err)
			}
			children[j] = new(big.Int).SetBytes(childBuf[:])
		}

		hashKey := new(big.Int).SetBytes(hashBuf[:]).Text(16)
		nodes[hashKey] = children

		if isLeaf {
			count++
		}
	}

	tree := smt.NewWithDepth(h, int(depth))
	tree.SetNodes(nodes, root, count)

	return tree, crlNumber, nil
}

// ExportBinaryFile atomically writes a binary snapshot to the given path.
// If the path ends with ".gz", the output is gzip-compressed.
func ExportBinaryFile(tree *smt.SMT, crlNumber uint64, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.CreateTemp(filepath.Dir(path), "snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := f.Name()

	var w io.Writer = f
	var gw *gzip.Writer
	if strings.HasSuffix(path, ".gz") {
		gw = gzip.NewWriter(f)
		w = gw
	}

	if err := ExportBinary(tree, crlNumber, w); err != nil {
		if gw != nil {
			gw.Close()
		}
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("export: %w", err)
	}
	if gw != nil {
		if err := gw.Close(); err != nil {
			f.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("gzip close: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// ImportBinaryFile reads a binary snapshot from the given path.
// If the path ends with ".gz", the input is gzip-decompressed.
func ImportBinaryFile(h smt.Hasher, path string) (*smt.SMT, uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return nil, 0, fmt.Errorf("gzip reader: %w", err)
		}
		defer gr.Close()
		r = gr
	}

	return ImportBinary(h, r)
}

// bigTo32Bytes converts a big.Int to a zero-padded 32-byte big-endian array.
func bigTo32Bytes(n *big.Int) [32]byte {
	var buf [32]byte
	if n != nil && n.Sign() > 0 {
		n.FillBytes(buf[:])
	}
	return buf
}

func writeByte(w io.Writer, b byte) error {
	_, err := w.Write([]byte{b})
	return err
}
