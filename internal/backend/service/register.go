package service

import (
	"context"
	"fmt"
	"log/slog"
	"snitch/internal/backend/dbclient"
	"snitch/internal/shared/ctxutil"
	snitchpb "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type RegisterServer struct {
	dbClient *dbclient.Client
}

func NewRegisterServer(dbClient *dbclient.Client) *RegisterServer {
	return &RegisterServer{dbClient: dbClient}
}

const ServerIDHeader = "X-Server-ID"

func getServerIDFromHeader(r *connect.Request[snitchpb.RegisterRequest]) (string, error) {
	serverID := r.Header().Get(ServerIDHeader)
	if serverID == "" {
		return "", fmt.Errorf("server ID header is required")
	}

	return serverID, nil
}

func (s *RegisterServer) Register(
	ctx context.Context,
	req *connect.Request[snitchpb.RegisterRequest],
) (*connect.Response[snitchpb.RegisterResponse], error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}
	
	serverID, err := getServerIDFromHeader(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Check if server is already registered
	_, err = s.dbClient.FindGroupByServer(ctx, serverID)
	if err == nil {
		slogger.ErrorContext(ctx, "Server is already registered")
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("server already registered"))
	}

	var groupID uuid.UUID

	if req.Msg.GroupId != nil {
		// Join existing group flow
		groupID, err = uuid.Parse(*req.Msg.GroupId)
		if err != nil {
			slogger.ErrorContext(ctx, "Invalid group ID format")
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		// Add server to existing group
		if err := s.dbClient.AddServerToGroup(ctx, serverID, groupID.String()); err != nil {
			slogger.ErrorContext(ctx, "Failed adding server to group", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else {
		// Create new group flow
		if req.Msg.GroupName == nil || *req.Msg.GroupName == "" {
			slogger.ErrorContext(ctx, "Group name is required when creating a new group")
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group name required"))
		}

		groupID = uuid.New()
		
		// Create the group
		if err := s.dbClient.CreateGroup(ctx, groupID.String(), *req.Msg.GroupName); err != nil {
			slogger.ErrorContext(ctx, "Failed to create group", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		// Create the group database
		if err := s.dbClient.CreateGroupDatabase(ctx, groupID.String()); err != nil {
			slogger.ErrorContext(ctx, "Failed to create group database", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		// Add server to the new group
		if err := s.dbClient.AddServerToGroup(ctx, serverID, groupID.String()); err != nil {
			slogger.ErrorContext(ctx, "Failed adding server to group", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	slogger.InfoContext(ctx, "Registration completed",
		"groupID", groupID.String(),
		"serverID", serverID,
		"isNewGroup", req.Msg.GroupId == nil)

	return connect.NewResponse(&snitchpb.RegisterResponse{
		ServerId: serverID,
		GroupId:  groupID.String(),
	}), nil
}
