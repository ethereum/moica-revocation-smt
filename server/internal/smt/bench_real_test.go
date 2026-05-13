//go:build bench_real

// This file is the SMT-side companion to the LeanIMT+ bench at
// moica-revocation-leanimt-plus/server/internal/leanimt_plus/bench_real_test.go.
// They MUST share the same query-selection helpers (seed=42, K=1024) so both
// impls measure on identical inputs.
//
// Place this file at: server/internal/smt/bench_real_test.go
// Run via: MOICA_BENCH_DER_DIR=<path> go test -tags=bench_real -bench=. ./internal/smt
package smt_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/moven0831/moica-revocation-smt/server/internal/crl"
	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
	"github.com/moven0831/moica-revocation-smt/server/internal/snapshot"
)

const (
	benchQueryCount = 1024
	benchSeed       = 42
	smtDepth        = 128
)

var smtValue = big.NewInt(1)

type datasetID string

const (
	datasetG2 datasetID = "G2"
	datasetG3 datasetID = "G3"
)

type loadedDataset struct {
	once    sync.Once
	serials []*big.Int
	err     error
}

var datasets = map[datasetID]*loadedDataset{
	datasetG2: {},
	datasetG3: {},
}

func derFilename(id datasetID) string {
	switch id {
	case datasetG2:
		return "g2.crl.der"
	case datasetG3:
		return "g3.crl.der"
	default:
		return ""
	}
}

func loadSerials(tb testing.TB, id datasetID) []*big.Int {
	dir := os.Getenv("MOICA_BENCH_DER_DIR")
	if dir == "" {
		tb.Skipf("MOICA_BENCH_DER_DIR not set; run via the leanimt-plus harness (bench/run.sh).")
	}
	entry := datasets[id]
	entry.once.Do(func() {
		path := filepath.Join(dir, derFilename(id))
		raw, err := os.ReadFile(path)
		if err != nil {
			entry.err = fmt.Errorf("read %s: %w", path, err)
			return
		}
		parsed, err := crl.ParseDER(raw)
		if err != nil {
			entry.err = fmt.Errorf("parse %s: %w", path, err)
			return
		}
		entry.serials = dedupAndSort(parsed.RevokedSerials)
	})
	if entry.err != nil {
		tb.Fatal(entry.err)
	}
	if len(entry.serials) == 0 {
		tb.Fatalf("dataset %s loaded empty", id)
	}
	return entry.serials
}

// dedupAndSort mirrors leanimt_plus/server/internal/crl.DedupAndSortSerials.
// Inlined here because the SMT-repo crl package does not export an equivalent
// helper, and both impls must consume the exact same set of keys.
func dedupAndSort(serials []*big.Int) []*big.Int {
	if len(serials) == 0 {
		return nil
	}
	out := make([]*big.Int, 0, len(serials))
	for _, s := range serials {
		if s != nil && s.Sign() > 0 {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cmp(out[j]) < 0 })
	if len(out) <= 1 {
		return out
	}
	w := 1
	for i := 1; i < len(out); i++ {
		if out[i].Cmp(out[w-1]) != 0 {
			out[w] = out[i]
			w++
		}
	}
	return out[:w]
}

// Helpers below must stay byte-identical to the LeanIMT+ side at
// moica-revocation-leanimt-plus/server/internal/leanimt_plus/bench_real_test.go.

func membershipQueries(serials []*big.Int, k int, seed int64) []*big.Int {
	r := rand.New(rand.NewSource(seed))
	out := make([]*big.Int, k)
	for i := range out {
		out[i] = new(big.Int).Set(serials[r.Intn(len(serials))])
	}
	return out
}

