package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	"github.com/andybalholm/brotli"
	"snitch/pkg/bucketclient"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"
)

type DatabaseInfo struct {
	Name    string
	GroupID string
}

type BackupService struct {
	bucketClient *bucketclient.Client
	dbClient     snitchv1connect.DatabaseServiceClient
	tempDir      string
	logger       *slog.Logger
}

func NewBackupService(dbServiceURL string, httpClient *http.Client, logger *slog.Logger) (*BackupService, error) {
	logger.Info("Initializing backup service", "db_service_url", dbServiceURL)

	// Get bucket configuration
	bucketConfig := bucketclient.Config{
		Endpoint:        os.Getenv("BACKUP_BUCKET_ENDPOINT"),
		Region:          getEnvOrDefault("BACKUP_BUCKET_REGION", "auto"),
		Bucket:          os.Getenv("BACKUP_BUCKET_NAME"),
		AccessKeyID:     os.Getenv("BACKUP_BUCKET_ACCESS_KEY"),
		SecretAccessKey: os.Getenv("BACKUP_BUCKET_SECRET_KEY"),
	}

	// Validate required fields
	if bucketConfig.Endpoint == "" {
		return nil, fmt.Errorf("BACKUP_BUCKET_ENDPOINT is required")
	}
	if bucketConfig.Bucket == "" {
		return nil, fmt.Errorf("BACKUP_BUCKET_NAME is required")
	}
	if bucketConfig.AccessKeyID == "" {
		return nil, fmt.Errorf("BACKUP_BUCKET_ACCESS_KEY is required")
	}
	if bucketConfig.SecretAccessKey == "" {
		return nil, fmt.Errorf("BACKUP_BUCKET_SECRET_KEY is required")
	}

	// Create bucket client
	bucketClient, err := bucketclient.New(context.Background(), bucketConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket client: %w", err)
	}

	// Create database service client with provided HTTP client
	dbClient := snitchv1connect.NewDatabaseServiceClient(
		httpClient,
		dbServiceURL,
	)

	// Setup temp directory
	tempDir := getEnvOrDefault("BACKUP_TEMP_DIR", "/tmp/snitch-backups")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &BackupService{
		bucketClient: bucketClient,
		dbClient:     dbClient,
		tempDir:      tempDir,
		logger:       logger,
	}, nil
}

func (s *BackupService) Close() error {
	// Nothing to close for now
	return nil
}

func (s *BackupService) PerformBackup(ctx context.Context) error {
	timestamp := time.Now().UTC().Format("2006-01-02-15-04")
	s.logger.Info("Starting backup", "timestamp", timestamp)

	// Get list of group IDs from database service
	groupIDsResp, err := s.dbClient.ListGroupIDs(ctx, connect.NewRequest(&snitchv1.ListGroupIDsRequest{}))
	if err != nil {
		return fmt.Errorf("failed to list group IDs: %w", err)
	}

	// Add metadata database to backup list
	databases := []DatabaseInfo{
		{Name: "metadata", GroupID: ""}, // Empty group ID for metadata
	}

	// Add all group databases
	for _, groupID := range groupIDsResp.Msg.GroupIds {
		databases = append(databases, DatabaseInfo{
			Name:    fmt.Sprintf("group_%s", groupID),
			GroupID: groupID,
		})
	}

	s.logger.Info("Found databases to backup", "count", len(databases))

	var uploadedDatabases []string

	for _, db := range databases {
		if err := s.backupSingleDatabase(ctx, db, timestamp, &uploadedDatabases); err != nil {
			s.logger.Error("Database backup failed", "database", db.Name, "error", err)
		}
	}

	s.logger.Info("Backup completed",
		"timestamp", timestamp,
		"uploaded_databases", len(uploadedDatabases),
		"total_databases", len(databases))
	return nil
}

