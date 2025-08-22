package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"
	_ "github.com/tursodatabase/go-libsql"
)

const (
	TEST_GROUP_ID  = "test-group-id"
	TEST_SERVER_ID = "test-server-id"
)

// Mock database service client for testing
type mockDatabaseServiceClient struct {
	groupIDs       []string
	groupIDsError  error
	backupError    error
	backupResponse *snitchv1.CreateBackupResponse
}

// Ensure mockDatabaseServiceClient implements the interface
var _ snitchv1connect.DatabaseServiceClient = (*mockDatabaseServiceClient)(nil)

func (m *mockDatabaseServiceClient) ListGroupIDs(ctx context.Context, req *connect.Request[snitchv1.ListGroupIDsRequest]) (*connect.Response[snitchv1.ListGroupIDsResponse], error) {
	if m.groupIDsError != nil {
		return nil, m.groupIDsError
	}
	resp := &snitchv1.ListGroupIDsResponse{
		GroupIds: m.groupIDs,
	}
	return connect.NewResponse(resp), nil
}

func (m *mockDatabaseServiceClient) CreateBackup(ctx context.Context, req *connect.Request[snitchv1.CreateBackupRequest]) (*connect.Response[snitchv1.CreateBackupResponse], error) {
	if m.backupError != nil {
		return nil, m.backupError
	}
	
	// Create a test backup file
	backupPath := req.Msg.BackupPath
	testData := "test backup data"
	err := os.WriteFile(backupPath, []byte(testData), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create test backup file: %w", err)
	}
	
	response := m.backupResponse
	if response == nil {
		response = &snitchv1.CreateBackupResponse{
			BackupPath: backupPath,
			SizeBytes:  int64(len(testData)),
		}
	}
	return connect.NewResponse(response), nil
}

