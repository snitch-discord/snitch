package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"snitch/internal/db/migrations"
	"snitch/internal/db/sqlc/gen/groupdb"
	"snitch/internal/db/sqlc/gen/metadata"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
	"github.com/pressly/goose/v3"
	_ "github.com/tursodatabase/go-libsql"
)

// Migration directory paths within embedded filesystem
const (
	metadataMigrationsPath = "metadata"
	tenantMigrationsPath   = "tenant"
)

type DatabaseService struct {
	metadataDB      *sql.DB
	metadataQueries *metadata.Queries
	groupDBs        map[string]*sql.DB
	groupQueries    map[string]*groupdb.Queries
	groupDBMutex    sync.RWMutex
	dbDir           string
	logger          *slog.Logger

	// Repository pattern
	GroupRepository  *GroupRepository
	ReportRepository *ReportRepository
	UserRepository   *UserRepository
	ServerRepository *ServerRepository
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

	// Initialize repositories
	service.GroupRepository = NewGroupRepository(service)
	service.ReportRepository = NewReportRepository(service)
	service.UserRepository = NewUserRepository(service)
	service.ServerRepository = NewServerRepository(service)

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
		"PRAGMA foreign_keys=ON",             // Enable foreign key constraints for data integrity
		"PRAGMA journal_mode=WAL",            // Enable WAL mode for better concurrency
		"PRAGMA synchronous=NORMAL",          // Balance between safety and performance
		"PRAGMA mmap_size=134217728",         // 128MB memory mapping
		"PRAGMA journal_size_limit=67108864", // 64MB WAL file limit (triggers auto-checkpoint)
		"PRAGMA cache_size=2000",             // 2000 pages cache (~8MB with 4KB pages)
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
func (s *DatabaseService) getGroupDB(_ context.Context, groupID string) (*sql.DB, error) {
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

// Delegation methods for gRPC endpoints - forward to appropriate repositories

// Group operations
func (s *DatabaseService) CreateGroupDatabase(ctx context.Context, req *connect.Request[snitchv1.CreateGroupDatabaseRequest]) (*connect.Response[snitchv1.CreateGroupDatabaseResponse], error) {
	return s.GroupRepository.CreateGroupDatabase(ctx, req)
}

// Report operations
func (s *DatabaseService) CreateReport(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceCreateReportRequest]) (*connect.Response[snitchv1.DatabaseServiceCreateReportResponse], error) {
	return s.ReportRepository.CreateReport(ctx, req)
}

func (s *DatabaseService) GetReport(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceGetReportRequest]) (*connect.Response[snitchv1.DatabaseServiceGetReportResponse], error) {
	return s.ReportRepository.GetReport(ctx, req)
}

func (s *DatabaseService) ListReports(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceListReportsRequest]) (*connect.Response[snitchv1.DatabaseServiceListReportsResponse], error) {
	return s.ReportRepository.ListReports(ctx, req)
}

func (s *DatabaseService) DeleteReport(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceDeleteReportRequest]) (*connect.Response[snitchv1.DatabaseServiceDeleteReportResponse], error) {
	return s.ReportRepository.DeleteReport(ctx, req)
}

// User operations
func (s *DatabaseService) CreateUserHistory(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceCreateUserHistoryRequest]) (*connect.Response[snitchv1.DatabaseServiceCreateUserHistoryResponse], error) {
	return s.UserRepository.CreateUserHistory(ctx, req)
}

func (s *DatabaseService) GetUserHistory(ctx context.Context, req *connect.Request[snitchv1.DatabaseServiceGetUserHistoryRequest]) (*connect.Response[snitchv1.DatabaseServiceGetUserHistoryResponse], error) {
	return s.UserRepository.GetUserHistory(ctx, req)
}

// Server and metadata operations
func (s *DatabaseService) CreateGroup(ctx context.Context, req *connect.Request[snitchv1.CreateGroupRequest]) (*connect.Response[snitchv1.CreateGroupResponse], error) {
	return s.ServerRepository.CreateGroup(ctx, req)
}

func (s *DatabaseService) FindGroupByServer(ctx context.Context, req *connect.Request[snitchv1.FindGroupByServerRequest]) (*connect.Response[snitchv1.FindGroupByServerResponse], error) {
	return s.ServerRepository.FindGroupByServer(ctx, req)
}

func (s *DatabaseService) AddServerToGroup(ctx context.Context, req *connect.Request[snitchv1.AddServerToGroupRequest]) (*connect.Response[snitchv1.AddServerToGroupResponse], error) {
	return s.ServerRepository.AddServerToGroup(ctx, req)
}

func (s *DatabaseService) ListServers(ctx context.Context, req *connect.Request[snitchv1.ListServersRequest]) (*connect.Response[snitchv1.ListServersResponse], error) {
	return s.ServerRepository.ListServers(ctx, req)
}

// Backup operations
func (s *DatabaseService) CreateMetadataBackup(ctx context.Context, backupPath string) error {
	s.logger.Info("Creating metadata database backup", "backup_path", backupPath)

	query := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
	_, err := s.metadataDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create metadata backup: %w", err)
	}

	s.logger.Info("Metadata database backup created successfully", "backup_path", backupPath)
	return nil
}

func (s *DatabaseService) CreateGroupBackup(ctx context.Context, groupID, backupPath string) error {
	s.logger.Info("Creating group database backup", "group_id", groupID, "backup_path", backupPath)

	s.groupDBMutex.RLock()
	groupDB, exists := s.groupDBs[groupID]
	s.groupDBMutex.RUnlock()

	if !exists {
		return fmt.Errorf("group database not found: %s", groupID)
	}

	query := fmt.Sprintf("VACUUM INTO '%s'", backupPath)
	_, err := groupDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create group backup: %w", err)
	}

	s.logger.Info("Group database backup created successfully", "group_id", groupID, "backup_path", backupPath)
	return nil
}

// gRPC Backup handlers
func (s *DatabaseService) CreateBackup(ctx context.Context, req *connect.Request[snitchv1.CreateBackupRequest]) (*connect.Response[snitchv1.CreateBackupResponse], error) {
	backupPath := req.Msg.GetBackupPath()
	groupID := req.Msg.GetGroupId()

	s.logger.Info("Creating database backup", "backup_path", backupPath, "group_id", groupID)

	var err error
	if groupID == "" {
		// Backup metadata database
		err = s.CreateMetadataBackup(ctx, backupPath)
	} else {
		// Backup specific group database
		err = s.CreateGroupBackup(ctx, groupID, backupPath)
	}

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("backup failed: %w", err))
	}

	// Get backup file size
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat backup file: %w", err))
	}

	response := &snitchv1.CreateBackupResponse{
		BackupPath: backupPath,
		SizeBytes:  info.Size(),
	}

	return connect.NewResponse(response), nil
}

func (s *DatabaseService) ListGroupIDs(ctx context.Context, req *connect.Request[snitchv1.ListGroupIDsRequest]) (*connect.Response[snitchv1.ListGroupIDsResponse], error) {
	s.groupDBMutex.RLock()
	groupIDs := make([]string, 0, len(s.groupDBs))
	for groupID := range s.groupDBs {
		groupIDs = append(groupIDs, groupID)
	}
	s.groupDBMutex.RUnlock()

	response := &snitchv1.ListGroupIDsResponse{
		GroupIds: groupIDs,
	}

	return connect.NewResponse(response), nil
}
