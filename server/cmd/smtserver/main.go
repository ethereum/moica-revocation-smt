package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/moven0831/moica-revocation-smt/server/internal/api/grpcapi"
	"github.com/moven0831/moica-revocation-smt/server/internal/api/rest"
	"github.com/moven0831/moica-revocation-smt/server/internal/config"
	"github.com/moven0831/moica-revocation-smt/server/internal/crl"
	"github.com/moven0831/moica-revocation-smt/server/internal/manager"
	"github.com/moven0831/moica-revocation-smt/server/internal/smt"
	pb "github.com/moven0831/moica-revocation-smt/server/pkg/proto/revocation"
)

func main() {
	cfg := config.Load()
	hasher := smt.NewPoseidonHasher()
	mgr := manager.New(hasher)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start CRL watcher
	issuers := []crl.IssuerConfig{
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
