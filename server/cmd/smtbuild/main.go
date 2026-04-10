package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/moven0831/moica-revocation-smt/server/internal/chain"
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
	postRoot := flag.Bool("post-root", false, "Post SMT roots on-chain (reads root.json files, skips SMT build)")
	exportBinary := flag.Bool("binary", false, "Also export binary format snapshot alongside JSON")
	convertBinary := flag.String("convert-binary", "", "Convert JSON snapshot to binary format (path to .json.gz input)")
	flag.Parse()

	cfg := config.Load()

	if *convertBinary != "" {
		convertJSONToBinary(*convertBinary)
		return
	}

	if *postRoot {
		postRootOnChain(cfg)
		return
	}

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

		var existingCRLNum uint64
		tree, existingCRLNum, err = snapshot.ImportFile(hasher, snapshotPath)
		if err != nil {
			log.Printf("[%s] No local snapshot, trying GitHub Release", iss.ID)
			if dlPath, dlErr := snapshot.Download(cfg.GitHubRepo, iss.ID, cfg.DataDir); dlErr == nil {
				tree, existingCRLNum, err = snapshot.ImportFile(hasher, dlPath)
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
			if existingRoot == newRoot && existingCRLNum > 0 {
				log.Printf("[%s] Root unchanged, skipping snapshot export", iss.ID)
				continue
			}
		}

		// Export snapshot (atomic write via temp file + rename)
		if err := snapshot.ExportFile(tree, parsed.CRLNumber.Uint64(), snapshotPath); err != nil {
			log.Printf("[%s] Skipping: export snapshot: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Snapshot exported to %s", iss.ID, snapshotPath)

		if *exportBinary {
			binaryPath := filepath.Join(issuerDir, "tree-snapshot.bin.gz")
			if err := snapshot.ExportBinaryFile(tree, parsed.CRLNumber.Uint64(), binaryPath); err != nil {
				log.Printf("[%s] Warning: binary export failed: %v", iss.ID, err)
			} else {
				log.Printf("[%s] Binary snapshot exported to %s", iss.ID, binaryPath)
			}
		}

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

func postRootOnChain(cfg *config.Config) {
	if cfg.RPCURL == "" || cfg.RelayerPrivateKey == "" || cfg.ContractAddress == "" {
		log.Println("Skipping on-chain posting: RPC_URL, RELAYER_PRIVATE_KEY, or CONTRACT_ADDRESS not set")
		return
	}

	client, err := chain.NewClient(cfg.RPCURL)
	if err != nil {
		log.Fatalf("Failed to connect to RPC: %v", err)
	}
	defer client.Close()

	relayer, err := chain.NewRelayer(client, cfg.RelayerPrivateKey, cfg.ContractAddress)
	if err != nil {
		log.Fatalf("Failed to create relayer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := relayer.VerifyContract(ctx); err != nil {
		cancel()
		log.Fatalf("Contract verification failed: %v", err)
	}
	cancel()
	log.Printf("Contract verified at %s (relayer: %s)", cfg.ContractAddress, relayer.Address().Hex())

	type issuerEntry struct {
		ID       string
		IssuerID [32]byte
	}
	entries := []issuerEntry{
		{ID: "g2", IssuerID: chain.IssuerG2},
		{ID: "g3", IssuerID: chain.IssuerG3},
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	for _, iss := range entries {
		rootPath := filepath.Join(cfg.DataDir, iss.ID, "root.json")
		data, err := os.ReadFile(rootPath)
		if err != nil {
			log.Printf("[%s] Skipping: %v", iss.ID, err)
			continue
		}

		var info rootInfo
		if err := json.Unmarshal(data, &info); err != nil {
			log.Printf("[%s] Skipping: parse root.json: %v", iss.ID, err)
			continue
		}

		root, ok := new(big.Int).SetString(strings.TrimPrefix(info.Root, "0x"), 16)
		if !ok {
			log.Printf("[%s] Skipping: invalid root hex: %s", iss.ID, info.Root)
			continue
		}

		crlNumber, ok := new(big.Int).SetString(info.CRLNumber, 10)
		if !ok {
			log.Printf("[%s] Skipping: invalid crlNumber: %s", iss.ID, info.CRLNumber)
			continue
		}

		log.Printf("[%s] Posting root on-chain: root=%s crlNumber=%s", iss.ID, info.Root, info.CRLNumber)
		tx, err := relayer.PostRoot(ctx, iss.IssuerID, root, crlNumber)
		if err != nil {
			if strings.Contains(err.Error(), "stale CRL") {
				log.Printf("[%s] Already posted (stale CRL), skipping", iss.ID)
				continue
			}
			log.Printf("[%s] Failed to post root: %v", iss.ID, err)
			continue
		}
		log.Printf("[%s] Root posted on-chain: tx=%s", iss.ID, tx.Hash().Hex())
	}

	log.Println("Done — on-chain posting complete")
}

func convertJSONToBinary(jsonPath string) {
	hasher := smt.NewPoseidonHasher()

	log.Printf("Loading JSON snapshot from %s", jsonPath)
	tree, crlNumber, err := snapshot.ImportFile(hasher, jsonPath)
	if err != nil {
		log.Fatalf("Failed to import JSON snapshot: %v", err)
	}
	log.Printf("Loaded: %d entries, root=0x%s, CRL#%d", tree.Count, tree.Root.Text(16)[:16], crlNumber)

	// Output path: same directory, replace .json.gz with .bin.gz
	outPath := strings.TrimSuffix(jsonPath, ".json.gz") + ".bin.gz"
	log.Printf("Exporting binary snapshot to %s", outPath)
	if err := snapshot.ExportBinaryFile(tree, crlNumber, outPath); err != nil {
		log.Fatalf("Failed to export binary: %v", err)
	}

	// Report sizes
	jsonInfo, _ := os.Stat(jsonPath)
	binInfo, _ := os.Stat(outPath)
	if jsonInfo != nil && binInfo != nil {
		log.Printf("JSON: %d bytes, Binary: %d bytes (%.1f%%)",
			jsonInfo.Size(), binInfo.Size(),
			float64(binInfo.Size())/float64(jsonInfo.Size())*100)
	}
	log.Println("Done")
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
