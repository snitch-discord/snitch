package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"snitch/internal/db/service"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
)

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func main() {
	port := flag.Int("port", 5200, "port to listen on")
	dbDir := flag.String("db-dir", "./data", "directory to store database files")
	certFile := flag.String("cert", "./certs/db/cert.pem", "TLS certificate file")
	keyFile := flag.String("key", "./certs/db/key.pem", "TLS private key file")
	flag.Parse()

	slogger := slog.Default()
	ctx := context.Background()

	// Initialize database service
	dbService, err := service.NewDatabaseService(ctx, *dbDir, slogger)
	if err != nil {
		fatal("Failed to initialize database service", "error", err)
	}
	defer func() {
		if err := dbService.Close(); err != nil {
			slogger.Error("Failed to close database service", "error", err)
		}
	}()

	// Run migrations on all existing tenant databases
	if err := dbService.RunMigrationsOnAllTenants(ctx); err != nil {
		slogger.Warn("Failed to migrate some tenant databases", "error", err)
	}

	// Load TLS certificate
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		fatal("Failed to load TLS certificate", "error", err)
	}

	// Setup gRPC handlers
	mux := http.NewServeMux()
	mux.Handle(snitchv1connect.NewDatabaseServiceHandler(dbService, connect.WithInterceptors()))

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Configure HTTP/2 explicitly
	if err := http2.ConfigureServer(server, &http2.Server{}); err != nil {
		fatal("Failed to configure HTTP/2", "error", err)
	}

	slogger.Info("Starting database service with TLS", "port", *port, "db_dir", *dbDir, "cert", *certFile)

	if err := server.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
		slogger.Error(err.Error())
	}
}