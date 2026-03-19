//go:build integration

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/moven0831/moica-revocation-smt/server/internal/api/grpcapi"
	"github.com/moven0831/moica-revocation-smt/server/internal/api/rest"
	"github.com/moven0831/moica-revocation-smt/server/internal/manager"
	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
	"github.com/moven0831/moica-revocation-smt/server/internal/snapshot"
	pb "github.com/moven0831/moica-revocation-smt/server/pkg/proto/revocation"
)

const (
	repo     = "moven0831/moica-revocation-smt"
	issuerID = "g2"
	bufSize  = 1024 * 1024
)

var (
	testRouter     http.Handler
	testGRPCClient pb.RevocationProofServiceClient
	grpcCleanup    func()
	hasher         smt.Hasher
	memberSerials  []string // known member serials extracted from the tree
	treeCount      int
)

func TestMain(m *testing.M) {
	h := smt.NewPoseidonHasher()
	hasher = h

	// Download G2 snapshot.
	tmpDir, err := os.MkdirTemp("", "integration-smt-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	snapshotPath, err := snapshot.Download(repo, issuerID, tmpDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to download snapshot: %v\n", err)
		os.Exit(1)
	}

	// Import snapshot into SMT.
	tree, err := snapshot.ImportFile(h, snapshotPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to import snapshot: %v\n", err)
		os.Exit(1)
	}
	treeCount = tree.Count

	// Extract known member serials from tree nodes.
	memberSerials = extractMemberSerials(tree, 10)
	if len(memberSerials) == 0 {
		fmt.Fprintf(os.Stderr, "no member serials found in tree\n")
		os.Exit(1)
	}

	// Set up TreeManager with the imported tree.
	mgr := manager.New(h)
	mgr.SetTree(issuerID, tree, 0)
	mgr.SetTree("g3", smt.New(h), 0)

	// Set up REST router.
	testRouter = rest.NewHandler(mgr).Router()

	// Set up gRPC in-process server.
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	pb.RegisterRevocationProofServiceServer(srv, grpcapi.NewRevocationServer(mgr))

	go func() {
		_ = srv.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "grpc.NewClient: %v\n", err)
		os.Exit(1)
	}
	testGRPCClient = pb.NewRevocationProofServiceClient(conn)
	grpcCleanup = func() {
		conn.Close()
		srv.Stop()
	}

	code := m.Run()

	grpcCleanup()
	os.Exit(code)
}

// extractMemberSerials finds leaf nodes in the tree and returns their keys as hex strings.
func extractMemberSerials(tree *smt.SMT, count int) []string {
	var serials []string
	for _, children := range tree.Nodes() {
		if children.IsLeaf() {
			key := children[0]
			// Verify this key is actually in the tree.
			if tree.Get(key) != nil {
				serials = append(serials, key.Text(16))
				if len(serials) >= count {
					break
				}
			}
		}
	}
	return serials
}

// --- REST helpers ---

func getProofResponse(t *testing.T, issuer, sn string) rest.ProofResponse {
	t.Helper()
	req := httptest.NewRequest("GET", fmt.Sprintf("/proof/%s/%s", issuer, sn), nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /proof/%s/%s: status %d, body: %s", issuer, sn, w.Code, w.Body.String())
	}

	var resp rest.ProofResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	return resp
}

func getStatusResponse(t *testing.T) rest.StatusResponse {
	t.Helper()
	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /status: status %d", w.Code)
	}

	var resp rest.StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	return resp
}

// parseHex converts a "0x"-prefixed hex string to *big.Int.
func parseHex(t *testing.T, s string) *big.Int {
	t.Helper()
	n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	if !ok {
		t.Fatalf("failed to parse hex: %s", s)
	}
	return n
}

