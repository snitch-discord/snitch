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

	"snitch/internal/db/service"
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
	port := flag.Int("port", 5200, "port to listen on")
	dbDir := flag.String("db-dir", "./data", "directory to store database files")
	flag.Parse()

	slogger := slog.Default()
	ctx := context.Background()

	// Initialize database service
	dbService, err := service.NewDatabaseService(ctx, *dbDir, slogger)
	if err != nil {
		fatal("Failed to initialize database service", "error", err)
	}
	defer dbService.Close()

	// Setup gRPC handlers
	mux := http.NewServeMux()
	mux.Handle(snitchv1connect.NewDatabaseServiceHandler(dbService, connect.WithInterceptors()))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slogger.Info("Starting database service", "port", *port, "db_dir", *dbDir)

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slogger.Error(err.Error())
	}
}