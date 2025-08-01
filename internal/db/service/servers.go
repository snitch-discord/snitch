package service

import (
	"context"
	"fmt"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
)

// ListServers retrieves all servers for a given group from the metadata database
func (s *DatabaseService) ListServers(
	ctx context.Context,
	req *connect.Request[snitchv1.ListServersRequest],
) (*connect.Response[snitchv1.ListServersResponse], error) {
	query := `SELECT server_id, group_id FROM servers WHERE group_id = ?`
	
	rows, err := s.metadataDB.QueryContext(ctx, query, req.Msg.GroupId)
	if err != nil {
		s.logger.Error("Failed to list servers", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list servers: %w", err))
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn("Failed to close rows", "error", err)
		}
	}()

	var servers []*snitchv1.ServerEntry
	for rows.Next() {
		var server snitchv1.ServerEntry
		
		err := rows.Scan(&server.ServerId, &server.GroupId)
		if err != nil {
			s.logger.Error("Failed to scan server entry", "group_id", req.Msg.GroupId, "error", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan server entry: %w", err))
		}

		servers = append(servers, &server)
	}

	if err = rows.Err(); err != nil {
		s.logger.Error("Row iteration error", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("row iteration error: %w", err))
	}

	response := &snitchv1.ListServersResponse{
		Servers: servers,
	}

	return connect.NewResponse(response), nil
}