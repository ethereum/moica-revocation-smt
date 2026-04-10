//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"runtime"
	"syscall/js"

	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

var (
	hasher smt.Hasher
	tree   *smt.SMT
	nodes  map[string]smt.ChildNodes
)

func main() {
	hasher = smt.NewPoseidonHasher()

	js.Global().Set("smtInitTree", js.FuncOf(initTree))
	js.Global().Set("smtAddNodeChunk", js.FuncOf(addNodeChunk))
	js.Global().Set("smtFinalize", js.FuncOf(finalize))
	js.Global().Set("smtCreateProof", js.FuncOf(createProof))
	js.Global().Set("smtVerifyProof", js.FuncOf(verifyProof))
	js.Global().Set("smtGetMemStats", js.FuncOf(getMemStats))
	js.Global().Set("smtReady", js.ValueOf(true))

	// Block forever
	select {}
}

// initTree(nodeCount, depth) — pre-allocates the node map.
func initTree(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return jsError("initTree requires (nodeCount, depth)")
	}
	nodeCount := args[0].Int()
	depth := args[1].Int()

	tree = smt.NewWithDepth(hasher, depth)
	nodes = make(map[string]smt.ChildNodes, nodeCount)
	return nil
}

// addNodeChunk(uint8Array) — receives a chunk of serialized binary nodes.
// Each node: 1-byte type + 32-byte hash + children (branch: 2x32, leaf: 3x32).
// Returns the number of nodes parsed from this chunk.
func addNodeChunk(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsError("addNodeChunk requires (uint8Array)")
	}
	if nodes == nil {
		return jsError("call smtInitTree before smtAddNodeChunk")
	}

	jsArr := args[0]
	length := jsArr.Get("length").Int()
	buf := make([]byte, length)
	js.CopyBytesToGo(buf, jsArr)

	parsed := 0
	offset := 0

	for offset < length {
		if offset+33 > length {
			break // not enough data for type + hash
		}

		isLeaf := buf[offset] == 1
		offset++

		var hashBuf [32]byte
		copy(hashBuf[:], buf[offset:offset+32])
		offset += 32

		numChildren := 2
		if isLeaf {
			numChildren = 3
		}

		childrenSize := numChildren * 32
		if offset+childrenSize > length {
			break // not enough data for children
		}

		children := make(smt.ChildNodes, numChildren)
		for j := 0; j < numChildren; j++ {
			children[j] = new(big.Int).SetBytes(buf[offset : offset+32])
			offset += 32
		}

		hashKey := new(big.Int).SetBytes(hashBuf[:]).Text(16)
		nodes[hashKey] = children
		parsed++
	}

	return parsed
}

// finalize(rootHex, count) — sets the root and count, making the tree ready for proofs.
func finalize(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return jsError("finalize requires (rootHex, count)")
	}
	if tree == nil {
		return jsError("call smtInitTree before smtFinalize")
	}

	rootHex := args[0].String()
	count := args[1].Int()

	root, ok := new(big.Int).SetString(rootHex, 16)
	if !ok {
		return jsError(fmt.Sprintf("invalid root hex: %s", rootHex))
	}

	tree.SetNodes(nodes, root, count)
	// Allow GC of the temporary nodes map reference
	nodes = nil

	return nil
}

// createProof(keyHex) — generates a proof and returns it as JSON.
func createProof(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsError("createProof requires (keyHex)")
	}
	if tree == nil {
		return jsError("tree not initialized — call smtInitTree and smtFinalize first")
	}

	keyHex := args[0].String()
	key, ok := new(big.Int).SetString(keyHex, 16)
	if !ok {
		return jsError(fmt.Sprintf("invalid key hex: %s", keyHex))
	}

	proof := tree.CreateProof(key)
	result := proofToJSON(proof)

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return jsError(fmt.Sprintf("marshal proof: %v", err))
	}
	return string(jsonBytes)
}

// verifyProof(proofJSON) — verifies a proof and returns a boolean.
func verifyProof(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return jsError("verifyProof requires (proofJSON)")
	}
	if tree == nil {
		return jsError("tree not initialized — call smtInitTree and smtFinalize first")
	}

	proofJSON := args[0].String()
	var jp jsonProof
	if err := json.Unmarshal([]byte(proofJSON), &jp); err != nil {
		return jsError(fmt.Sprintf("unmarshal proof: %v", err))
	}

	proof := jsonToProof(&jp)
	return smt.VerifyProof(hasher, proof, tree.Depth)
}

// getMemStats() — returns Go runtime memory statistics as JSON.
func getMemStats(_ js.Value, _ []js.Value) any {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	result := map[string]any{
		"alloc":      m.Alloc,
		"totalAlloc": m.TotalAlloc,
		"sys":        m.Sys,
		"heapInuse":  m.HeapInuse,
		"heapAlloc":  m.HeapAlloc,
		"numGC":      m.NumGC,
	}

	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

func jsError(msg string) any {
	return js.Global().Get("Error").New(msg)
}

// JSON proof serialization types
type jsonProof struct {
	Entry         []string `json:"entry"`
	MatchingEntry []string `json:"matchingEntry,omitempty"`
	Siblings      []string `json:"siblings"`
	Root          string   `json:"root"`
	Membership    bool     `json:"membership"`
}

func bigToHex(n *big.Int) string {
	if n == nil || n.Sign() == 0 {
		return "0"
	}
	return n.Text(16)
}

func hexToBig(s string) *big.Int {
	n, _ := new(big.Int).SetString(s, 16)
	if n == nil {
		return new(big.Int)
	}
	return n
}

func proofToJSON(p *smt.MerkleProof) *jsonProof {
	jp := &jsonProof{
		Entry:      make([]string, len(p.Entry)),
		Siblings:   make([]string, len(p.Siblings)),
		Root:       bigToHex(p.Root),
		Membership: p.Membership,
	}
	for i, v := range p.Entry {
		jp.Entry[i] = bigToHex(v)
	}
	for i, v := range p.Siblings {
		jp.Siblings[i] = bigToHex(v)
	}
	if p.MatchingEntry != nil {
		jp.MatchingEntry = make([]string, len(p.MatchingEntry))
		for i, v := range p.MatchingEntry {
			jp.MatchingEntry[i] = bigToHex(v)
		}
	}
	return jp
}

func jsonToProof(jp *jsonProof) *smt.MerkleProof {
	p := &smt.MerkleProof{
		Entry:      make([]*big.Int, len(jp.Entry)),
		Siblings:   make([]*big.Int, len(jp.Siblings)),
		Root:       hexToBig(jp.Root),
		Membership: jp.Membership,
	}
	for i, s := range jp.Entry {
		p.Entry[i] = hexToBig(s)
	}
	for i, s := range jp.Siblings {
		p.Siblings[i] = hexToBig(s)
	}
	if jp.MatchingEntry != nil {
		p.MatchingEntry = make([]*big.Int, len(jp.MatchingEntry))
		for i, s := range jp.MatchingEntry {
			p.MatchingEntry[i] = hexToBig(s)
		}
	}
	return p
}
