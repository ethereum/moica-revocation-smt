package snapshot

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Download fetches a snapshot from a GitHub Release and writes it to destDir/{issuerID}/tree-snapshot.json.gz.
// Returns the file path on success.
func Download(repo, issuerID, destDir string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/snapshot-latest/%s-tree-snapshot.json.gz", repo, issuerID)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download snapshot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download snapshot: HTTP %d", resp.StatusCode)
	}

	dir := filepath.Join(destDir, issuerID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	destPath := filepath.Join(dir, "tree-snapshot.json.gz")
	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(destPath)
		return "", fmt.Errorf("write snapshot: %w", err)
	}

	return destPath, nil
}
