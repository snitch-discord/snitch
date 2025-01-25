package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"snitch/internal/backend/metadata"
	"strconv"
)

type contextKey string

const (
	ServerIDHeader     = "X-Server-ID"
	serverIDContextKey = contextKey("server_id")
	groupIDContextKey  = contextKey("group_id")
)

func getServerID(r *http.Request) (int, error) {
	serverIDStr := r.Header.Get(ServerIDHeader)
	if serverIDStr == "" {
		return 0, fmt.Errorf("server ID header is required")
	}

	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid server ID format")
	}
	return serverID, nil
}

func GroupContext(next http.HandlerFunc, metadataDB *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serverID, err := getServerID(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		groupID, err := metadata.FindGroupIDByServerID(r.Context(), metadataDB, serverID)
		if err != nil {
			http.Error(w, "Server not found", http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), serverIDContextKey, serverID)
		ctx = context.WithValue(ctx, groupIDContextKey, groupID.String())

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetServerID(ctx context.Context) (int, error) {
	serverID, ok := ctx.Value(serverIDContextKey).(int)
	if !ok {
		return 0, fmt.Errorf("server ID not found in context")
	}
	return serverID, nil
}

func GetGroupID(ctx context.Context) (string, error) {
	groupID, ok := ctx.Value(groupIDContextKey).(string)
	if !ok {
		return "", fmt.Errorf("group ID not found in context")
	}
	return groupID, nil
}