// Implement other required methods as no-ops for this test
func (m *mockDatabaseServiceClient) CreateGroup(ctx context.Context, req *connect.Request[snitchv1.CreateGroupRequest]) (*connect.Response[snitchv1.CreateGroupResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) FindGroupByServer(ctx context.Context, req *connect.Request[snitchv1.FindGroupByServerRequest]) (*connect.Response[snitchv1.FindGroupByServerResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) AddServerToGroup(ctx context.Context, req *connect.Request[snitchv1.AddServerToGroupRequest]) (*connect.Response[snitchv1.AddServerToGroupResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) CreateGroupDatabase(ctx context.Context, req *connect.Request[snitchv1.CreateGroupDatabaseRequest]) (*connect.Response[snitchv1.CreateGroupDatabaseResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) CreateReport(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceCreateReportRequest]) (*connect.Response[snitchv1.DatabaseServiceCreateReportResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) GetReport(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceGetReportRequest]) (*connect.Response[snitchv1.DatabaseServiceGetReportResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) ListReports(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceListReportsRequest]) (*connect.Response[snitchv1.DatabaseServiceListReportsResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) DeleteReport(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceDeleteReportRequest]) (*connect.Response[snitchv1.DatabaseServiceDeleteReportResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) CreateUserHistory(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceCreateUserHistoryRequest]) (*connect.Response[snitchv1.DatabaseServiceCreateUserHistoryResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) GetUserHistory(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceGetUserHistoryRequest]) (*connect.Response[snitchv1.DatabaseServiceGetUserHistoryResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockDatabaseServiceClient) ListServers(ctx context.Context, req *connect.Request[snitchv1.ListServersRequest]) (*connect.Response[snitchv1.ListServersResponse], error) {
	return nil, fmt.Errorf("not implemented")
}

// Helper function to create a test file with content
func createTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return filePath
}

// Helper function to create a realistic SQLite database file
func createTestDatabase(t *testing.T, dir, filename string, populateData bool) string {
	dbPath := filepath.Join(dir, filename)
	
	// Create database using libsql like the actual service
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("Failed to close test database: %v", closeErr)
		}
	}()
	
	// Create realistic database schema similar to the actual service
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			user_id TEXT NOT NULL,
			server_id TEXT NOT NULL,
			username TEXT,
			discriminator TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			reporter_id TEXT NOT NULL,
			server_id TEXT NOT NULL,
			reason TEXT NOT NULL,
			evidence_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS user_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			server_id TEXT NOT NULL,
			action TEXT NOT NULL,
			reason TEXT,
			evidence_url TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	
	// Create tables
	for _, schema := range schemas {
		_, err = db.Exec(schema)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	}
	
	// Populate with realistic data if requested
	if populateData {
		// Insert many users (repetitive data that should compress well)
		for i := 0; i < 1000; i++ {
			_, err = db.Exec(`INSERT INTO users (user_id, server_id, username, discriminator) 
				VALUES (?, ?, ?, ?)`, 
				fmt.Sprintf("user_%d", i),
				fmt.Sprintf("server_%d", i%10), // Repeat server IDs for better compression
				fmt.Sprintf("username_%d", i),
				fmt.Sprintf("%04d", i%10000))
			if err != nil {
				t.Fatalf("Failed to insert user data: %v", err)
			}
		}
		
		// Insert reports (more repetitive structured data)
		for i := 0; i < 500; i++ {
			_, err = db.Exec(`INSERT INTO reports (user_id, reporter_id, server_id, reason, evidence_url) 
				VALUES (?, ?, ?, ?, ?)`,
				fmt.Sprintf("user_%d", i%100),
				fmt.Sprintf("reporter_%d", i%50),
				fmt.Sprintf("server_%d", i%10),
				"Inappropriate behavior in chat", // Repetitive reasons
				fmt.Sprintf("https://discord.com/evidence_%d.png", i))
			if err != nil {
				t.Fatalf("Failed to insert report data: %v", err)
			}
		}
		
		// Insert user history
		for i := 0; i < 300; i++ {
			_, err = db.Exec(`INSERT INTO user_history (user_id, server_id, action, reason) 
				VALUES (?, ?, ?, ?)`,
				fmt.Sprintf("user_%d", i%100),
				fmt.Sprintf("server_%d", i%10),
				"banned", // Repetitive actions
				"Violation of community guidelines")
			if err != nil {
				t.Fatalf("Failed to insert history data: %v", err)
			}
		}
	}
	
	return dbPath
}


func TestBackupService_compressBrotli(t *testing.T) {
	// Create a minimal service for testing compression functionality
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := &BackupService{
		bucketClient: nil, // Not needed for compression tests
		dbClient:     nil, // Not needed for compression tests
		tempDir:      tempDir,
		logger:       logger,
	}

	tests := []struct {
		name         string
		createInput  func(t *testing.T, dir string) string
		expectError  bool
		shouldShrink bool // Should compression make it smaller?
	}{
		{
			name: "populated database compression",
			createInput: func(t *testing.T, dir string) string {
				return createTestDatabase(t, dir, "populated.db", true)
			},
			shouldShrink: true, // Database with repetitive data should compress well
		},
		{
			name: "empty database compression",
			createInput: func(t *testing.T, dir string) string {
				return createTestDatabase(t, dir, "empty.db", false)
			},
			shouldShrink: false, // Empty database might not compress much
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create input database using the test-specific function
			inputPath := tt.createInput(t, tempDir)
			outputPath := filepath.Join(tempDir, "output.br")
			
			err := service.compressBrotli(inputPath, outputPath)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}
			
			// Check that output file exists
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("Expected output file to be created")
				return
			}
			
			// Analyze compression results
			inputInfo, _ := os.Stat(inputPath)
			outputInfo, _ := os.Stat(outputPath)
			
			// Compressed file should not be empty (unless input was empty)
			if outputInfo.Size() == 0 && inputInfo.Size() > 0 {
				t.Errorf("Compressed file should not be empty when input has content")
				return
			}
			
			// Calculate and log compression ratio
			compressionRatio := float64(outputInfo.Size()) / float64(inputInfo.Size())
			t.Logf("%s: %d -> %d bytes (ratio: %.2f%%)", 
				tt.name, inputInfo.Size(), outputInfo.Size(), compressionRatio*100)
			
			// Check compression effectiveness based on expectations
			if tt.shouldShrink {
				if outputInfo.Size() >= inputInfo.Size() {
					t.Errorf("Expected compression to reduce file size: input=%d bytes, output=%d bytes (ratio=%.2f%%)", 
						inputInfo.Size(), outputInfo.Size(), compressionRatio*100)
				}
				
				// For databases with repetitive data, expect good compression
				if compressionRatio > 0.8 {
					t.Logf("Warning: Compression ratio %.2f%% is higher than expected for database with repetitive data", compressionRatio*100)
				}
			} else {
				// For empty/small databases, just ensure compression doesn't blow up the size
				if outputInfo.Size() > inputInfo.Size()*3 {
					t.Errorf("Compression overhead seems excessive: input=%d bytes, output=%d bytes (ratio=%.2f%%)", 
						inputInfo.Size(), outputInfo.Size(), compressionRatio*100)
				}
			}
		})
	}
}

