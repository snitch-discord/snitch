package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
)

// CreateUserHistory creates a new user history entry in the group database
func (s *DatabaseService) CreateUserHistory(
	ctx context.Context,
	req *connect.Request[snitchv1.DbCreateUserHistoryRequest],
) (*connect.Response[snitchv1.DbCreateUserHistoryResponse], error) {
	db, err := s.getGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	// Ensure user and server exist
	if err := s.ensureUserExists(ctx, db, req.Msg.UserId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure user exists: %w", err))
	}
	if err := s.ensureServerExists(ctx, db, req.Msg.ServerId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure server exists: %w", err))
	}

	query := `INSERT INTO user_history (user_id, server_id, action, reason, evidence_url) 
			  VALUES (?, ?, ?, ?, ?) RETURNING history_id`

	var historyID int64
	err = db.QueryRowContext(ctx, query, 
		req.Msg.UserId, 
		req.Msg.ServerId, 
		req.Msg.Action, 
		req.Msg.Reason, 
		req.Msg.EvidenceUrl).Scan(&historyID)
		
	if err != nil {
		s.logger.Error("Failed to create user history", "group_id", req.Msg.GroupId, "user_id", req.Msg.UserId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create user history: %w", err))
	}

	response := &snitchv1.DbCreateUserHistoryResponse{
		HistoryId: historyID,
	}

	s.logger.Info("Created user history", 
		"group_id", req.Msg.GroupId, 
		"user_id", req.Msg.UserId, 
		"history_id", historyID,
		"action", req.Msg.Action)
		
	return connect.NewResponse(response), nil
}

// GetUserHistory retrieves user history entries from the group database
func (s *DatabaseService) GetUserHistory(
	ctx context.Context,
	req *connect.Request[snitchv1.DbGetUserHistoryRequest],
) (*connect.Response[snitchv1.DbGetUserHistoryResponse], error) {
	db, err := s.getGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	query := `SELECT history_id, user_id, server_id, action, reason, evidence_url, created_at 
			  FROM user_history WHERE user_id = ? ORDER BY created_at DESC`
	args := []interface{}{req.Msg.UserId}

	if req.Msg.Limit != nil {
		query += ` LIMIT ?`
		args = append(args, *req.Msg.Limit)
	}

	if req.Msg.Offset != nil {
		query += ` OFFSET ?`
		args = append(args, *req.Msg.Offset)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		s.logger.Error("Failed to get user history", "group_id", req.Msg.GroupId, "user_id", req.Msg.UserId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user history: %w", err))
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.Warn("Failed to close rows", "error", err)
		}
	}()

	var entries []*snitchv1.DbUserHistoryEntry
	for rows.Next() {
		var entry snitchv1.DbUserHistoryEntry
		var createdAt time.Time
		var reason, evidenceURL sql.NullString
		
		err := rows.Scan(
			&entry.Id,
			&entry.UserId,
			&entry.ServerId,
			&entry.Action,
			&reason,
			&evidenceURL,
			&createdAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan user history entry", "group_id", req.Msg.GroupId, "user_id", req.Msg.UserId, "error", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan user history entry: %w", err))
		}

		entry.CreatedAt = createdAt.Format(time.RFC3339)
		if reason.Valid {
			entry.Reason = &reason.String
		}
		if evidenceURL.Valid {
			entry.EvidenceUrl = &evidenceURL.String
		}

		entries = append(entries, &entry)
	}

	if err = rows.Err(); err != nil {
		s.logger.Error("Row iteration error", "group_id", req.Msg.GroupId, "user_id", req.Msg.UserId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("row iteration error: %w", err))
	}

	response := &snitchv1.DbGetUserHistoryResponse{
		Entries: entries,
	}

	return connect.NewResponse(response), nil
}