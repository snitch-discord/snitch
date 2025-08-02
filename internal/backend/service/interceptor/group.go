package interceptor

import (
	"context"
	"fmt"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
)

type contextKey string

const (
	ServerIDHeader     = "X-Server-ID"
	serverIDContextKey = contextKey("server_id")
	groupIDContextKey  = contextKey("group_id")
)

func getServerID(req connect.AnyRequest) (string, error) {
	serverID := req.Header().Get(ServerIDHeader)
	if serverID == "" {
		return "", fmt.Errorf("server ID header is required")
	}

	return serverID, nil
}

func NewGroupContextInterceptor(dbClient snitchv1connect.DatabaseServiceClient) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			serverID, err := getServerID(req)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}

			// Find group ID using database service
			findGroupReq := &snitchv1.FindGroupByServerRequest{
				ServerId: serverID,
			}
			findGroupResp, err := dbClient.FindGroupByServer(ctx, connect.NewRequest(findGroupReq))
			if err != nil {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}

			ctx = context.WithValue(ctx, serverIDContextKey, serverID)
			ctx = context.WithValue(ctx, groupIDContextKey, findGroupResp.Msg.GroupId)

			return next(ctx, req)
		})
	}
	return connect.UnaryInterceptorFunc(interceptor)
}

func GetServerID(ctx context.Context) (string, error) {
	serverID, ok := ctx.Value(serverIDContextKey).(string)
	if !ok {
		return "", fmt.Errorf("server ID not found in context")
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
