package service

import (
	"context"
	"database/sql"
	"fmt"

	"snitch/internal/db/sqlc/gen/metadata"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ServerRepository handles server and metadata operations
type ServerRepository struct {
	service *DatabaseService
}

// NewServerRepository creates a new ServerRepository
func NewServerRepository(service *DatabaseService) *ServerRepository {
	return &ServerRepository{
		service: service,
	}
}

// CreateGroup creates a new group in the metadata database using sqlc
func (r *ServerRepository) CreateGroup(
	ctx context.Context,
	req *connect.Request[snitchv1.CreateGroupRequest],
) (*connect.Response[emptypb.Empty], error) {
	queries := metadata.New(r.service.metadataDB)

	err := queries.CreateGroup(ctx, metadata.CreateGroupParams{
		GroupID:   req.Msg.GroupId,
		GroupName: req.Msg.GroupName,
	})
	if err != nil {
		r.service.logger.Error("Failed to create group", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create group: %w", err))
	}

	r.service.logger.Info("Created group", "group_id", req.Msg.GroupId, "group_name", req.Msg.GroupName)
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// FindGroupByServer finds the group ID for a given server ID using sqlc
func (r *ServerRepository) FindGroupByServer(
	ctx context.Context,
	req *connect.Request[snitchv1.FindGroupByServerRequest],
) (*connect.Response[snitchv1.FindGroupByServerResponse], error) {
	queries := metadata.New(r.service.metadataDB)

	groupID, err := queries.FindGroupByServer(ctx, req.Msg.ServerId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("server not found: %s", req.Msg.ServerId))
		}
		r.service.logger.Error("Failed to find group by server", "server_id", req.Msg.ServerId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find group: %w", err))
	}

	response := &snitchv1.FindGroupByServerResponse{
		GroupId: groupID,
	}

	return connect.NewResponse(response), nil
}

// AddServerToGroup adds a server to a group using sqlc
func (r *ServerRepository) AddServerToGroup(
	ctx context.Context,
	req *connect.Request[snitchv1.AddServerToGroupRequest],
) (*connect.Response[emptypb.Empty], error) {
	queries := metadata.New(r.service.metadataDB)

	// Use default values for output_channel and permission_level as in the original queries
	err := queries.AddServerToGroup(ctx, metadata.AddServerToGroupParams{
		ServerID:        req.Msg.ServerId,
		OutputChannel:   69420,
		GroupID:         req.Msg.GroupId,
		PermissionLevel: 777,
	})
	if err != nil {
		r.service.logger.Error("Failed to add server to group",
			"server_id", req.Msg.ServerId,
			"group_id", req.Msg.GroupId,
			"error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add server to group: %w", err))
	}

	r.service.logger.Info("Added server to group",
		"server_id", req.Msg.ServerId,
		"group_id", req.Msg.GroupId)

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// ListServers retrieves all servers for a given group from the metadata database using sqlc
func (r *ServerRepository) ListServers(
	ctx context.Context,
	req *connect.Request[snitchv1.ListServersRequest],
) (*connect.Response[snitchv1.ListServersResponse], error) {
	queries := metadata.New(r.service.metadataDB)

	serverRows, err := queries.ListServers(ctx, req.Msg.GroupId)
	if err != nil {
		r.service.logger.Error("Failed to list servers", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list servers: %w", err))
	}

	var servers []*snitchv1.ServerEntry
	for _, serverRow := range serverRows {
		server := &snitchv1.ServerEntry{
			ServerId: serverRow.ServerID,
			GroupId:  serverRow.GroupID,
		}
		servers = append(servers, server)
	}

	response := &snitchv1.ListServersResponse{
		Servers: servers,
	}

	return connect.NewResponse(response), nil
}
