//go:build integration

package crl

import (
	"math/big"
	"testing"
	"time"

	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

const (
	crlG2URL = "https://moica.nat.gov.tw/repository/MOICA/CRL2/complete.crl"
	crlG3URL = "https://moica.nat.gov.tw/repository/MOICA/CRL3/complete.crl"
)

func TestIntegrationG2CRL(t *testing.T) {
	testCRLIntegration(t, "G2", crlG2URL, 100_000)
}

func TestIntegrationG3CRL(t *testing.T) {
	testCRLIntegration(t, "G3", crlG3URL, 50_000)
}

func testCRLIntegration(t *testing.T, name, url string, minSerials int) {
	t.Helper()

	// Step 1: Fetch CRL
	t.Logf("[%s] Fetching CRL from %s", name, url)
	fetchStart := time.Now()
	derBytes, err := FetchDER(url)
	if err != nil {
		t.Skipf("Skipping %s: MOICA server unreachable: %v", name, err)
	}
	t.Logf("[%s] Fetched %d bytes in %v", name, len(derBytes), time.Since(fetchStart))

	// Step 2: Parse CRL
	parseStart := time.Now()
	parsed, err := ParseDER(derBytes)
	if err != nil {
		t.Fatalf("[%s] ParseDER failed: %v", name, err)
	}
	t.Logf("[%s] Parsed %d revoked serials (CRLNumber=%s) in %v",
		name, len(parsed.RevokedSerials), parsed.CRLNumber, time.Since(parseStart))

	if parsed.CRLNumber == nil {
		t.Fatalf("[%s] CRLNumber is nil", name)
	}
	if len(parsed.RevokedSerials) < minSerials {
		t.Fatalf("[%s] Expected at least %d serials, got %d", name, minSerials, len(parsed.RevokedSerials))
	}

	// Step 3: Deduplicate serials
	seen := make(map[string]struct{}, len(parsed.RevokedSerials))
	uniqueSerials := make([]*big.Int, 0, len(parsed.RevokedSerials))
	for _, s := range parsed.RevokedSerials {
		key := s.Text(16)
		if _, dup := seen[key]; !dup {
			seen[key] = struct{}{}
			uniqueSerials = append(uniqueSerials, s)
		}
	}
	t.Logf("[%s] %d unique serials (removed %d duplicates)",
		name, len(uniqueSerials), len(parsed.RevokedSerials)-len(uniqueSerials))

	// Step 4: Build SMT (depth 128 is sufficient for CRL serial numbers)
	const treeDepth = 128
	hasher := smt.NewPoseidonHasher()
	tree := smt.NewWithDepth(hasher, treeDepth)
	entryVal := big.NewInt(1)

	buildStart := time.Now()
	batchSize := 10_000
	err = tree.BatchAddWithProgress(uniqueSerials, entryVal, batchSize, func(done, total int) {
		t.Logf("[%s] Added %d / %d entries", name, done, total)
	})
	if err != nil {
		t.Fatalf("[%s] BatchAdd failed: %v", name, err)
	}
	buildDuration := time.Since(buildStart)
	t.Logf("[%s] SMT built: count=%d, root=0x%s, duration=%v",
		name, tree.Count, tree.Root.Text(16), buildDuration)

	if tree.Count < minSerials {
		t.Fatalf("[%s] Expected tree count >= %d, got %d", name, minSerials, tree.Count)
	}
	if tree.Root.Sign() == 0 {
		t.Fatalf("[%s] Root is zero after adding entries", name)
	}

	// Step 5: Membership proof (first serial)
	memberKey := uniqueSerials[0]
	proofStart := time.Now()
	memberProof := tree.CreateProof(memberKey)
	t.Logf("[%s] Membership proof generated in %v", name, time.Since(proofStart))

	if !memberProof.Membership {
		t.Fatalf("[%s] Expected membership=true for serial 0x%s", name, memberKey.Text(16))
	}
	if !smt.VerifyProof(hasher, memberProof, treeDepth) {
		t.Fatalf("[%s] Membership proof verification failed for serial 0x%s", name, memberKey.Text(16))
	}
	t.Logf("[%s] Membership proof verified for serial 0x%s", name, memberKey.Text(16))

	// Step 6: Non-membership proof (key unlikely to be a real serial)
	nonMemberKey := big.NewInt(9999)
	nonMemberProof := tree.CreateProof(nonMemberKey)

	if nonMemberProof.Membership {
		t.Fatalf("[%s] Expected membership=false for key 9999", name)
	}
	if !smt.VerifyProof(hasher, nonMemberProof, treeDepth) {
		t.Fatalf("[%s] Non-membership proof verification failed for key 9999", name)
	}
	t.Logf("[%s] Non-membership proof verified for key 9999", name)

	t.Logf("[%s] Integration test passed: %d entries, root=0x%s",
		name, tree.Count, tree.Root.Text(16))
}
