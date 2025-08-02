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

	"snitch/internal/backend/service"
	"snitch/internal/backend/service/interceptor"
	"snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
	"github.com/robfig/cron/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	port := flag.Int("port", 4200, "port to listen on")
	flag.Parse()

	// Get database service connection details from environment
	dbHost := os.Getenv("SNITCH_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("SNITCH_DB_PORT")
	if dbPort == "" {
		dbPort = "5200"
	}

	// Create database service client (Connect RPC over HTTP)
	dbServiceURL := fmt.Sprintf("http://%s:%s", dbHost, dbPort)
	dbClient := snitchv1connect.NewDatabaseServiceClient(
		&http.Client{
			Timeout: 30 * time.Second,
		}, 
		dbServiceURL,
	)

	// Get backup service connection details from environment
	backupHost := os.Getenv("SNITCH_BACKUP_HOST")
	if backupHost == "" {
		backupHost = "localhost"
	}
	backupPort := os.Getenv("SNITCH_BACKUP_PORT")
	if backupPort == "" {
		backupPort = "5300"
	}

	// Create backup service client
	backupServiceURL := fmt.Sprintf("http://%s:%s", backupHost, backupPort)
	backupClient := snitchv1connect.NewBackupServiceClient(
		&http.Client{
			Timeout: 60 * time.Minute, // Longer timeout for backups
		},
		backupServiceURL,
	)

	eventService := service.NewEventService(dbClient)
	registrar := service.NewRegisterServer(dbClient)
	reportServer := service.NewReportServer(dbClient, eventService)
	userServer := service.NewUserServer(dbClient)

	// Initialize backup scheduler
	initBackupScheduler(backupClient)

	baseInterceptors := connect.WithInterceptors(
		interceptor.NewRecoveryInterceptor(),
		interceptor.NewLogInterceptor(),
		interceptor.NewTraceInterceptor(),
	)

	mux := http.NewServeMux()
	mux.Handle(snitchv1connect.NewRegistrarServiceHandler(registrar, baseInterceptors))
	mux.Handle(snitchv1connect.NewReportServiceHandler(reportServer, baseInterceptors))
	mux.Handle(snitchv1connect.NewUserHistoryServiceHandler(userServer, baseInterceptors))
	mux.Handle(snitchv1connect.NewEventServiceHandler(eventService, baseInterceptors))

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
		// No ReadTimeout/WriteTimeout for streaming support
	}

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		slog.Error(err.Error())
	}
}

func initBackupScheduler(backupClient snitchv1connect.BackupServiceClient) {
	backupSchedule := os.Getenv("BACKUP_SCHEDULE")
	if backupSchedule == "" {
		slog.Info("No backup schedule configured (BACKUP_SCHEDULE not set)")
		return
	}

	// Create cron scheduler
	c := cron.New()

	// Add backup job
	_, err := c.AddFunc(backupSchedule, func() {
		slog.Info("Starting scheduled backup")
		
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
		defer cancel()
		
		req := connect.NewRequest(&snitchv1.TriggerBackupRequest{})
		response, err := backupClient.TriggerBackup(ctx, req)
		if err != nil {
			slog.Error("Scheduled backup failed", "error", err)
		} else {
			slog.Info("Scheduled backup completed successfully",
				"timestamp", response.Msg.BackupTimestamp,
				"files_backed_up", response.Msg.FilesBackedUp,
				"databases", response.Msg.DatabaseNames)
		}
	})

	if err != nil {
		slog.Error("Failed to add backup job to scheduler", "error", err)
		return
	}

	// Start the scheduler
	c.Start()
	slog.Info("Backup scheduler started", "schedule", backupSchedule)
}
