package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"snitch/internal/db/migrations"
	"snitch/internal/db/sqlcgen/groupdb"
	"snitch/internal/db/sqlcgen/metadata"
	"snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"github.com/pressly/goose/v3"
	_ "github.com/tursodatabase/go-libsql"
)

// Migration directory paths within embedded filesystem
const (
	metadataMigrationsPath = "metadata"
	tenantMigrationsPath   = "tenant"
)


type DatabaseService struct {
	metadataDB     *sql.DB
	metadataQueries *metadata.Queries
	groupDBs       map[string]*sql.DB
	groupQueries   map[string]*groupdb.Queries
	groupDBMutex   sync.RWMutex
	dbDir          string
	logger         *slog.Logger
}

func NewDatabaseService(ctx context.Context, dbDir string, logger *slog.Logger) (*DatabaseService, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Initialize metadata database
	metadataPath := filepath.Join(dbDir, "metadata.db")
	metadataDB, err := sql.Open("libsql", "file:"+metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata database: %w", err)
	}

	// Configure metadata database with optimized PRAGMA settings
	if err := configureConnection(ctx, metadataDB, logger); err != nil {
		if closeErr := metadataDB.Close(); closeErr != nil {
			logger.Warn("Failed to close metadata database during error cleanup", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to configure metadata database: %w", err)
	}

	// Run metadata migrations
	if err := runMetadataMigrations(ctx, metadataDB, logger); err != nil {
		if closeErr := metadataDB.Close(); closeErr != nil {
			logger.Warn("Failed to close metadata database during error cleanup", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to run metadata migrations: %w", err)
	}

	service := &DatabaseService{
		metadataDB:      metadataDB,
		metadataQueries: metadata.New(metadataDB),
		groupDBs:        make(map[string]*sql.DB),
		groupQueries:    make(map[string]*groupdb.Queries),
		dbDir:           dbDir,
		logger:          logger,
	}

	return service, nil
}

func (s *DatabaseService) Close() error {
	s.groupDBMutex.Lock()
	defer s.groupDBMutex.Unlock()

	// Close all group databases
	for groupID, db := range s.groupDBs {
		if err := db.Close(); err != nil {
			s.logger.Error("Failed to close group database", "group_id", groupID, "error", err)
		}
	}

	// Close metadata database
	if err := s.metadataDB.Close(); err != nil {
		s.logger.Error("Failed to close metadata database", "error", err)
		return err
	}

	return nil
}

// runMetadataMigrations applies metadata database migrations using goose
func runMetadataMigrations(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	// Set goose to use the embedded migration files
	goose.SetBaseFS(migrations.MetadataMigrations)

	// Set SQLite dialect for goose
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Run migrations up to the latest version
	if err := goose.UpContext(ctx, db, metadataMigrationsPath); err != nil {
		return fmt.Errorf("failed to run metadata migrations: %w", err)
	}

	logger.Info("Successfully applied metadata migrations")
	return nil
}

// configureConnection applies Rails-inspired PRAGMA settings for optimal SQLite performance
func configureConnection(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	pragmas := []string{
		"PRAGMA foreign_keys=ON",              // Enable foreign key constraints for data integrity
		"PRAGMA journal_mode=WAL",             // Enable WAL mode for better concurrency
		"PRAGMA synchronous=NORMAL",           // Balance between safety and performance
		"PRAGMA mmap_size=134217728",          // 128MB memory mapping
		"PRAGMA journal_size_limit=67108864", // 64MB WAL file limit (triggers auto-checkpoint)
		"PRAGMA cache_size=2000",              // 2000 pages cache (~8MB with 4KB pages)
	}

	for _, pragma := range pragmas {
		// Use QueryContext for PRAGMA statements as they return results
		rows, err := db.QueryContext(ctx, pragma)
		if err != nil {
			// Log the error but continue with other pragmas
			logger.Warn("Failed to execute PRAGMA", "pragma", pragma, "error", err)
			
			// Only fail on critical pragmas
			if pragma == "PRAGMA foreign_keys=ON" || pragma == "PRAGMA journal_mode=WAL" {
				return fmt.Errorf("failed to execute critical pragma %q: %w", pragma, err)
			}
		} else {
			// Close the result set immediately
			if err := rows.Close(); err != nil {
				logger.Warn("Failed to close rows", "error", err)
			}
			logger.Debug("Applied PRAGMA successfully", "pragma", pragma)
		}
	}

	return nil
}

// getGroupDB returns an existing group database or error if it doesn't exist
func (s *DatabaseService) getGroupDB(ctx context.Context, groupID string) (*sql.DB, error) {
	s.groupDBMutex.RLock()
	defer s.groupDBMutex.RUnlock()
	
	if db, exists := s.groupDBs[groupID]; exists {
		return db, nil
	}
	
	return nil, fmt.Errorf("group database not found for group %s", groupID)
}

// createGroupDB explicitly creates a new group database
func (s *DatabaseService) createGroupDB(ctx context.Context, groupID string) (*sql.DB, error) {
	s.groupDBMutex.Lock()
	defer s.groupDBMutex.Unlock()

	// Check if it already exists
	if db, exists := s.groupDBs[groupID]; exists {
		return db, nil
	}

	// Create new group database
	groupPath := filepath.Join(s.dbDir, fmt.Sprintf("group_%s.db", groupID))
	db, err := sql.Open("libsql", "file:"+groupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open group database for %s: %w", groupID, err)
	}

	// Configure group database with optimized PRAGMA settings
	if err := configureConnection(ctx, db, s.logger); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			s.logger.Warn("Failed to close group database during error cleanup", "group_id", groupID, "error", closeErr)
		}
		return nil, fmt.Errorf("failed to configure group database for %s: %w", groupID, err)
	}

	// Run tenant migrations
	if err := s.runTenantMigrations(ctx, db, groupID); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			s.logger.Warn("Failed to close group database during error cleanup", "group_id", groupID, "error", closeErr)
		}
		return nil, fmt.Errorf("failed to run tenant migrations for %s: %w", groupID, err)
	}

	s.groupDBs[groupID] = db
	s.groupQueries[groupID] = groupdb.New(db)
	s.logger.Info("Created new group database", "group_id", groupID)

	return db, nil
}

