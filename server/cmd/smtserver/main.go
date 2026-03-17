package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/moven0831/moica-revocation-smt/server/internal/api/grpcapi"
	"github.com/moven0831/moica-revocation-smt/server/internal/api/rest"
	"github.com/moven0831/moica-revocation-smt/server/internal/config"
	"github.com/moven0831/moica-revocation-smt/server/internal/crl"
	"github.com/moven0831/moica-revocation-smt/server/internal/manager"
	"github.com/moven0831/moica-revocation-smt/server/internal/snapshot"
	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
	pb "github.com/moven0831/moica-revocation-smt/server/pkg/proto/revocation"
)

func main() {
	cfg := config.Load()
	hasher := smt.NewPoseidonHasher()
	mgr := manager.New(hasher)

	// Try loading snapshots for fast startup
	issuers := []crl.IssuerConfig{
		{ID: "g2", URL: cfg.CRLG2URL},
		{ID: "g3", URL: cfg.CRLG3URL},
	}
	for _, iss := range issuers {
		snapPath := filepath.Join(cfg.DataDir, iss.ID, "tree-snapshot.json.gz")
		tree, err := snapshot.ImportFile(hasher, snapPath)
		if err != nil {
			log.Printf("No local snapshot for %s, downloading...", iss.ID)
			dlPath, dlErr := snapshot.Download(cfg.GitHubRepo, iss.ID, cfg.DataDir)
			if dlErr != nil {
				log.Printf("Snapshot download failed for %s: %v", iss.ID, dlErr)
				continue
			}
			tree, err = snapshot.ImportFile(hasher, dlPath)
			if err != nil {
				log.Printf("Snapshot import failed for %s: %v", iss.ID, err)
				continue
			}
		}
		mgr.SetTree(iss.ID, tree, 0) // CRLNumber unknown from snapshot; watcher will update
		log.Printf("Loaded snapshot for %s: count=%d", iss.ID, tree.Count)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start CRL watcher
	issuers = []crl.IssuerConfig{
		{ID: "g2", URL: cfg.CRLG2URL},
		{ID: "g3", URL: cfg.CRLG3URL},
	}
	watcher := crl.NewWatcher(
		time.Duration(cfg.CRLPollInterval)*time.Second,
		issuers, mgr, hasher,
	)
	go watcher.Start(ctx)

	// REST server
	restHandler := rest.NewHandler(mgr)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: restHandler.Router(),
	}

	go func() {
		log.Printf("REST server listening on :%d", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("REST server: %v", err)
		}
	}()

	// gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("gRPC listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRevocationProofServiceServer(grpcServer, grpcapi.NewRevocationServer(mgr))

	go func() {
		log.Printf("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server: %v", err)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()
	grpcServer.GracefulStop()
	httpServer.Shutdown(context.Background())
}
