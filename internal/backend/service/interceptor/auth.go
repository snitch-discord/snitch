package interceptor

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"
)

type contextKey string

const (
	serverIDContextKey = contextKey("server_id")
	groupIDContextKey  = contextKey("group_id")
)

func NewAuthInterceptor(jwtSecret string, dbClient snitchv1connect.DatabaseServiceClient) connect.UnaryInterceptorFunc {
	interceptor := func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing authorization header"))
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid authorization header format"))
			}
			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token: %w", err))
			}

			if !token.Valid {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token"))
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid token claims"))
			}

			serverID, ok := claims["sub"].(string)
			if !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("missing 'sub' claim"))
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