func TestBackupService_compressBrotli_Errors(t *testing.T) {
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := &BackupService{
		bucketClient: nil,
		dbClient:     nil,
		tempDir:      tempDir,
		logger:       logger,
	}
	
	t.Run("input file does not exist", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "nonexistent.txt")
		outputPath := filepath.Join(tempDir, "output.br")
		
		err := service.compressBrotli(nonExistentPath, outputPath)
		if err == nil {
			t.Errorf("Expected error for non-existent input file")
		}
	})
	
	t.Run("output directory does not exist", func(t *testing.T) {
		inputPath := createTestFile(t, tempDir, "input.txt", "test data")
		outputPath := filepath.Join(tempDir, "nonexistent", "output.br")
		
		err := service.compressBrotli(inputPath, outputPath)
		if err == nil {
			t.Errorf("Expected error for non-existent output directory")
		}
	})
}

func TestBackupService_cleanupTempFiles(t *testing.T) {
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := &BackupService{
		bucketClient: nil,
		dbClient:     nil,
		tempDir:      tempDir,
		logger:       logger,
	}
	
	t.Run("cleanup existing files", func(t *testing.T) {
		backupPath := createTestFile(t, tempDir, "backup.db", "backup data")
		compressedPath := createTestFile(t, tempDir, "backup.br", "compressed data")
		
		// Files should exist before cleanup
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Fatalf("Backup file should exist before cleanup")
		}
		if _, err := os.Stat(compressedPath); os.IsNotExist(err) {
			t.Fatalf("Compressed file should exist before cleanup")
		}
		
		service.cleanupTempFiles(backupPath, compressedPath)
		
		// Files should not exist after cleanup
		if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
			t.Errorf("Backup file should be cleaned up")
		}
		if _, err := os.Stat(compressedPath); !os.IsNotExist(err) {
			t.Errorf("Compressed file should be cleaned up")
		}
	})
	
	t.Run("cleanup non-existent files", func(t *testing.T) {
		nonExistentBackup := filepath.Join(tempDir, "nonexistent_backup.db")
		nonExistentCompressed := filepath.Join(tempDir, "nonexistent_compressed.br")
		
		// This should not panic or return error
		service.cleanupTempFiles(nonExistentBackup, nonExistentCompressed)
	})
	
	t.Run("cleanup empty paths", func(t *testing.T) {
		// This should not panic or return error
		service.cleanupTempFiles("", "")
	})
}

func TestBackupService_Close(t *testing.T) {
	tempDir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := &BackupService{
		bucketClient: nil,
		dbClient:     nil,
		tempDir:      tempDir,
		logger:       logger,
	}
	
	err := service.Close()
	if err != nil {
		t.Errorf("Expected no error from Close(), got: %v", err)
	}
}