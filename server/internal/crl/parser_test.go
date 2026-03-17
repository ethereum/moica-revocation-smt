package crl

import (
	"os"
	"testing"
)

func TestParseDER(t *testing.T) {
	// Try to use the real G3 CRL fixture from the TS project
	fixturePath := "../../../testdata/MOICA-G3-complete.crl"
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Skipf("CRL fixture not found at %s, skipping: %v", fixturePath, err)
	}

	parsed, err := ParseDER(data)
	if err != nil {
		t.Fatal("ParseDER:", err)
	}

	if parsed.CRLNumber == nil {
		t.Fatal("CRLNumber should not be nil")
	}

	if len(parsed.RevokedSerials) == 0 {
		t.Fatal("expected revoked serials")
	}

	t.Logf("CRL number: %s", parsed.CRLNumber.String())
	t.Logf("Revoked serials: %d", len(parsed.RevokedSerials))
	t.Logf("First serial: %s", parsed.RevokedSerials[0].Text(16))
}
