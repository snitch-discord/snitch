package metadata

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"snitch/internal/backend/dbconfig"
	"snitch/internal/backend/libsqladmin"
	"snitch/internal/backend/metadata/gen/sqlc"
	"snitch/internal/shared/ctxutil"

	"github.com/google/uuid"
	"github.com/tursodatabase/go-libsql"
)

func NewMetadataDB(ctx context.Context, token string, config dbconfig.LibSQLConfig) (*sql.DB, error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	exists, err := libsqladmin.DoesNamespaceExist("metadata", ctx, token, config)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed checking if namespace exists", "Error", err)
		return nil, fmt.Errorf("couldnt check if namespace exists: %w", err)
	}

	if !exists {
		if err := libsqladmin.CreateNamespace("metadata", ctx, token, config); err != nil {
			slogger.ErrorContext(ctx, "Failed creating metadata namespace", "Error", err)
			return nil, fmt.Errorf("couldnt create namespace: %w", err)
		}
	}

	metadataURL, err := config.MetadataURL()
	if err != nil {
		return nil, fmt.Errorf("cant get metadata: %w", err)
	}

	dbPath := filepath.Join("/localdb", "local.metadata.db")

	conn, err := libsql.NewEmbeddedReplicaConnector(dbPath, metadataURL.String(), libsql.WithAuthToken(token))
	if err != nil {
		slogger.ErrorContext(ctx, "Error opening DB", "Error", err)
		return nil, fmt.Errorf("couldnt open db: %w", err)
	}

	db := sql.OpenDB(conn)

	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		slogger.ErrorContext(ctx, "Failed starting transaction", "Error", err)
		return nil, fmt.Errorf("couldnt start transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); !errors.Is(err, sql.ErrTxDone) {
			slogger.ErrorContext(ctx, "Failed to rollback transaction", "Error", err)
		}
	}()

	queries := sqlc.New(tx)

	qtx := queries.WithTx(tx)

	if err := qtx.CreateGroupTable(ctx); err != nil {
		slogger.ErrorContext(ctx, "Failed creating group table", "Error", err)
		return nil, fmt.Errorf("couldnt create group table: %w", err)
	}

	if err := qtx.CreateServerTable(ctx); err != nil {
		slogger.ErrorContext(ctx, "Failed creating server table", "Error", err)
		return nil, fmt.Errorf("couldnt create server table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		slogger.ErrorContext(ctx, "Failed to commit transaction", "Error", err)
		return nil, fmt.Errorf("couldnt commit transaction: %w", err)
	}
	return db, nil
}

func FindGroupIDByServerID(ctx context.Context, db *sql.DB, serverID string) (uuid.UUID, error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	var groupID uuid.UUID
	queries := sqlc.New(db)
	groupID, err := queries.FindGroupIDByServerID(ctx, serverID)

	if err != nil {
		slogger.ErrorContext(ctx, "Failed finding group id", "Error", err)
		return uuid.Nil, fmt.Errorf("couldnt find group id: %w", err)
	}

	return groupID, nil
}

func AddServerToGroup(ctx context.Context, db *sql.DB, serverID string, groupID uuid.UUID) error {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	queries := sqlc.New(db)
	if err := queries.AddServerToGroup(ctx, sqlc.AddServerToGroupParams{
		GroupID:  groupID,
		ServerID: serverID,
	}); err != nil {
		slogger.ErrorContext(ctx, "Failed adding server to group", "Error", err)
		return fmt.Errorf("couldnt add server to group: %w", err)
	}

	return nil
}
