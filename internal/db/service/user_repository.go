package service

import (
	"context"
	"database/sql"
	"fmt"

	"snitch/internal/db/sqlc/gen/groupdb"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
)

// UserRepository handles user history operations
type UserRepository struct {
	service *DatabaseService
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(service *DatabaseService) *UserRepository {
	return &UserRepository{
		service: service,
	}
}

// CreateUserHistory creates a new user history entry in the group database using sqlc
func (r *UserRepository) CreateUserHistory(
	ctx context.Context,
	req *connect.Request[snitchv1.DbCreateUserHistoryRequest],
) (*connect.Response[snitchv1.DbCreateUserHistoryResponse], error) {
	db, err := r.service.getGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	queries := groupdb.New(db)

	// Ensure user and server exist using sqlc
	if err := queries.EnsureUserExists(ctx, req.Msg.UserId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure user exists: %w", err))
	}
	if err := queries.EnsureServerExists(ctx, req.Msg.ServerId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure server exists: %w", err))
	}

	// Create user history using sqlc
	var reason, evidenceUrl sql.NullString
	if req.Msg.Reason != nil {
		reason = sql.NullString{String: *req.Msg.Reason, Valid: true}
	}
	if req.Msg.EvidenceUrl != nil {
		evidenceUrl = sql.NullString{String: *req.Msg.EvidenceUrl, Valid: true}
	}

	historyID, err := queries.CreateUserHistory(ctx, groupdb.CreateUserHistoryParams{
		UserID:      req.Msg.UserId,
		ServerID:    req.Msg.ServerId,
		Action:      req.Msg.Action,
		Reason:      reason,
		EvidenceUrl: evidenceUrl,
	})
	if err != nil {
		r.service.logger.Error("Failed to create user history", "group_id", req.Msg.GroupId, "user_id", req.Msg.UserId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user history: %w", err))
	}

	response := &snitchv1.DbCreateUserHistoryResponse{
		HistoryId: historyID,
	}

	r.service.logger.Info("Created user history",
		"group_id", req.Msg.GroupId,
		"user_id", req.Msg.UserId,
		"history_id", historyID,
		"action", req.Msg.Action)

	return connect.NewResponse(response), nil
}

// GetUserHistory retrieves user history entries from the group database using sqlc
func (r *UserRepository) GetUserHistory(
	ctx context.Context,
	req *connect.Request[snitchv1.DbGetUserHistoryRequest],
) (*connect.Response[snitchv1.DbGetUserHistoryResponse], error) {
	db, err := r.service.getGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	queries := groupdb.New(db)

	// Get user history using sqlc
	historyRows, err := queries.GetUserHistory(ctx, req.Msg.UserId)
	if err != nil {
		r.service.logger.Error("Failed to get user history", "group_id", req.Msg.GroupId, "user_id", req.Msg.UserId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user history: %w", err))
	}

	// Apply limit and offset manually since sqlc doesn't support dynamic LIMIT/OFFSET
	start := 0
	if req.Msg.Offset != nil {
		start = int(*req.Msg.Offset)
	}

	end := len(historyRows)
	if req.Msg.Limit != nil {
		end = start + int(*req.Msg.Limit)
		if end > len(historyRows) {
			end = len(historyRows)
		}
	}

	if start >= len(historyRows) {
		historyRows = []groupdb.UserHistory{}
	} else {
		historyRows = historyRows[start:end]
	}

	var entries []*snitchv1.DbUserHistoryEntry
	for _, historyRow := range historyRows {
		entry := &snitchv1.DbUserHistoryEntry{
			Id:       historyRow.HistoryID,
			UserId:   historyRow.UserID,
			ServerId: historyRow.ServerID,
			Action:   historyRow.Action,
		}

		// Handle nullable CreatedAt field
		if historyRow.CreatedAt.Valid {
			entry.CreatedAt = historyRow.CreatedAt.String
		}

		// Handle nullable fields
		if historyRow.Reason.Valid {
			entry.Reason = &historyRow.Reason.String
		}
		if historyRow.EvidenceUrl.Valid {
			entry.EvidenceUrl = &historyRow.EvidenceUrl.String
		}

		entries = append(entries, entry)
	}

	response := &snitchv1.DbGetUserHistoryResponse{
		Entries: entries,
	}

	return connect.NewResponse(response), nil
}