// runTenantMigrations applies tenant database migrations using goose
func (s *DatabaseService) runTenantMigrations(ctx context.Context, db *sql.DB, groupID string) error {
	// Set goose to use the embedded migration files
	goose.SetBaseFS(migrations.TenantMigrations)

	// Set SQLite dialect for goose
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Run migrations up to the latest version
	if err := goose.UpContext(ctx, db, tenantMigrationsPath); err != nil {
		return fmt.Errorf("failed to run tenant migrations for group %s: %w", groupID, err)
	}

	s.logger.Info("Successfully applied tenant migrations", "group_id", groupID)
	return nil
}

// RunMigrationsOnAllTenants discovers and migrates all existing tenant databases
func (s *DatabaseService) RunMigrationsOnAllTenants(ctx context.Context) error {
	// Find all existing tenant database files
	files, err := filepath.Glob(filepath.Join(s.dbDir, "group_*.db"))
	if err != nil {
		return fmt.Errorf("failed to discover tenant databases: %w", err)
	}

	for _, file := range files {
		// Extract group ID from filename
		basename := filepath.Base(file)
		if !strings.HasPrefix(basename, "group_") || !strings.HasSuffix(basename, ".db") {
			continue
		}
		groupID := strings.TrimSuffix(strings.TrimPrefix(basename, "group_"), ".db")

		// Open database connection
		db, err := sql.Open("libsql", "file:"+file)
		if err != nil {
			s.logger.Error("Failed to open tenant database for migration", "group_id", groupID, "error", err)
			continue
		}

		// Configure connection
		if err := configureConnection(ctx, db, s.logger); err != nil {
			s.logger.Error("Failed to configure tenant database connection", "group_id", groupID, "error", err)
			if closeErr := db.Close(); closeErr != nil {
				s.logger.Error("Failed to close tenant database after configuration error", "group_id", groupID, "error", closeErr)
			}
			continue
		}

		// Run migrations
		if err := s.runTenantMigrations(ctx, db, groupID); err != nil {
			s.logger.Error("Failed to migrate tenant database", "group_id", groupID, "error", err)
			if closeErr := db.Close(); closeErr != nil {
				s.logger.Error("Failed to close tenant database after migration error", "group_id", groupID, "error", closeErr)
			}
			continue
		}

		// Store the connection for future use
		s.groupDBMutex.Lock()
		s.groupDBs[groupID] = db
		s.groupQueries[groupID] = groupdb.New(db)
		s.groupDBMutex.Unlock()

		s.logger.Info("Successfully migrated tenant database", "group_id", groupID)
	}

	return nil
}





// Backup file creation methods

