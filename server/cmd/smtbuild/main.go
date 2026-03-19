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

		// Try loading existing snapshot for incremental update
		var tree *smt.SMT
		buildStart := time.Now()
		issuerDir := filepath.Join(cfg.DataDir, iss.ID)
		snapshotPath := filepath.Join(issuerDir, "tree-snapshot.json.gz")

		tree, _, err = snapshot.ImportFile(hasher, snapshotPath)
		if err != nil {
			log.Printf("[%s] No local snapshot, trying GitHub Release", iss.ID)
			if dlPath, dlErr := snapshot.Download(cfg.GitHubRepo, iss.ID, cfg.DataDir); dlErr == nil {
				tree, _, err = snapshot.ImportFile(hasher, dlPath)
				if err != nil {
					log.Printf("[%s] Failed to import downloaded snapshot: %v", iss.ID, err)
				}
			} else {
				log.Printf("[%s] No snapshot available: %v", iss.ID, dlErr)
			}
		}

		if tree != nil && tree.Count > 0 {
			// Incremental update: compute delta
			existingKeys := tree.Keys()
			existingSet := make(map[string]struct{}, len(existingKeys))
			for _, k := range existingKeys {
				existingSet[k.Text(16)] = struct{}{}
			}

			newSet := make(map[string]struct{}, len(uniqueSerials))
			for _, s := range uniqueSerials {
				newSet[s.Text(16)] = struct{}{}
			}

			// Compute toAdd: in new but not existing
			var toAdd []*big.Int
			for _, s := range uniqueSerials {
				if _, ok := existingSet[s.Text(16)]; !ok {
					toAdd = append(toAdd, s)
				}
			}

			// Compute toDelete: in existing but not new
			var toDelete []*big.Int
			for _, k := range existingKeys {
				if _, ok := newSet[k.Text(16)]; !ok {
					toDelete = append(toDelete, k)
				}
			}

			if len(toAdd) == 0 && len(toDelete) == 0 {
				log.Printf("[%s] No changes detected, skipping", iss.ID)
				continue
			}

			log.Printf("[%s] Incremental: +%d adds, -%d deletes (from %d existing)",
				iss.ID, len(toAdd), len(toDelete), len(existingKeys))

			if len(toDelete) > 0 {
				if err := tree.BatchDelete(toDelete); err != nil {
					log.Printf("[%s] Skipping: batch delete error: %v", iss.ID, err)
					continue
				}
			}

			if len(toAdd) > 0 {
				err = tree.BatchAddWithProgress(toAdd, entryVal, 10000, func(done, total int) {
					log.Printf("[%s] Added %d / %d entries", iss.ID, done, total)
				})
				if err != nil {
					log.Printf("[%s] Skipping: batch add error: %v", iss.ID, err)
					continue
				}
			}
		} else {
			// Full rebuild
			tree = smt.New(hasher)
			log.Printf("[%s] Full rebuild: %d entries", iss.ID, len(uniqueSerials))
			err = tree.BatchAddWithProgress(uniqueSerials, entryVal, 10000, func(done, total int) {
				log.Printf("[%s] Added %d / %d entries", iss.ID, done, total)
			})
			if err != nil {
				log.Printf("[%s] Skipping: batch add error: %v", iss.ID, err)
				continue
			}
		}
		log.Printf("[%s] SMT ready: count=%d, root=0x%s, duration=%v",
			iss.ID, tree.Count, tree.Root.Text(16), time.Since(buildStart))

		// Check if root changed
		newRoot := "0x" + tree.Root.Text(16)
		rootPath := filepath.Join(issuerDir, "root.json")
		if existingRoot, err := readExistingRoot(rootPath); err == nil {
			if existingRoot == newRoot {
				log.Printf("[%s] Root unchanged, skipping snapshot export", iss.ID)
				continue
			}
		}

		// Write output files
		if err := os.MkdirAll(issuerDir, 0o755); err != nil {
			log.Printf("[%s] Skipping: mkdir error: %v", iss.ID, err)
			continue
		}

		// Export snapshot
		f, err := os.Create(snapshotPath)
		if err != nil {
			log.Printf("[%s] Skipping: create snapshot file: %v", iss.ID, err)
			continue
		}
		if err := snapshot.Export(tree, parsed.CRLNumber.Uint64(), f); err != nil {
			f.Close()
			log.Printf("[%s] Skipping: export snapshot: %v", iss.ID, err)
			continue
		}
		f.Close()
		log.Printf("[%s] Snapshot exported to %s", iss.ID, snapshotPath)

		// Write root.json
		info := rootInfo{
			Root:      newRoot,
			Count:     tree.Count,
			CRLNumber: parsed.CRLNumber.String(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		rootJSON, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Printf("[%s] Skipping: marshal root.json: %v", iss.ID, err)
			continue
		}
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

func readExistingRoot(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var info rootInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return "", err
	}
	return info.Root, nil
}
