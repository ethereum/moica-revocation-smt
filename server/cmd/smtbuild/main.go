package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/moven0831/moica-revocation-smt/server/internal/config"
	"github.com/moven0831/moica-revocation-smt/server/internal/crl"
	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
	"github.com/moven0831/moica-revocation-smt/server/internal/snapshot"
)

type issuer struct {
	ID  string
	URL string
}

type rootInfo struct {
	Root      string `json:"root"`
	Count     int    `json:"count"`
	CRLNumber string `json:"crlNumber"`
	Timestamp string `json:"timestamp"`
}

func main() {
	cfg := config.Load()

	issuers := []issuer{
		{ID: "g2", URL: cfg.CRLG2URL},
		{ID: "g3", URL: cfg.CRLG3URL},
	}

	hasher := smt.NewPoseidonHasher()
	entryVal := big.NewInt(1)
	anyChanged := false

	for _, iss := range issuers {
		log.Printf("[%s] Fetching CRL from %s", iss.ID, iss.URL)
		derBytes, err := crl.FetchDER(iss.URL)
		if err != nil {
			log.Printf("[%s] Skipping: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Fetched %d bytes", iss.ID, len(derBytes))

		parsed, err := crl.ParseDER(derBytes)
		if err != nil {
			log.Printf("[%s] Skipping: parse error: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Parsed %d revoked serials (CRLNumber=%s)",
			iss.ID, len(parsed.RevokedSerials), parsed.CRLNumber)

		// Deduplicate serials
		seen := make(map[string]struct{}, len(parsed.RevokedSerials))
		uniqueSerials := make([]*big.Int, 0, len(parsed.RevokedSerials))
		for _, s := range parsed.RevokedSerials {
			key := s.Text(16)
			if _, dup := seen[key]; !dup {
				seen[key] = struct{}{}
				uniqueSerials = append(uniqueSerials, s)
			}
		}
		log.Printf("[%s] %d unique serials (removed %d duplicates)",
			iss.ID, len(uniqueSerials), len(parsed.RevokedSerials)-len(uniqueSerials))

		// Build SMT with default depth 256 for wire-compatibility
		tree := smt.New(hasher)
		buildStart := time.Now()
		err = tree.BatchAddWithProgress(uniqueSerials, entryVal, 10000, func(done, total int) {
			log.Printf("[%s] Added %d / %d entries", iss.ID, done, total)
		})
		if err != nil {
			log.Printf("[%s] Skipping: batch add error: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] SMT built: count=%d, root=0x%s, duration=%v",
			iss.ID, tree.Count, tree.Root.Text(16), time.Since(buildStart))

		// Write output files
		issuerDir := filepath.Join(cfg.DataDir, iss.ID)
		if err := os.MkdirAll(issuerDir, 0o755); err != nil {
			log.Printf("[%s] Skipping: mkdir error: %v", iss.ID, err)
			continue
		}

		// Export snapshot
		snapshotPath := filepath.Join(issuerDir, "tree-snapshot.json.gz")
		f, err := os.Create(snapshotPath)
		if err != nil {
			log.Printf("[%s] Skipping: create snapshot file: %v", iss.ID, err)
			continue
		}
		if err := snapshot.Export(tree, f); err != nil {
			f.Close()
			log.Printf("[%s] Skipping: export snapshot: %v", iss.ID, err)
			continue
		}
		f.Close()
		log.Printf("[%s] Snapshot exported to %s", iss.ID, snapshotPath)

		// Write root.json
		info := rootInfo{
			Root:      "0x" + tree.Root.Text(16),
			Count:     tree.Count,
			CRLNumber: parsed.CRLNumber.String(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		rootJSON, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Printf("[%s] Skipping: marshal root.json: %v", iss.ID, err)
			continue
		}
		rootPath := filepath.Join(issuerDir, "root.json")
		if err := os.WriteFile(rootPath, rootJSON, 0o644); err != nil {
			log.Printf("[%s] Skipping: write root.json: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Root info written to %s", iss.ID, rootPath)

		anyChanged = true
	}

	// Write changed output for GitHub Actions
	if ghOutput := os.Getenv("GITHUB_OUTPUT"); ghOutput != "" {
		f, err := os.OpenFile(ghOutput, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("Failed to open GITHUB_OUTPUT: %v", err)
		}
		defer f.Close()
		fmt.Fprintf(f, "changed=%t\n", anyChanged)
	}

	if anyChanged {
		log.Println("Done — SMT data updated")
	} else {
		log.Println("Done — no changes")
	}
}
