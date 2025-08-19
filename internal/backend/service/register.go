package service

import (
	"context"
	"fmt"
	"log/slog"
	"snitch/internal/shared/ctxutil"
	snitchpb "snitch/pkg/proto/gen/snitch/v1"
	"snitch/pkg/proto/gen/snitch/v1/snitchv1connect"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type RegisterServer struct {
	dbClient snitchv1connect.DatabaseServiceClient
}

func NewRegisterServer(dbClient snitchv1connect.DatabaseServiceClient) *RegisterServer {
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
	findGroupReq := &snitchpb.FindGroupByServerRequest{
		ServerId: serverID,
	}
	_, err = s.dbClient.FindGroupByServer(ctx, connect.NewRequest(findGroupReq))
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
		addServerReq := &snitchpb.AddServerToGroupRequest{
			ServerId: serverID,
			GroupId:  groupID.String(),
		}
		_, err := s.dbClient.AddServerToGroup(ctx, connect.NewRequest(addServerReq))
		if err != nil {
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
		createGroupReq := &snitchpb.CreateGroupRequest{
			GroupId:   groupID.String(),
			GroupName: *req.Msg.GroupName,
		}
		_, err := s.dbClient.CreateGroup(ctx, connect.NewRequest(createGroupReq))
		if err != nil {
			slogger.ErrorContext(ctx, "Failed to create group", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		// Create the group database
		createGroupDbReq := &snitchpb.CreateGroupDatabaseRequest{
			GroupId: groupID.String(),
		}
		_, err = s.dbClient.CreateGroupDatabase(ctx, connect.NewRequest(createGroupDbReq))
		if err != nil {
			slogger.ErrorContext(ctx, "Failed to create group database", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		// Add server to the new group
		addServerToNewGroupReq := &snitchpb.AddServerToGroupRequest{
			ServerId: serverID,
			GroupId:  groupID.String(),
		}
		_, err = s.dbClient.AddServerToGroup(ctx, connect.NewRequest(addServerToNewGroupReq))
		if err != nil {
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
