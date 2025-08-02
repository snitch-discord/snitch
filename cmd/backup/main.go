package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"snitch/internal/backup/service"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func main() {
	port := flag.Int("port", 5300, "port to listen on")
	dbServiceEndpoint := flag.String("db-endpoint", "localhost:5200", "database service endpoint")
	flag.Parse()

	slogger := slog.Default()
	ctx := context.Background()

	// Get database service connection details from environment (Docker-first, flag fallback)
	dbHost := os.Getenv("SNITCH_DB_HOST")
	dbPort := os.Getenv("SNITCH_DB_PORT")
	
	var finalEndpoint string
	if dbHost != "" {
		if dbPort == "" {
			dbPort = "5200"
		}
		finalEndpoint = fmt.Sprintf("%s:%s", dbHost, dbPort)
		slogger.Info("Using database endpoint from environment", "endpoint", finalEndpoint)
	} else {
		finalEndpoint = *dbServiceEndpoint
		slogger.Info("Using database endpoint from flag", "endpoint", finalEndpoint)
	}

	// Initialize backup service
	backupService, err := service.NewBackupService(ctx, finalEndpoint, slogger)
	if err != nil {
		fatal("Failed to initialize backup service", "error", err)
	}
	defer func() {
		if err := backupService.Close(); err != nil {
			slogger.Error("Failed to close backup service", "error", err)
		}
	}()

	// Setup gRPC handlers
	mux := http.NewServeMux()
	mux.Handle(snitchv1connect.NewBackupServiceHandler(backupService, connect.WithInterceptors()))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slogger.Info("Starting backup service", "port", *port, "db_endpoint", *dbServiceEndpoint)

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slogger.Error(err.Error())
	}
}