// verifyRESTProofIndependently recomputes root from a REST proof response.
func verifyRESTProofIndependently(t *testing.T, resp *rest.ProofResponse) {
	t.Helper()

	siblings := make([]*big.Int, len(resp.Siblings))
	for i, s := range resp.Siblings {
		siblings[i] = parseHex(t, s)
	}

	root := parseHex(t, resp.Root)

	var node *big.Int
	var pathKey *big.Int

	if resp.MatchingEntry != nil {
		meKey := parseHex(t, resp.MatchingEntry[0])
		meVal := parseHex(t, resp.MatchingEntry[1])
		meMark := parseHex(t, resp.MatchingEntry[2])
		node = hasher.Hash3(meKey, meVal, meMark)
		pathKey = meKey
	} else if resp.Membership {
		key := parseHex(t, resp.Entry[0])
		val := parseHex(t, resp.Entry[1])
		mark := parseHex(t, resp.Entry[2])
		node = hasher.Hash3(key, val, mark)
		pathKey = key
	} else {
		node = new(big.Int)
		pathKey = parseHex(t, resp.Entry[0])
	}

	path := smt.KeyToPath(pathKey, resp.Depth)
	for i := len(siblings) - 1; i >= 0; i-- {
		if path[i] == 1 {
			node = hasher.Hash2(siblings[i], node)
		} else {
			node = hasher.Hash2(node, siblings[i])
		}
	}

	if node.Cmp(root) != 0 {
		t.Errorf("independent root recomputation failed: got 0x%s, want %s", node.Text(16), resp.Root)
	}
}

// verifyGRPCProofIndependently recomputes root from a gRPC proof response.
func verifyGRPCProofIndependently(t *testing.T, resp *pb.GetProofResponse) {
	t.Helper()

	siblings := make([]*big.Int, len(resp.Siblings))
	for i, s := range resp.Siblings {
		siblings[i] = parseHex(t, s)
	}

	root := parseHex(t, resp.Root)
	depth := int(resp.Depth)

	var node *big.Int
	var pathKey *big.Int

	if len(resp.MatchingEntry) > 0 {
		meKey := parseHex(t, resp.MatchingEntry[0])
		meVal := parseHex(t, resp.MatchingEntry[1])
		meMark := parseHex(t, resp.MatchingEntry[2])
		node = hasher.Hash3(meKey, meVal, meMark)
		pathKey = meKey
	} else if resp.Membership {
		key := parseHex(t, resp.Entry[0])
		val := parseHex(t, resp.Entry[1])
		mark := parseHex(t, resp.Entry[2])
		node = hasher.Hash3(key, val, mark)
		pathKey = key
	} else {
		node = new(big.Int)
		pathKey = parseHex(t, resp.Entry[0])
	}

	path := smt.KeyToPath(pathKey, depth)
	for i := len(siblings) - 1; i >= 0; i-- {
		if path[i] == 1 {
			node = hasher.Hash2(siblings[i], node)
		} else {
			node = hasher.Hash2(node, siblings[i])
		}
	}

	if node.Cmp(root) != 0 {
		t.Errorf("independent root recomputation failed: got 0x%s, want %s", node.Text(16), resp.Root)
	}
}

// --- REST Tests ---

func TestIntegrationRESTMembershipProof(t *testing.T) {
	for _, serial := range memberSerials {
		t.Run(serial[:8], func(t *testing.T) {
			resp := getProofResponse(t, issuerID, serial)

			if !resp.Membership {
				t.Fatal("expected membership=true")
			}
			if len(resp.Entry) != 3 {
				t.Fatalf("entry length: got %d, want 3", len(resp.Entry))
			}
			if resp.Entry[1] != "0x1" {
				t.Errorf("entry[1]: got %s, want 0x1", resp.Entry[1])
			}
			if resp.Entry[2] != "0x1" {
				t.Errorf("entry[2]: got %s, want 0x1", resp.Entry[2])
			}
			if resp.Depth != smt.DefaultDepth {
				t.Errorf("depth: got %d, want %d", resp.Depth, smt.DefaultDepth)
			}

			ok, err := rest.VerifyProofFromResponse(hasher, &resp, smt.DefaultDepth)
			if err != nil {
				t.Fatalf("VerifyProofFromResponse: %v", err)
			}
			if !ok {
				t.Error("VerifyProofFromResponse failed")
			}

			verifyRESTProofIndependently(t, &resp)
		})
	}
}