func nonMembershipQueries(serials []*big.Int, k int, seed int64) []*big.Int {
	r := rand.New(rand.NewSource(seed))
	maxBits := serials[len(serials)-1].BitLen()
	if maxBits < 8 {
		maxBits = 8
	}
	bound := new(big.Int).Lsh(big.NewInt(1), uint(maxBits))
	out := make([]*big.Int, 0, k)
	for len(out) < k {
		v := new(big.Int).Rand(r, bound)
		if v.Sign() == 0 {
			continue
		}
		idx := sort.Search(len(serials), func(i int) bool { return serials[i].Cmp(v) >= 0 })
		if idx < len(serials) && serials[idx].Cmp(v) == 0 {
			continue
		}
		out = append(out, v)
	}
	return out
}

func newTreeFor(tb testing.TB, serials []*big.Int) *smt.SMT {
	h := smt.NewPoseidonHasher()
	tree := smt.New(h)
	if err := tree.BatchAdd(serials, smtValue); err != nil {
		tb.Fatalf("build tree: %v", err)
	}
	return tree
}

// SMT non-membership proofs add the matching entry to the wire shape.
func proofSize(p *smt.MerkleProof) int {
	n := len(p.Siblings)
	if p.MatchingEntry != nil {
		n++
	}
	return n
}

func BenchmarkReal_Build(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			h := smt.NewPoseidonHasher()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tree := smt.New(h)
				if err := tree.BatchAdd(serials, smtValue); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkReal_ProofGen(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			tree := newTreeFor(b, serials)
			memQ := membershipQueries(serials, benchQueryCount, benchSeed)
			nonQ := nonMembershipQueries(serials, benchQueryCount, benchSeed)

			b.Run("Membership", func(b *testing.B) { benchProofGen(b, tree, memQ) })
			b.Run("NonMembership", func(b *testing.B) { benchProofGen(b, tree, nonQ) })
		})
	}
}

func benchProofGen(b *testing.B, tree *smt.SMT, queries []*big.Int) {
	sample := tree.CreateProof(queries[0])
	if sample == nil {
		b.Fatal("sample proof was nil")
	}
	blob, err := json.Marshal(sample)
	if err != nil {
		b.Fatalf("marshal sample: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if p := tree.CreateProof(queries[i%len(queries)]); p == nil {
			b.Fatal("nil proof")
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(proofSize(sample)), "siblings/op")
	b.ReportMetric(float64(len(blob)), "proofBytes/op")
}

func BenchmarkReal_Verify(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			h := smt.NewPoseidonHasher()
			tree := newTreeFor(b, serials)
			memQ := membershipQueries(serials, benchQueryCount, benchSeed)
			nonQ := nonMembershipQueries(serials, benchQueryCount, benchSeed)
			memProofs := preGenerate(b, tree, memQ)
			nonProofs := preGenerate(b, tree, nonQ)

			b.Run("Membership", func(b *testing.B) { benchVerify(b, h, memProofs) })
			b.Run("NonMembership", func(b *testing.B) { benchVerify(b, h, nonProofs) })
		})
	}
}

func preGenerate(b *testing.B, tree *smt.SMT, queries []*big.Int) []*smt.MerkleProof {
	out := make([]*smt.MerkleProof, len(queries))
	for i, q := range queries {
		p := tree.CreateProof(q)
		if p == nil {
			b.Fatalf("nil proof for query %d", i)
		}
		out[i] = p
	}
	return out
}

func benchVerify(b *testing.B, h smt.Hasher, proofs []*smt.MerkleProof) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !smt.VerifyProof(h, proofs[i%len(proofs)], smtDepth) {
			b.Fatal("proof failed to verify")
		}
	}
}

// Kept off the main timing benches: atomic.Add adds ~5-10 ns/op.
type countingHasher struct {
	inner smt.Hasher
	n2    atomic.Uint64
	n3    atomic.Uint64
}

func (c *countingHasher) Hash2(a, b *big.Int) *big.Int {
	c.n2.Add(1)
	return c.inner.Hash2(a, b)
}

