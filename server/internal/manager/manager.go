package manager

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

// TreeEntry holds a single issuer's SMT and metadata.
type TreeEntry struct {
	Tree      *smt.SMT
	CRLNumber uint64
	LoadedAt  time.Time
}

// TreeManager holds per-issuer SMTs with thread-safe access.
type TreeManager struct {
	mu      sync.RWMutex
	trees   map[string]*TreeEntry
	hasher  smt.Hasher
}

// New creates a TreeManager with the given hasher.
func New(h smt.Hasher) *TreeManager {
	return &TreeManager{
		trees:  make(map[string]*TreeEntry),
		hasher: h,
	}
}

// Hasher returns the hasher used by this manager.
func (m *TreeManager) Hasher() smt.Hasher {
	return m.hasher
}

// GetTree returns the tree entry for the given issuer ID.
func (m *TreeManager) GetTree(issuerID string) (*TreeEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.trees[issuerID]
	if !ok {
		return nil, fmt.Errorf("unknown issuer: %s", issuerID)
	}
	return entry, nil
}

// SetTree atomically replaces the tree for an issuer ID.
func (m *TreeManager) SetTree(issuerID string, tree *smt.SMT, crlNumber uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.trees[issuerID] = &TreeEntry{
		Tree:      tree,
		CRLNumber: crlNumber,
		LoadedAt:  time.Now(),
	}
}

// GetProof generates a proof for the given issuer and serial number.
func (m *TreeManager) GetProof(issuerID string, serialNumber *big.Int) (*smt.MerkleProof, error) {
	m.mu.RLock()
	entry, ok := m.trees[issuerID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown issuer: %s", issuerID)
	}

	return entry.Tree.CreateProof(serialNumber), nil
}

// IssuerIDs returns a list of all loaded issuer IDs.
func (m *TreeManager) IssuerIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.trees))
	for id := range m.trees {
		ids = append(ids, id)
	}
	return ids
}

// Status returns status info for all loaded issuers.
type IssuerStatus struct {
	Loaded    bool   `json:"loaded"`
	Count     int    `json:"count"`
	Root      string `json:"root"`
	CRLNumber uint64 `json:"crlNumber"`
	LoadedAt  string `json:"loadedAt"`
}

func (m *TreeManager) Status() map[string]IssuerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]IssuerStatus)
	for id, entry := range m.trees {
		root := "0x0"
		if entry.Tree.Root != nil && entry.Tree.Root.Sign() != 0 {
			root = "0x" + entry.Tree.Root.Text(16)
		}
		status[id] = IssuerStatus{
			Loaded:    true,
			Count:     entry.Tree.Count,
			Root:      root,
			CRLNumber: entry.CRLNumber,
			LoadedAt:  entry.LoadedAt.UTC().Format(time.RFC3339),
		}
	}
	return status
}
