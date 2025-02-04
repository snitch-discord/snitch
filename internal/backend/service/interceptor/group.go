package interceptor

import (
	"context"
	"database/sql"
	"fmt"
	"snitch/internal/backend/metadata"
	"strconv"

	"connectrpc.com/connect"
)

type contextKey string

const (
	ServerIDHeader     = "X-Server-ID"
	serverIDContextKey = contextKey("server_id")
	groupIDContextKey  = contextKey("group_id")
)

func getServerID(req connect.AnyRequest) (int, error) {
	serverIDStr := req.Header().Get(ServerIDHeader)
	if serverIDStr == "" {
		return 0, fmt.Errorf("server ID header is required")
	}

	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid server ID format")
	}
	return serverID, nil
}

func NewGroupContextInterceptor(metadataDB *sql.DB) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			serverID, err := getServerID(req)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}

			groupID, err := metadata.FindGroupIDByServerID(ctx, metadataDB, serverID)
			if err != nil {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}

			ctx = context.WithValue(ctx, serverIDContextKey, serverID)
			ctx = context.WithValue(ctx, groupIDContextKey, groupID.String())

			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
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
