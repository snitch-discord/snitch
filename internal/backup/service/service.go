package service

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"snitch/pkg/bucketclient"
	"snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
)

type BackupService struct {
	dbClient     snitchv1connect.DatabaseServiceClient
	bucketClient *bucketclient.Client
	tempDir      string
	logger       *slog.Logger
}

func NewBackupService(ctx context.Context, dbEndpoint string, logger *slog.Logger) (*BackupService, error) {
	dbServiceURL := "http://" + dbEndpoint
	logger.Info("Initializing backup service", "db_endpoint", dbEndpoint, "db_service_url", dbServiceURL)
	
	// Connect to database service
	dbClient := snitchv1connect.NewDatabaseServiceClient(
		&http.Client{Timeout: 30 * time.Second},
		dbServiceURL,
	)

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
	bucketClient, err := bucketclient.New(bucketConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket client: %w", err)
	}

	// Setup temp directory
	tempDir := getEnvOrDefault("BACKUP_TEMP_DIR", "/tmp/snitch-backups")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &BackupService{
		dbClient:     dbClient,
		bucketClient: bucketClient,
		tempDir:      tempDir,
		logger:       logger,
	}, nil
}

func (s *BackupService) Close() error {
	// Nothing to close for now
	return nil
}

func (s *BackupService) TriggerBackup(ctx context.Context, req *connect.Request[snitchv1.TriggerBackupRequest]) (*connect.Response[snitchv1.TriggerBackupResponse], error) {
	timestamp := time.Now().UTC().Format("2006-01-02-15-04")
	s.logger.Info("Starting backup", "timestamp", timestamp)

	// 1. Request DB service to create backup files
	s.logger.Info("Requesting database service to create backup files", "temp_dir", s.tempDir)
	createReq := connect.NewRequest(&snitchv1.CreateBackupFilesRequest{
		TempDir: s.tempDir,
	})

	createResp, err := s.dbClient.CreateBackupFiles(ctx, createReq)
	if err != nil {
		s.logger.Error("Failed to create backup files", "error", err)
		return nil, fmt.Errorf("failed to create backup files: %w", err)
	}

	files := createResp.Msg.Files
	s.logger.Info("Backup files created", "count", len(files))

	// 2. Upload each file to bucket
	var uploadedFiles []string
	var filePaths []string
	
	for _, file := range files {
		filePaths = append(filePaths, file.FilePath)
		
		// Compress and upload
		bucketKey := fmt.Sprintf("backups/%s/%s.db.gz", timestamp, file.DatabaseName)
		err = s.compressAndUpload(ctx, file.FilePath, bucketKey)
		if err != nil {
			s.logger.Error("Upload failed", "file", file.DatabaseName, "error", err)
			continue
		}
		
		uploadedFiles = append(uploadedFiles, file.DatabaseName)
		s.logger.Info("Backup uploaded", 
			"database", file.DatabaseName, 
			"size_mb", file.FileSize/(1024*1024),
			"key", bucketKey)
	}

	// 3. Cleanup temp files
	cleanupReq := connect.NewRequest(&snitchv1.CleanupBackupFilesRequest{
		FilePaths: filePaths,
	})
	
	_, err = s.dbClient.CleanupBackupFiles(ctx, cleanupReq)
	if err != nil {
		s.logger.Warn("Failed to cleanup backup files", "error", err)
	}

	s.logger.Info("Backup completed", 
		"timestamp", timestamp,
		"files_uploaded", len(uploadedFiles),
		"total_files", len(files))

	return connect.NewResponse(&snitchv1.TriggerBackupResponse{
		BackupTimestamp: timestamp,
		FilesBackedUp:   int32(len(uploadedFiles)),
		DatabaseNames:   uploadedFiles,
	}), nil
}

func (s *BackupService) compressAndUpload(ctx context.Context, filePath, bucketKey string) error {
	// Check source file first
	sourceInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	s.logger.Info("Starting compression and upload", 
		"source_path", filePath, 
		"source_size_bytes", sourceInfo.Size(),
		"bucket_key", bucketKey)

	// Create temporary compressed file
	tempCompressedPath := filePath + ".gz"
	defer os.Remove(tempCompressedPath) // Clean up temp file

	// Open source file
	sourceFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create compressed file
	compressedFile, err := os.Create(tempCompressedPath)
	if err != nil {
		return fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer compressedFile.Close()

	// Compress to file
	gzWriter := gzip.NewWriter(compressedFile)
	bytesRead, err := io.Copy(gzWriter, sourceFile)
	if err != nil {
		gzWriter.Close()
		return fmt.Errorf("compression failed: %w", err)
	}
	
	err = gzWriter.Close()
	if err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}
	
	err = compressedFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close compressed file: %w", err)
	}

	// Get compressed file size
	compressedInfo, err := os.Stat(tempCompressedPath)
	if err != nil {
		return fmt.Errorf("failed to stat compressed file: %w", err)
	}

	s.logger.Info("Compression completed", 
		"bytes_read_from_source", bytesRead,
		"expected_bytes", sourceInfo.Size(),
		"compressed_file_size", compressedInfo.Size())

	// Upload compressed file using direct file upload for better R2 compatibility
	s.logger.Info("Starting upload to bucket", 
		"bucket_key", bucketKey,
		"compressed_size_bytes", compressedInfo.Size())
	
	// Use UploadFile instead of Upload for better R2 compatibility (explicit content length)
	err = s.bucketClient.UploadFile(ctx, bucketKey, tempCompressedPath, "application/gzip")
	if err != nil {
		s.logger.Error("Upload to bucket failed", 
			"error", err,
			"bytes_compressed", bytesRead,
			"compressed_file_size", compressedInfo.Size())
		return fmt.Errorf("failed to upload to bucket: %w", err)
	}

	s.logger.Info("Upload completed successfully", 
		"bucket_key", bucketKey,
		"source_bytes", sourceInfo.Size(),
		"bytes_compressed", bytesRead,
		"compressed_file_size", compressedInfo.Size())

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}