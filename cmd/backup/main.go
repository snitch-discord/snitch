package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"snitch/internal/backup/backupconfig"
	"snitch/internal/backup/service"

	"github.com/robfig/cron/v3"
)

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func main() {
	config, err := backupconfig.FromEnv()
	if err != nil {
		fatal("Failed to load backup configuration from environment", "error", err)
	}

	slogger := slog.Default()
	ctx := context.Background()

	// Load CA certificate for database service validation
	caCert, err := os.ReadFile(config.CaCertFilePath)
	if err != nil {
		fatal("Failed to read CA certificate", "error", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		fatal("Failed to parse CA certificate")
	}

	// Create TLS-enabled HTTP client for database service
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	// Initialize backup service with TLS-configured client
	backupService, err := service.NewBackupService(config.DatabaseServiceURL, httpClient, slogger)
	if err != nil {
		fatal("Failed to initialize backup service", "error", err)
	}
	defer func() {
		if err := backupService.Close(); err != nil {
			slogger.Error("Failed to close backup service", "error", err)
		}
	}()

	c := cron.New()

	slogger.Info("Scheduling backup job", "schedule", config.CronSchedule, "database_service_url", config.DatabaseServiceURL)

	_, err = c.AddFunc(config.CronSchedule, func() {
		slogger.Info("Starting scheduled backup")
		if err := backupService.PerformBackup(ctx); err != nil {
			slogger.Error("Scheduled backup failed", "error", err)
		} else {
			slogger.Info("Scheduled backup completed successfully")
		}
	})
	if err != nil {
		fatal("Failed to schedule backup job", "error", err)
	}

	c.Start()
	slogger.Info("Backup service started", "schedule", config.CronSchedule)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	slogger.Info("Received shutdown signal, stopping backup service")

	stopCtx := c.Stop()
	<-stopCtx.Done()

	slogger.Info("Backup service stopped")
}
