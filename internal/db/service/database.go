package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"snitch/internal/db/sqlcgen/groupdb"
	"snitch/internal/db/sqlcgen/metadata"

	_ "github.com/tursodatabase/go-libsql"
)

// Schema file paths for loading at runtime
const (
	metadataSchemaPath = "/app/internal/db/schemas/metadata.sql"
	groupSchemaPath    = "/app/internal/db/schemas/group_tables.sql"
)

// loadSchemaFile loads SQL schema from file
func loadSchemaFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open schema file %s: %w", filePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file %s: %w", filePath, err)
	}

	return string(content), nil
}


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

	// Create metadata tables
	if err := createMetadataTables(ctx, metadataDB); err != nil {
		if closeErr := metadataDB.Close(); closeErr != nil {
			logger.Warn("Failed to close metadata database during error cleanup", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create metadata tables: %w", err)
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

func createMetadataTables(ctx context.Context, db *sql.DB) error {
	schema, err := loadSchemaFile(metadataSchemaPath)
	if err != nil {
		return fmt.Errorf("failed to load metadata schema: %w", err)
	}

	queries := strings.Split(strings.TrimSpace(schema), ";")
	
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

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

	// Create group tables
	if err := s.createGroupTables(ctx, db); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			s.logger.Warn("Failed to close group database during error cleanup", "group_id", groupID, "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create group tables for %s: %w", groupID, err)
	}

	s.groupDBs[groupID] = db
	s.groupQueries[groupID] = groupdb.New(db)
	s.logger.Info("Created new group database", "group_id", groupID)

	return db, nil
}

func (s *DatabaseService) createGroupTables(ctx context.Context, db *sql.DB) error {
	// Load complete group schema (tables + indexes)
	schema, err := loadSchemaFile(groupSchemaPath)
	if err != nil {
		return fmt.Errorf("failed to load group schema: %w", err)
	}

	// Execute all schema statements
	queries := strings.Split(strings.TrimSpace(schema), ";")
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	return nil
}