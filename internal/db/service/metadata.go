package service

import (
	"context"
	"database/sql"
	"fmt"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CreateGroup creates a new group in the metadata database
func (s *DatabaseService) CreateGroup(
	ctx context.Context,
	req *connect.Request[snitchv1.CreateGroupRequest],
) (*connect.Response[emptypb.Empty], error) {
	query := `INSERT INTO groups (group_id, group_name) VALUES (?, ?)`
	
	_, err := s.metadataDB.ExecContext(ctx, query, req.Msg.GroupId, req.Msg.GroupName)
	if err != nil {
		s.logger.Error("Failed to create group", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create group: %w", err))
	}

	s.logger.Info("Created group", "group_id", req.Msg.GroupId, "group_name", req.Msg.GroupName)
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// FindGroupByServer finds the group ID for a given server ID
func (s *DatabaseService) FindGroupByServer(
	ctx context.Context,
	req *connect.Request[snitchv1.FindGroupByServerRequest],
) (*connect.Response[snitchv1.FindGroupByServerResponse], error) {
	query := `SELECT group_id FROM servers WHERE server_id = ?`
	
	var groupID string
	err := s.metadataDB.QueryRowContext(ctx, query, req.Msg.ServerId).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("server not found: %s", req.Msg.ServerId))
		}
		s.logger.Error("Failed to find group by server", "server_id", req.Msg.ServerId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find group: %w", err))
	}

	response := &snitchv1.FindGroupByServerResponse{
		GroupId: groupID,
	}

	return connect.NewResponse(response), nil
}

// AddServerToGroup adds a server to a group
func (s *DatabaseService) AddServerToGroup(
	ctx context.Context,
	req *connect.Request[snitchv1.AddServerToGroupRequest],
) (*connect.Response[emptypb.Empty], error) {
	query := `INSERT INTO servers (server_id, output_channel, group_id, permission_level) VALUES (?, ?, ?, ?)`
	
	// Use default values for output_channel and permission_level as in the original queries
	_, err := s.metadataDB.ExecContext(ctx, query, req.Msg.ServerId, 69420, req.Msg.GroupId, 777)
	if err != nil {
		s.logger.Error("Failed to add server to group", 
			"server_id", req.Msg.ServerId, 
			"group_id", req.Msg.GroupId, 
			"error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add server to group: %w", err))
	}

	s.logger.Info("Added server to group", 
		"server_id", req.Msg.ServerId, 
		"group_id", req.Msg.GroupId)
	
	return connect.NewResponse(&emptypb.Empty{}), nil
}