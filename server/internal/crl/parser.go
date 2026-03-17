package crl

import (
	"crypto/x509"
	"fmt"
	"math/big"
)

// ParsedCRL holds the extracted data from a DER-encoded CRL.
type ParsedCRL struct {
	CRLNumber      *big.Int
	RevokedSerials []*big.Int
}

// ParseDER parses a DER-encoded X.509 CRL and extracts serial numbers.
func ParseDER(derBytes []byte) (*ParsedCRL, error) {
	rl, err := x509.ParseRevocationList(derBytes)
	if err != nil {
		return nil, fmt.Errorf("parse CRL: %w", err)
	}

	serials := make([]*big.Int, 0, len(rl.RevokedCertificateEntries))
	for _, entry := range rl.RevokedCertificateEntries {
		serials = append(serials, new(big.Int).Set(entry.SerialNumber))
	}

	return &ParsedCRL{
		CRLNumber:      rl.Number,
		RevokedSerials: serials,
	}, nil
}
