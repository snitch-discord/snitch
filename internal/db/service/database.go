package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/tursodatabase/go-libsql"
)

type DatabaseService struct {
	metadataDB   *sql.DB
	groupDBs     map[string]*sql.DB
	groupDBMutex sync.RWMutex
	dbDir        string
	logger       *slog.Logger
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

	// Create metadata tables
	if err := createMetadataTables(ctx, metadataDB); err != nil {
		metadataDB.Close()
		return nil, fmt.Errorf("failed to create metadata tables: %w", err)
	}

	service := &DatabaseService{
		metadataDB: metadataDB,
		groupDBs:   make(map[string]*sql.DB),
		dbDir:      dbDir,
		logger:     logger,
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
	queries := []string{
		`CREATE TABLE IF NOT EXISTS groups (
			group_id TEXT PRIMARY KEY,
			group_name TEXT NOT NULL
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS servers (
			server_id TEXT NOT NULL,
			output_channel INTEGER NOT NULL,
			group_id TEXT NOT NULL REFERENCES groups(group_id),
			permission_level INTEGER NOT NULL,
			PRIMARY KEY (server_id, group_id)
		) STRICT`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	return nil
}

func (s *DatabaseService) getOrCreateGroupDB(ctx context.Context, groupID string) (*sql.DB, error) {
	s.groupDBMutex.RLock()
	if db, exists := s.groupDBs[groupID]; exists {
		s.groupDBMutex.RUnlock()
		return db, nil
	}
	s.groupDBMutex.RUnlock()

	s.groupDBMutex.Lock()
	defer s.groupDBMutex.Unlock()

	// Double-check after acquiring write lock
	if db, exists := s.groupDBs[groupID]; exists {
		return db, nil
	}

	// Create new group database
	groupPath := filepath.Join(s.dbDir, fmt.Sprintf("group_%s.db", groupID))
	db, err := sql.Open("libsql", "file:"+groupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open group database for %s: %w", groupID, err)
	}

	// Create group tables
	if err := s.createGroupTables(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create group tables for %s: %w", groupID, err)
	}

	s.groupDBs[groupID] = db
	s.logger.Info("Created new group database", "group_id", groupID)

	return db, nil
}

func (s *DatabaseService) createGroupTables(ctx context.Context, db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS servers (
			server_id TEXT PRIMARY KEY
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS reports (
			report_id INTEGER PRIMARY KEY,
			report_text TEXT NOT NULL CHECK(length(report_text) <= 2000 AND length(report_text) > 0),
			reporter_id TEXT NOT NULL REFERENCES users(user_id),
			reported_user_id TEXT NOT NULL REFERENCES users(user_id),
			origin_server_id TEXT NOT NULL REFERENCES servers(server_id),
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		) STRICT`,
		`CREATE TABLE IF NOT EXISTS user_history (
			history_id INTEGER PRIMARY KEY,
			user_id TEXT NOT NULL,
			server_id TEXT NOT NULL,
			action TEXT NOT NULL CHECK(length(action) <= 100),
			reason TEXT CHECK(reason IS NULL OR length(reason) <= 1000),
			evidence_url TEXT CHECK(evidence_url IS NULL OR length(evidence_url) <= 500),
			created_at TEXT DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(user_id),
			FOREIGN KEY (server_id) REFERENCES servers(server_id)
		) STRICT`,
		`CREATE INDEX IF NOT EXISTS idx_reports_reporter_id ON reports(reporter_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_reported_user_id ON reports(reported_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_created_at ON reports(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_origin_server_id ON reports(origin_server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_history_user_id ON user_history(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_history_server_id ON user_history(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_history_created_at ON user_history(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_user_date ON reports(reported_user_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_reports_server_date ON reports(origin_server_id, created_at)`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	return nil
}