func (s *DatabaseService) CreateBackupFiles(ctx context.Context, req *connect.Request[snitchv1.CreateBackupFilesRequest]) (*connect.Response[snitchv1.CreateBackupFilesResponse], error) {
	tempDir := req.Msg.TempDir
	timestamp := time.Now().UTC().Format("2006-01-02-15-04")
	
	s.logger.Info("Creating backup files", "temp_dir", tempDir, "timestamp", timestamp)
	
	// Ensure the temp directory exists (it should be mounted as a shared volume)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	
	var backupFiles []*snitchv1.BackupFile
	
	// Backup metadata database
	metadataFile := fmt.Sprintf("metadata_%s.db", timestamp)
	metadataPath := filepath.Join(tempDir, metadataFile)
	
	err := s.createDatabaseBackup(ctx, s.metadataDB, metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to backup metadata database: %w", err)
	}
	
	fileInfo, err := os.Stat(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat metadata backup: %w", err)
	}
	
	checksum, err := s.calculateFileChecksum(metadataPath)
	if err != nil {
		s.logger.Warn("Failed to calculate checksum for metadata", "error", err)
	}
	
	backupFiles = append(backupFiles, &snitchv1.BackupFile{
		DatabaseName: "metadata",
		FilePath:     metadataPath,
		FileSize:     fileInfo.Size(),
		Checksum:     checksum,
	})
	
	s.logger.Info("Created metadata backup", "path", metadataPath, "size_mb", fileInfo.Size()/(1024*1024))
	
	// Backup all group databases
	s.groupDBMutex.RLock()
	for groupID, groupDB := range s.groupDBs {
		groupFile := fmt.Sprintf("group_%s_%s.db", groupID, timestamp)
		groupPath := filepath.Join(tempDir, groupFile)
		
		err := s.createDatabaseBackup(ctx, groupDB, groupPath)
		if err != nil {
			s.logger.Error("Failed to backup group database", "group_id", groupID, "error", err)
			continue
		}
		
		fileInfo, err := os.Stat(groupPath)
		if err != nil {
			s.logger.Error("Failed to stat group backup", "group_id", groupID, "error", err)
			continue
		}
		
		checksum, err := s.calculateFileChecksum(groupPath)
		if err != nil {
			s.logger.Warn("Failed to calculate checksum for group", "group_id", groupID, "error", err)
		}
		
		backupFiles = append(backupFiles, &snitchv1.BackupFile{
			DatabaseName: fmt.Sprintf("group_%s", groupID),
			FilePath:     groupPath,
			FileSize:     fileInfo.Size(),
			Checksum:     checksum,
		})
		
		s.logger.Info("Created group backup", "group_id", groupID, "path", groupPath, "size_mb", fileInfo.Size()/(1024*1024))
	}
	s.groupDBMutex.RUnlock()
	
	s.logger.Info("All backup files created", "count", len(backupFiles))
	
	return connect.NewResponse(&snitchv1.CreateBackupFilesResponse{
		Files: backupFiles,
	}), nil
}

func (s *DatabaseService) CleanupBackupFiles(ctx context.Context, req *connect.Request[snitchv1.CleanupBackupFilesRequest]) (*connect.Response[emptypb.Empty], error) {
	filePaths := req.Msg.FilePaths
	
	s.logger.Info("Cleaning up backup files", "count", len(filePaths))
	
	for _, filePath := range filePaths {
		if err := os.Remove(filePath); err != nil {
			s.logger.Warn("Failed to remove backup file", "path", filePath, "error", err)
		} else {
			s.logger.Debug("Removed backup file", "path", filePath)
		}
	}
	
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *DatabaseService) createDatabaseBackup(ctx context.Context, db *sql.DB, backupPath string) error {
	// Check source database size first
	var pageCount, pageSize int
	err := db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		s.logger.Warn("Failed to get page count", "error", err)
	}
	err = db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		s.logger.Warn("Failed to get page size", "error", err)
	}
	
	sourceSize := int64(pageCount * pageSize)
	s.logger.Info("Source database info", 
		"backup_path", backupPath,
		"page_count", pageCount, 
		"page_size", pageSize, 
		"estimated_size_bytes", sourceSize)
	
	// Use VACUUM INTO for atomic backup
	s.logger.Info("Starting VACUUM INTO", "target_path", backupPath)
	_, err = db.ExecContext(ctx, "VACUUM INTO ?", backupPath)
	if err != nil {
		return fmt.Errorf("VACUUM INTO failed: %w", err)
	}
	
	// Check if backup file was created and has expected size
	if stat, err := os.Stat(backupPath); err == nil {
		s.logger.Info("Backup file created", 
			"path", backupPath, 
			"actual_size_bytes", stat.Size(),
			"expected_size_bytes", sourceSize)
	} else {
		s.logger.Error("Backup file not found after VACUUM INTO", "path", backupPath, "error", err)
	}
	
	return nil
}

func (s *DatabaseService) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}