package smt

import (
	"fmt"
	"math/big"
	"sync"
)

var (
	zero     = big.NewInt(0)
	one      = big.NewInt(1)
	entryVal = big.NewInt(1) // All entries use value=1 as membership marker
)

// SMT is a Sparse Merkle Tree compatible with @zk-kit/smt v1.0.2 (bigNumbers mode).
type SMT struct {
	mu    sync.RWMutex
	hash  Hasher
	nodes map[string]ChildNodes // nodeHash (hex) → child nodes
	Root  *big.Int
	Count int
	Depth int // Tree depth (number of bits in key path)
}

// New creates a new empty SMT with the given hasher and default depth (128).
func New(h Hasher) *SMT {
	return NewWithDepth(h, DefaultDepth)
}

// NewWithDepth creates a new empty SMT with the given hasher and depth.
func NewWithDepth(h Hasher, depth int) *SMT {
	return &SMT{
		hash:  h,
		nodes: make(map[string]ChildNodes),
		Root:  new(big.Int),
		Depth: depth,
	}
}

// growNodes pre-sizes the internal nodes map if needed.
// Each entry adds ~1 leaf + several branch nodes; estimate ~3x entries for safety.
func (t *SMT) growNodes(additionalEntries int) {
	needed := len(t.nodes) + additionalEntries*3
	if needed > len(t.nodes) {
		grown := make(map[string]ChildNodes, needed)
		for k, v := range t.nodes {
			grown[k] = v
		}
		t.nodes = grown
	}
}

// nodeKey returns the map key for a big.Int node hash.
func nodeKey(n *big.Int) string {
	return n.Text(16)
}

// isZero checks if a node is the zero node.
func isZero(n *big.Int) bool {
	return n.Sign() == 0
}

// Get returns the value for a key, or nil if the key doesn't exist.
func (t *SMT) Get(key *big.Int) *big.Int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entry, _, _ := t.retrieveEntry(key)
	if len(entry) >= 2 && entry[1] != nil {
		return new(big.Int).Set(entry[1])
	}
	return nil
}

// Add inserts a new key/value entry into the tree.
func (t *SMT) Add(key, value *big.Int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.addUnlocked(key, value)
}

// addUnlocked is the lock-free core of Add, for use by batch methods that hold the lock.
func (t *SMT) addUnlocked(key, value *big.Int) error {
	if key.BitLen() > t.Depth {
		return fmt.Errorf("key exceeds tree depth: %d bits > %d", key.BitLen(), t.Depth)
	}

	entry, matchingEntry, siblings := t.retrieveEntry(key)

	if len(entry) >= 2 && entry[1] != nil {
		return fmt.Errorf("key %s already exists", key.Text(16))
	}

	path := KeyToPath(key, t.Depth)

	// If there is a matching entry, its node hash is computed; otherwise zero.
	node := new(big.Int)
	if matchingEntry != nil {
		node = t.hash.Hash3(matchingEntry[0], matchingEntry[1], matchingEntry[2])
	}

	// Delete old nodes along the path.
	if len(siblings) > 0 {
		t.deleteOldNodes(node, path, siblings)
	}

	// If there is a matching entry, add zero siblings for common bits, then the matching node.
	if matchingEntry != nil {
		matchingPath := KeyToPath(matchingEntry[0], t.Depth)
		for i := len(siblings); matchingPath[i] == path[i]; i++ {
			siblings = append(siblings, new(big.Int))
		}
		siblings = append(siblings, new(big.Int).Set(node))
	}

	// Add the new leaf entry.
	newNode := t.hash.Hash3(key, value, one)
	t.nodes[nodeKey(newNode)] = ChildNodes{
		new(big.Int).Set(key),
		new(big.Int).Set(value),
		new(big.Int).Set(one),
	}

	t.Root = t.addNewNodes(newNode, path, siblings, len(siblings)-1)
	t.Count++
	return nil
}

// Delete removes an entry from the tree.
func (t *SMT) Delete(key *big.Int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.deleteUnlocked(key)
}

