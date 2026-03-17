package grpcapi

import (
	"context"
	"math/big"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/moven0831/moica-revocation-smt/server/internal/manager"
	pb "github.com/moven0831/moica-revocation-smt/server/pkg/proto/revocation"
)

// RevocationServer implements the gRPC RevocationProofService.
type RevocationServer struct {
	pb.UnimplementedRevocationProofServiceServer
	mgr       *manager.TreeManager
	startTime time.Time
}

// NewRevocationServer creates a new gRPC server.
func NewRevocationServer(mgr *manager.TreeManager) *RevocationServer {
	return &RevocationServer{
		mgr:       mgr,
		startTime: time.Now(),
	}
}

func bigToHex(n *big.Int) string {
	if n == nil || n.Sign() == 0 {
		return "0x0"
	}
	return "0x" + n.Text(16)
}

func (s *RevocationServer) GetProof(ctx context.Context, req *pb.GetProofRequest) (*pb.GetProofResponse, error) {
	snHex := strings.TrimPrefix(req.SerialNumber, "0x")
	if len(snHex) == 0 || len(snHex) > 64 {
		return nil, status.Error(codes.InvalidArgument, "invalid serial number")
	}

	sn, ok := new(big.Int).SetString(snHex, 16)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid serial number hex")
	}

	proof, err := s.mgr.GetProof(req.IssuerId, sn)
	if err != nil {
		if strings.Contains(err.Error(), "unknown issuer") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &pb.GetProofResponse{
		IssuerId:     req.IssuerId,
		SerialNumber: "0x" + snHex,
		Entry:        bigSliceToHex(proof.Entry),
		Siblings:     bigSliceToHex(proof.Siblings),
		Root:         bigToHex(proof.Root),
		Membership:   proof.Membership,
	}
	if proof.MatchingEntry != nil {
		resp.MatchingEntry = bigSliceToHex(proof.MatchingEntry)
	}

	return resp, nil
}

func (s *RevocationServer) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	mgStatus := s.mgr.Status()

	generations := make(map[string]*pb.IssuerStatus, len(mgStatus))
	for id, st := range mgStatus {
		generations[id] = &pb.IssuerStatus{
			Loaded:    st.Loaded,
			Count:     int32(st.Count),
			Root:      st.Root,
			CrlNumber: st.CRLNumber,
			LoadedAt:  st.LoadedAt,
		}
	}

	return &pb.GetStatusResponse{
		Generations:   generations,
		UptimeSeconds: time.Since(s.startTime).Seconds(),
	}, nil
}

func bigSliceToHex(s []*big.Int) []string {
	result := make([]string, len(s))
	for i, v := range s {
		result[i] = bigToHex(v)
	}
	return result
}
