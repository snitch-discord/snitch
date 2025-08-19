package service

import (
	"context"
	"fmt"
	"log/slog"

	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
)

type UserServer struct {
	dbClient snitchv1connect.DatabaseServiceClient
}

func NewUserServer(dbClient snitchv1connect.DatabaseServiceClient) *UserServer {
	return &UserServer{
		dbClient: dbClient,
	}
}

func (s *UserServer) CreateUserHistory(
	ctx context.Context,
	req *connect.Request[snitchv1.CreateUserHistoryRequest],
) (*connect.Response[snitchv1.CreateUserHistoryResponse], error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	// Get server ID from header
	serverID := req.Header().Get(ServerIDHeader)
	if serverID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("server ID header is required"))
	}

	// Find group ID for this server
	findGroupReq := &snitchv1.FindGroupByServerRequest{
		ServerId: serverID,
	}
	findGroupResp, err := s.dbClient.FindGroupByServer(ctx, connect.NewRequest(findGroupReq))
	if err != nil {
		slogger.Error("Failed to find group for server", "server_id", serverID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	groupID := findGroupResp.Msg.GroupId

	// Create user history entry
	createHistoryReq := &snitchv1.DatabaseServiceCreateUserHistoryRequest{
		GroupId:     groupID,
		UserId:      req.Msg.UserId,
		ServerId:    serverID,
		Action:      "username_change",
		Reason:      &req.Msg.Username,
		EvidenceUrl: nil,
	}
	createHistoryResp, err := s.dbClient.CreateUserHistory(ctx, connect.NewRequest(createHistoryReq))
	if err != nil {
		slogger.Error("Failed to create user history", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	historyID := createHistoryResp.Msg.HistoryId
	slogger.Info("User history created", "history_id", historyID, "group_id", groupID, "user_id", req.Msg.UserId)

	return connect.NewResponse(&snitchv1.CreateUserHistoryResponse{
		UserId: req.Msg.UserId,
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

	// Get server ID from header
	serverID := req.Header().Get(ServerIDHeader)
	if serverID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("server ID header is required"))
	}

	// Find group ID for this server
	findGroupReq := &snitchv1.FindGroupByServerRequest{
		ServerId: serverID,
	}
	findGroupResp, err := s.dbClient.FindGroupByServer(ctx, connect.NewRequest(findGroupReq))
	if err != nil {
		slogger.Error("Failed to find group for server", "server_id", serverID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	groupID := findGroupResp.Msg.GroupId

	// Get user history
	getUserHistoryReq := &snitchv1.DatabaseServiceGetUserHistoryRequest{
		GroupId: groupID,
		UserId:  req.Msg.UserId,
		Limit:   nil,
		Offset:  nil,
	}
	getUserHistoryResp, err := s.dbClient.GetUserHistory(ctx, connect.NewRequest(getUserHistoryReq))
	if err != nil {
		slogger.Error("Failed to get user history", "group_id", groupID, "user_id", req.Msg.UserId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert to API format
	var userHistory []*snitchv1.CreateUserHistoryRequest
	for _, entry := range getUserHistoryResp.Msg.Entries {
		// Extract username from reason field as a workaround
		username := ""
		if entry.Reason != nil {
			username = *entry.Reason
		}

		userHistory = append(userHistory, &snitchv1.CreateUserHistoryRequest{
			UserId:     entry.UserId,
			Username:   username,
			GlobalName: "", // Not available in new format
			ChangedAt:  entry.CreatedAt,
		})
	}

	return connect.NewResponse(&snitchv1.ListUserHistoryResponse{
		UserHistory: userHistory,
	}), nil
}