// deleteUnlocked is the lock-free core of Delete, for use by batch methods that hold the lock.
func (t *SMT) deleteUnlocked(key *big.Int) error {
	entry, _, siblings := t.retrieveEntry(key)

	if len(entry) < 2 || entry[1] == nil {
		return fmt.Errorf("key %s does not exist", key.Text(16))
	}

	path := KeyToPath(key, t.Depth)

	// Delete the entry node.
	node := t.hash.Hash3(entry[0], entry[1], entry[2])
	delete(t.nodes, nodeKey(node))

	t.Root = new(big.Int)

	if len(siblings) > 0 {
		t.deleteOldNodes(node, path, siblings)

		lastSibling := siblings[len(siblings)-1]
		if !t.isLeaf(lastSibling) {
			t.Root = t.addNewNodes(new(big.Int), path, siblings, len(siblings)-1)
		} else {
			// Pop last sibling
			firstSibling := new(big.Int).Set(lastSibling)
			siblings = siblings[:len(siblings)-1]
			i := getIndexOfLastNonZero(siblings)
			t.Root = t.addNewNodes(firstSibling, path, siblings, i)
		}
	}

	t.Count--
	return nil
}

// Nodes returns the internal nodes map (for snapshot export).
func (t *SMT) Nodes() map[string]ChildNodes {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.nodes
}

// Keys returns copies of all leaf keys (serial numbers) in the tree.
func (t *SMT) Keys() []*big.Int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	keys := make([]*big.Int, 0, t.Count)
	for _, cn := range t.nodes {
		if cn.IsLeaf() {
			keys = append(keys, new(big.Int).Set(cn[0]))
		}
	}
	return keys
}

// SetNodes restores internal state (for snapshot import).
func (t *SMT) SetNodes(nodes map[string]ChildNodes, root *big.Int, count int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes = nodes
	t.Root = root
	t.Count = count
}

// retrieveEntry searches for an entry in the tree.
// Returns (entry, matchingEntry, siblings).
func (t *SMT) retrieveEntry(key *big.Int) (ChildNodes, ChildNodes, []*big.Int) {
	path := KeyToPath(key, t.Depth)
	var siblings []*big.Int

	node := new(big.Int).Set(t.Root)

	for i := 0; !isZero(node); i++ {
		childNodes, ok := t.nodes[nodeKey(node)]
		if !ok {
			break
		}

		// Leaf node check: 3rd element present
		if childNodes.IsLeaf() {
			if childNodes[0].Cmp(key) == 0 {
				// Found exact match
				return childNodes, nil, siblings
			}
			// Matching entry (different key but shares path prefix)
			return ChildNodes{new(big.Int).Set(key)}, childNodes, siblings
		}

		// Branch node: follow the path direction
		direction := path[i]
		node = new(big.Int).Set(childNodes[direction])
		siblings = append(siblings, new(big.Int).Set(childNodes[1-direction]))
	}

	// Reached zero node
	return ChildNodes{new(big.Int).Set(key)}, nil, siblings
}

// addNewNodes creates new branch nodes bottom-up and stores them.
func (t *SMT) addNewNodes(node *big.Int, path []byte, siblings []*big.Int, i int) *big.Int {
	n := new(big.Int).Set(node)
	for ; i >= 0; i-- {
		var childNodes ChildNodes
		if path[i] == 1 {
			childNodes = ChildNodes{new(big.Int).Set(siblings[i]), new(big.Int).Set(n)}
		} else {
			childNodes = ChildNodes{new(big.Int).Set(n), new(big.Int).Set(siblings[i])}
		}
		n = t.hash.Hash2(childNodes[0], childNodes[1])
		t.nodes[nodeKey(n)] = childNodes
	}
	return n
}

// deleteOldNodes removes branch nodes bottom-up.
func (t *SMT) deleteOldNodes(node *big.Int, path []byte, siblings []*big.Int) {
	n := new(big.Int).Set(node)
	for i := len(siblings) - 1; i >= 0; i-- {
		var left, right *big.Int
		if path[i] == 1 {
			left, right = siblings[i], n
		} else {
			left, right = n, siblings[i]
		}
		n = t.hash.Hash2(left, right)
		delete(t.nodes, nodeKey(n))
	}
}

// isLeaf checks if a node hash corresponds to a leaf node.
func (t *SMT) isLeaf(node *big.Int) bool {
	cn, ok := t.nodes[nodeKey(node)]
	return ok && cn.IsLeaf()
}

// getIndexOfLastNonZero returns the index of the last non-zero element.
// Returns -1 if all elements are zero.
func getIndexOfLastNonZero(arr []*big.Int) int {
	for i := len(arr) - 1; i >= 0; i-- {
		if arr[i].Sign() != 0 {
			return i
		}
	}
	return -1
}