func TestIntegrationRESTNonMembershipProof(t *testing.T) {
	nonMembers := []string{"1", "2", "DEAD"}

	for _, serial := range nonMembers {
		t.Run(serial, func(t *testing.T) {
			resp := getProofResponse(t, issuerID, serial)

			if resp.Membership {
				t.Fatal("expected membership=false")
			}
			if resp.MatchingEntry != nil {
				if len(resp.MatchingEntry) != 3 {
					t.Fatalf("matchingEntry length: got %d, want 3", len(resp.MatchingEntry))
				}
			}

			ok, err := rest.VerifyProofFromResponse(hasher, &resp, smt.DefaultDepth)
			if err != nil {
				t.Fatalf("VerifyProofFromResponse: %v", err)
			}
			if !ok {
				t.Error("VerifyProofFromResponse failed")
			}

			verifyRESTProofIndependently(t, &resp)
		})
	}
}

func TestIntegrationRESTRootConsistency(t *testing.T) {
	statusResp := getStatusResponse(t)

	g2Status, ok := statusResp.Generations[issuerID]
	if !ok {
		t.Fatal("g2 not in generations")
	}

	if !g2Status.Loaded {
		t.Fatal("g2 not loaded")
	}

	// Verify tree count is in the expected range for G2 (~400k+).
	if g2Status.Count < 100000 {
		t.Errorf("g2 count too low: got %d, expected 100k+", g2Status.Count)
	}
	if g2Status.Count != treeCount {
		t.Errorf("g2 count mismatch: status=%d, tree=%d", g2Status.Count, treeCount)
	}

	// Compare root from /status with root from /proof responses.
	proofResp := getProofResponse(t, issuerID, memberSerials[0])
	if proofResp.Root != g2Status.Root {
		t.Errorf("root mismatch: proof=%s, status=%s", proofResp.Root, g2Status.Root)
	}

	// Verify a non-member proof also returns the same root.
	nonMemberResp := getProofResponse(t, issuerID, "DEAD")
	if nonMemberResp.Root != g2Status.Root {
		t.Errorf("non-member root mismatch: proof=%s, status=%s", nonMemberResp.Root, g2Status.Root)
	}
}

// --- gRPC Tests ---

func TestIntegrationGRPCMembershipProof(t *testing.T) {
	ctx := context.Background()

	for _, serial := range memberSerials {
		t.Run(serial[:8], func(t *testing.T) {
			resp, err := testGRPCClient.GetProof(ctx, &pb.GetProofRequest{
				IssuerId:     issuerID,
				SerialNumber: serial,
			})
			if err != nil {
				t.Fatalf("GetProof: %v", err)
			}

			if !resp.Membership {
				t.Fatal("expected membership=true")
			}
			if len(resp.Entry) != 3 {
				t.Fatalf("entry length: got %d, want 3", len(resp.Entry))
			}
			if resp.Entry[1] != "0x1" {
				t.Errorf("entry[1]: got %s, want 0x1", resp.Entry[1])
			}
			if resp.Entry[2] != "0x1" {
				t.Errorf("entry[2]: got %s, want 0x1", resp.Entry[2])
			}
			if resp.Depth != int32(smt.DefaultDepth) {
				t.Errorf("depth: got %d, want %d", resp.Depth, smt.DefaultDepth)
			}

			verifyGRPCProofIndependently(t, resp)
		})
	}
}

func TestIntegrationGRPCNonMembershipProof(t *testing.T) {
	ctx := context.Background()
	nonMembers := []string{"1", "2", "DEAD"}

	for _, serial := range nonMembers {
		t.Run(serial, func(t *testing.T) {
			resp, err := testGRPCClient.GetProof(ctx, &pb.GetProofRequest{
				IssuerId:     issuerID,
				SerialNumber: serial,
			})
			if err != nil {
				t.Fatalf("GetProof: %v", err)
			}

			if resp.Membership {
				t.Fatal("expected membership=false")
			}
			if len(resp.MatchingEntry) > 0 {
				if len(resp.MatchingEntry) != 3 {
					t.Fatalf("matchingEntry length: got %d, want 3", len(resp.MatchingEntry))
				}
			}

			verifyGRPCProofIndependently(t, resp)
		})
	}
}
