package service

import (
	"context"
	"fmt"
	"log/slog"
	"snitch/internal/backend/dbconfig"
	"snitch/internal/backend/group"
	groupSQLc "snitch/internal/backend/group/gen/sqlc"
	"snitch/internal/backend/jwt"
	"snitch/internal/backend/service/interceptor"
	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type UserServer struct {
	tokenCache   *jwt.TokenCache
	libSQLConfig dbconfig.LibSQLConfig
}

func NewUserServer(tokenCache *jwt.TokenCache, libSQLConfig dbconfig.LibSQLConfig) *UserServer {
	return &UserServer{tokenCache: tokenCache, libSQLConfig: libSQLConfig}
}

func (s *UserServer) CreateUserHistory(
	ctx context.Context,
	req *connect.Request[snitchv1.CreateUserHistoryRequest],
) (*connect.Response[snitchv1.CreateUserHistoryResponse], error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	groupID, err := interceptor.GetGroupID(ctx)
	if err != nil {
		slogger.Error("Couldn't get group id", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	db, err := group.GetGroupDB(ctx, s.tokenCache.Get(), s.libSQLConfig, groupID)
	if err != nil {
		slogger.Error("Failed creating group db", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	queries := groupSQLc.New(db)

	userHistoryID, err := queries.CreateUserHistory(ctx, groupSQLc.CreateUserHistoryParams{
		HistoryID:  uuid.New().String(),
		UserID:     int(req.Msg.UserId),
		Username:   req.Msg.Username,
		GlobalName: req.Msg.GlobalName,
		ChangedAt:  req.Msg.ChangedAt,
	})

	if err != nil {
		slogger.Error(fmt.Sprintf("failed to add user history %d", req.Msg.UserId), "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&snitchv1.CreateUserHistoryResponse{
		UserId: int32(userHistoryID.UserID),
	}), nil
}

func (s *UserServer) ListUserHistory(
	ctx context.Context,
	req *connect.Request[snitchv1.ListUserHistoryRequest],
) (*connect.Response[snitchv1.ListUserHistoryResponse], error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	groupID, err := interceptor.GetGroupID(ctx)
	if err != nil {
		slogger.Error("Couldn't get group id", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	slogger.Info("Group ID", "ID", groupID)

	db, err := group.GetGroupDB(ctx, s.tokenCache.Get(), s.libSQLConfig, groupID)
	if err != nil {
		slogger.Error("Failed getting group db", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	queries := groupSQLc.New(db)

	dbUserHistory, err := queries.GetUserHistory(ctx, int(req.Msg.UserId))

	if err != nil {
		slogger.Error(fmt.Sprintf("failed to get user history %d", req.Msg.UserId), "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	rpcHistory := make([]*snitchv1.CreateUserHistoryRequest, 0, len(dbUserHistory))

	for _, userHistory := range dbUserHistory {
		rpcHistory = append(rpcHistory, &snitchv1.CreateUserHistoryRequest{
			Username:   userHistory.Username,
			ChangedAt:  userHistory.ChangedAt,
			UserId:     int32(userHistory.UserID),
			GlobalName: userHistory.GlobalName,
		})
	}

	return connect.NewResponse(&snitchv1.ListUserHistoryResponse{
		UserHistory: rpcHistory,
	}), nil
}
