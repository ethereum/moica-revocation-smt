package crl

import (
	"fmt"
	"io"
	"net/http"
)

// FetchDER downloads a DER-encoded CRL from the given URL.
func FetchDER(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch CRL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch CRL: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read CRL body: %w", err)
	}
	return data, nil
}