func (s *BackupService) backupSingleDatabase(ctx context.Context, db DatabaseInfo, timestamp string, uploadedDatabases *[]string) error {
	s.logger.Info("Backing up database", "database", db.Name, "group_id", db.GroupID)

	// Create safe backup using database service
	tempBackupPath := filepath.Join(s.tempDir, fmt.Sprintf("%s_backup.db", db.Name))
	tempCompressedPath := filepath.Join(s.tempDir, fmt.Sprintf("%s_backup.br", db.Name))

	// Ensure cleanup happens even on panic or early return
	defer func() {
		s.cleanupTempFiles(tempBackupPath, tempCompressedPath)
	}()

	var groupIDPtr *string
	if db.GroupID != "" {
		groupIDPtr = &db.GroupID
	}
	// For metadata database, groupIDPtr remains nil

	backupReq := &snitchv1.CreateBackupRequest{
		BackupPath: tempBackupPath,
		GroupId:    groupIDPtr, // nil for metadata, pointer to string for groups
	}

	backupResp, err := s.dbClient.CreateBackup(ctx, connect.NewRequest(backupReq))
	if err != nil {
		return fmt.Errorf("database backup failed: %w", err)
	}

	s.logger.Info("Database backup created",
		"database", db.Name,
		"size_bytes", backupResp.Msg.SizeBytes,
		"path", backupResp.Msg.BackupPath)

	// Compress with Brotli
	start := time.Now()
	err = s.compressBrotli(tempBackupPath, tempCompressedPath)
	if err != nil {
		return fmt.Errorf("brotli compression failed for database %s: %w", db.Name, err)
	}
	compressionTime := time.Since(start)

	// Get compressed file size
	compressedInfo, err := os.Stat(tempCompressedPath)
	if err != nil {
		return fmt.Errorf("failed to stat compressed file for database %s: %w", db.Name, err)
	}

	s.logger.Info("Database compressed",
		"database", db.Name,
		"original_size_bytes", backupResp.Msg.SizeBytes,
		"compressed_size_bytes", compressedInfo.Size(),
		"compression_time", compressionTime)

	// Upload compressed file
	bucketKey := fmt.Sprintf("backups/%s/%s.br", timestamp, db.Name)
	err = s.bucketClient.UploadFile(ctx, bucketKey, tempCompressedPath, "application/x-brotli")
	if err != nil {
		return fmt.Errorf("upload failed for database %s: %w", db.Name, err)
	}

	s.logger.Info("Database backup uploaded",
		"database", db.Name,
		"compression", "brotli",
		"size_bytes", compressedInfo.Size(),
		"compression_time", compressionTime,
		"key", bucketKey)

	*uploadedDatabases = append(*uploadedDatabases, db.Name)
	return nil
}

func (s *BackupService) compressBrotli(inputPath, outputPath string) error {
	// Open input file
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer func() {
		if closeErr := inputFile.Close(); closeErr != nil {
			s.logger.Error("Failed to close input file", "error", closeErr)
		}
	}()

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if closeErr := outputFile.Close(); closeErr != nil {
			s.logger.Error("Failed to close output file", "error", closeErr)
		}
	}()

	// Create brotli writer
	brotliWriter := brotli.NewWriter(outputFile)
	defer func() {
		if closeErr := brotliWriter.Close(); closeErr != nil {
			s.logger.Error("Failed to close brotli writer", "error", closeErr)
		}
	}()

	// Copy and compress with fixed buffer to limit memory usage
	buffer := make([]byte, 64*1024) // 64KB buffer
	_, err = io.CopyBuffer(brotliWriter, inputFile, buffer)
	if err != nil {
		return fmt.Errorf("brotli compression failed: %w", err)
	}

	// Close brotli writer
	if err := brotliWriter.Close(); err != nil {
		return fmt.Errorf("failed to close brotli writer: %w", err)
	}

	if err := outputFile.Close(); err != nil {
		return fmt.Errorf("failed to close output file: %w", err)
	}

	return nil
}

func (s *BackupService) cleanupTempFiles(backupPath, compressedPath string) {
	// Remove backup file
	if backupPath != "" {
		if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
			s.logger.Warn("Failed to cleanup backup file", "path", backupPath, "error", err)
		}
	}

	// Remove compressed file
	if compressedPath != "" {
		if err := os.Remove(compressedPath); err != nil && !os.IsNotExist(err) {
			s.logger.Warn("Failed to cleanup compressed file", "path", compressedPath, "error", err)
		}
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
