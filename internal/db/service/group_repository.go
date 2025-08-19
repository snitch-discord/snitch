package service

import (
	"context"
	"fmt"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GroupRepository handles group database management operations
type GroupRepository struct {
	service *DatabaseService
}

// NewGroupRepository creates a new GroupRepository
func NewGroupRepository(service *DatabaseService) *GroupRepository {
	return &GroupRepository{
		service: service,
	}
}

// CreateGroupDatabase ensures the group database exists and tables are created
func (r *GroupRepository) CreateGroupDatabase(
	ctx context.Context,
	req *connect.Request[snitchv1.CreateGroupDatabaseRequest],
) (*connect.Response[emptypb.Empty], error) {
	_, err := r.service.createGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		r.service.logger.Error("Failed to create group database", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create group database: %w", err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}
