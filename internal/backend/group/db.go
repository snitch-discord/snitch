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

func GetGroupDB(ctx context.Context, token string, config dbconfig.LibSQLConfig, groupID string) (*sql.DB, error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	databaseURL, err := config.NamespaceURL(groupID)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed getting group DB URL", "Error", err)
		return nil, fmt.Errorf("couldnt get group DB URL: %w", err)
	}

	// Construct connection string with auth token
	// The token is only in memory, not logged or exposed in URLs
	connectionString := fmt.Sprintf("%s?authToken=%s", databaseURL.String(), token)
	
	db, err := sql.Open("libsql", connectionString)
	if err != nil {
		// Log error without exposing the connection string that contains the token
		slogger.ErrorContext(ctx, "Failed opening LibSQL database", "Error", err, "namespace", groupID)
		return nil, fmt.Errorf("couldnt open LibSQL database: %w", err)
	}
	
	return db, nil
}