func (c *countingHasher) Hash3(a, b, d *big.Int) *big.Int {
	c.n3.Add(1)
	return c.inner.Hash3(a, b, d)
}

func BenchmarkReal_HashCount_Build(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			c := &countingHasher{inner: smt.NewPoseidonHasher()}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				c.n2.Store(0)
				c.n3.Store(0)
				tree := smt.New(c)
				if err := tree.BatchAdd(serials, smtValue); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(c.n2.Load()), "hash2/op")
			b.ReportMetric(float64(c.n3.Load()), "hash3/op")
		})
	}
}

func BenchmarkReal_HashCount_Verify(b *testing.B) {
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		ds := ds
		b.Run(string(ds), func(b *testing.B) {
			serials := loadSerials(b, ds)
			h := smt.NewPoseidonHasher()
			tree := newTreeFor(b, serials)
			memQ := membershipQueries(serials, benchQueryCount, benchSeed)
			nonQ := nonMembershipQueries(serials, benchQueryCount, benchSeed)
			memProofs := preGenerate(b, tree, memQ)
			nonProofs := preGenerate(b, tree, nonQ)
			c := &countingHasher{inner: h}

			b.Run("Membership", func(b *testing.B) {
				c.n2.Store(0)
				c.n3.Store(0)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if !smt.VerifyProof(c, memProofs[i%len(memProofs)], smtDepth) {
						b.Fatal("verify failed")
					}
				}
				b.StopTimer()
				if b.N > 0 {
					b.ReportMetric(float64(c.n2.Load())/float64(b.N), "hash2/op")
					b.ReportMetric(float64(c.n3.Load())/float64(b.N), "hash3/op")
				}
			})
			b.Run("NonMembership", func(b *testing.B) {
				c.n2.Store(0)
				c.n3.Store(0)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if !smt.VerifyProof(c, nonProofs[i%len(nonProofs)], smtDepth) {
						b.Fatal("verify failed")
					}
				}
				b.StopTimer()
				if b.N > 0 {
					b.ReportMetric(float64(c.n2.Load())/float64(b.N), "hash2/op")
					b.ReportMetric(float64(c.n3.Load())/float64(b.N), "hash3/op")
				}
			})
		})
	}
}

func TestReal_SnapshotSize(t *testing.T) {
	tmp := t.TempDir()
	for _, ds := range []datasetID{datasetG2, datasetG3} {
		serials := loadSerials(t, ds)
		tree := newTreeFor(t, serials)

		jsonPath := filepath.Join(tmp, fmt.Sprintf("%s.json.gz", ds))
		if err := snapshot.ExportFile(tree, 0, jsonPath); err != nil {
			t.Fatalf("%s: ExportFile: %v", ds, err)
		}
		jsonGz, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatalf("%s: read json: %v", ds, err)
		}
		jsonRaw := gunzipLen(t, jsonGz)
		t.Logf("SNAPSHOT %s gz=%d raw=%d leaves=%d depth=%d format=json.gz",
			ds, len(jsonGz), jsonRaw, len(serials), smtDepth)

		binPath := filepath.Join(tmp, fmt.Sprintf("%s.bin.gz", ds))
		if err := snapshot.ExportBinaryFile(tree, 0, binPath); err != nil {
			t.Fatalf("%s: ExportBinaryFile: %v", ds, err)
		}
		binGz, err := os.ReadFile(binPath)
		if err != nil {
			t.Fatalf("%s: read bin: %v", ds, err)
		}
		binRaw := gunzipLen(t, binGz)
		t.Logf("SNAPSHOT %s gz=%d raw=%d leaves=%d depth=%d format=bin.gz",
			ds, len(binGz), binRaw, len(serials), smtDepth)
	}
}

func gunzipLen(t *testing.T, gz []byte) int {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		t.Fatalf("gunzip: %v", err)
	}
	defer gr.Close()
	raw, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("gunzip read: %v", err)
	}
	return len(raw)
}
