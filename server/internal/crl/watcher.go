package crl

import (
	"context"
	"log"
	"math/big"
	"path/filepath"
	"time"

	"github.com/moven0831/moica-revocation-smt/server/internal/manager"
	"github.com/moven0831/moica-revocation-smt/server/internal/snapshot"
	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

// IssuerConfig holds the configuration for one issuer's CRL.
type IssuerConfig struct {
	ID  string
	URL string
}

// Watcher periodically fetches CRLs and rebuilds SMTs.
type Watcher struct {
	interval time.Duration
	issuers  []IssuerConfig
	mgr      *manager.TreeManager
	hasher   smt.Hasher
	dataDir  string
}

// NewWatcher creates a CRL watcher that polls at the given interval.
func NewWatcher(interval time.Duration, issuers []IssuerConfig, mgr *manager.TreeManager, hasher smt.Hasher, dataDir string) *Watcher {
	return &Watcher{
		interval: interval,
		issuers:  issuers,
		mgr:      mgr,
		hasher:   hasher,
		dataDir:  dataDir,
	}
}

// Start begins the periodic CRL polling loop. Blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	// Initial fetch
	w.fetchAll()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.fetchAll()
		}
	}
}

func (w *Watcher) fetchAll() {
	for _, issuer := range w.issuers {
		if err := w.fetchAndRebuild(issuer); err != nil {
			log.Printf("CRL fetch error for %s: %v", issuer.ID, err)
		}
	}
}

func (w *Watcher) fetchAndRebuild(issuer IssuerConfig) error {
	log.Printf("Fetching CRL for %s from %s", issuer.ID, issuer.URL)

	derBytes, err := FetchDER(issuer.URL)
	if err != nil {
		return err
	}

	parsed, err := ParseDER(derBytes)
	if err != nil {
		return err
	}

	// Check if CRL number is newer than what we have
	existing, _ := w.mgr.GetTree(issuer.ID)
	if existing != nil && parsed.CRLNumber != nil {
		if parsed.CRLNumber.Uint64() <= existing.CRLNumber {
			log.Printf("CRL for %s is not newer (have %d, got %d), skipping",
				issuer.ID, existing.CRLNumber, parsed.CRLNumber.Uint64())
			return nil
		}
	}

	log.Printf("Building SMT for %s with %d revoked serials", issuer.ID, len(parsed.RevokedSerials))

	tree := smt.New(w.hasher)
	for _, serial := range parsed.RevokedSerials {
		if err := tree.Add(serial, big.NewInt(1)); err != nil {
			// Skip duplicates
			continue
		}
	}

	var crlNum uint64
	if parsed.CRLNumber != nil {
		crlNum = parsed.CRLNumber.Uint64()
	}

	w.mgr.SetTree(issuer.ID, tree, crlNum)
	log.Printf("SMT for %s loaded: root=%s count=%d crl=%d",
		issuer.ID, tree.Root.Text(16)[:16]+"...", tree.Count, crlNum)

	// Persist snapshot to disk so future restarts load fresh data instantly
	go func() {
		snapPath := filepath.Join(w.dataDir, issuer.ID, "tree-snapshot.json.gz")
		if err := snapshot.ExportFile(tree, crlNum, snapPath); err != nil {
			log.Printf("Snapshot export failed for %s: %v", issuer.ID, err)
		} else {
			log.Printf("Snapshot exported for %s to %s", issuer.ID, snapPath)
		}
	}()

	return nil
}
