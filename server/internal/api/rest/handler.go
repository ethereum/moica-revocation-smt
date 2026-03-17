package rest

import (
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/moven0831/moica-revocation-smt/server/internal/manager"
	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
)

// Handler holds the REST API dependencies.
type Handler struct {
	mgr       *manager.TreeManager
	startTime time.Time
}

// NewHandler creates a new REST handler.
func NewHandler(mgr *manager.TreeManager) *Handler {
	return &Handler{
		mgr:       mgr,
		startTime: time.Now(),
	}
}

// Router returns a chi router with all REST endpoints.
func (h *Handler) Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/proof/{issuerId}/{sn}", h.getProof)
	r.Get("/status", h.getStatus)

	return r
}

// ProofResponse matches the TS API response format.
type ProofResponse struct {
	IssuerID      string   `json:"issuerId"`
	SerialNumber  string   `json:"serialNumber"`
	Entry         []string `json:"entry"`
	MatchingEntry []string `json:"matchingEntry,omitempty"`
	Siblings      []string `json:"siblings"`
	Root          string   `json:"root"`
	Membership    bool     `json:"membership"`
}

// StatusResponse matches the TS API status format.
type StatusResponse struct {
	Generations   map[string]manager.IssuerStatus `json:"generations"`
	UptimeSeconds float64                          `json:"uptimeSeconds"`
}

func bigToHex(n *big.Int) string {
	if n == nil || n.Sign() == 0 {
		return "0x0"
	}
	return "0x" + n.Text(16)
}

func (h *Handler) getProof(w http.ResponseWriter, r *http.Request) {
	issuerID := chi.URLParam(r, "issuerId")
	snHex := chi.URLParam(r, "sn")

	// Validate serial number: must be hex, max 64 chars
	snHex = strings.TrimPrefix(snHex, "0x")
	if len(snHex) == 0 || len(snHex) > 64 {
		http.Error(w, `{"error":"invalid serial number"}`, http.StatusBadRequest)
		return
	}
	sn, ok := new(big.Int).SetString(snHex, 16)
	if !ok {
		http.Error(w, `{"error":"invalid serial number hex"}`, http.StatusBadRequest)
		return
	}

	proof, err := h.mgr.GetProof(issuerID, sn)
	if err != nil {
		if strings.Contains(err.Error(), "unknown issuer") {
			http.Error(w, `{"error":"unknown issuer"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		}
		return
	}

	resp := ProofResponse{
		IssuerID:     issuerID,
		SerialNumber: "0x" + snHex,
		Entry:        bigSliceToHex(proof.Entry),
		Siblings:     bigSliceToHex(proof.Siblings),
		Root:         bigToHex(proof.Root),
		Membership:   proof.Membership,
	}
	if proof.MatchingEntry != nil {
		resp.MatchingEntry = bigSliceToHex(proof.MatchingEntry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	resp := StatusResponse{
		Generations:   h.mgr.Status(),
		UptimeSeconds: time.Since(h.startTime).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func bigSliceToHex(s []*big.Int) []string {
	result := make([]string, len(s))
	for i, v := range s {
		result[i] = bigToHex(v)
	}
	return result
}

// VerifyProofFromResponse is a helper to verify proofs returned by the API.
// Exported for testing purposes.
func VerifyProofFromResponse(h smt.Hasher, resp *ProofResponse, depth int) (bool, error) {
	entry := make([]*big.Int, len(resp.Entry))
	for i, s := range resp.Entry {
		n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
		if !ok {
			return false, nil
		}
		entry[i] = n
	}

	siblings := make([]*big.Int, len(resp.Siblings))
	for i, s := range resp.Siblings {
		n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
		if !ok {
			return false, nil
		}
		siblings[i] = n
	}

	root, ok := new(big.Int).SetString(strings.TrimPrefix(resp.Root, "0x"), 16)
	if !ok {
		return false, nil
	}

	proof := &smt.MerkleProof{
		Entry:    entry,
		Siblings: siblings,
		Root:     root,
		Membership: resp.Membership,
	}

	if resp.MatchingEntry != nil {
		me := make([]*big.Int, len(resp.MatchingEntry))
		for i, s := range resp.MatchingEntry {
			n, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
			if !ok {
				return false, nil
			}
			me[i] = n
		}
		proof.MatchingEntry = me
	}

	return smt.VerifyProof(h, proof, depth), nil
}
