package group

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"snitch/internal/backend/dbconfig"

	"snitch/internal/shared/ctxutil"

	_ "github.com/tursodatabase/go-libsql"
)

func NewGroupDB(ctx context.Context, token string, config dbconfig.LibSQLConfig, groupID string) (*sql.DB, error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	databaseURL, err := config.NamespaceURL(groupID, token)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed getting group DB URL", "Error", err)
		return nil, fmt.Errorf("couldnt get group DB URL: %w", err)
	}

	db, err := sql.Open("libsql", databaseURL.String())
	if err != nil {
		slogger.ErrorContext(ctx, "Failed creating group DB", "Error", err)
		return nil, fmt.Errorf("couldnt create group DB: %w", err)
	}

	return db, nil
}
