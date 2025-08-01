package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CreateGroupDatabase ensures the group database exists and tables are created
func (s *DatabaseService) CreateGroupDatabase(
	ctx context.Context,
	req *connect.Request[snitchv1.CreateGroupDatabaseRequest],
) (*connect.Response[emptypb.Empty], error) {
	_, err := s.getOrCreateGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		s.logger.Error("Failed to create group database", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create group database: %w", err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// CreateReport creates a new report in the group database
func (s *DatabaseService) CreateReport(
	ctx context.Context,
	req *connect.Request[snitchv1.DbCreateReportRequest],
) (*connect.Response[snitchv1.DbCreateReportResponse], error) {
	db, err := s.getOrCreateGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	// Ensure users and servers exist
	if err := s.ensureUserExists(ctx, db, req.Msg.UserId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure user exists: %w", err))
	}
	if err := s.ensureUserExists(ctx, db, req.Msg.ReporterId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure reporter exists: %w", err))
	}
	if err := s.ensureServerExists(ctx, db, req.Msg.ServerId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure server exists: %w", err))
	}

	query := `INSERT INTO reports (report_text, reporter_id, reported_user_id, origin_server_id) 
			  VALUES (?, ?, ?, ?) RETURNING report_id`

	var reportID int64
	err = db.QueryRowContext(ctx, query, req.Msg.Reason, req.Msg.ReporterId, req.Msg.UserId, req.Msg.ServerId).Scan(&reportID)
	if err != nil {
		s.logger.Error("Failed to create report", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create report: %w", err))
	}

	response := &snitchv1.DbCreateReportResponse{
		ReportId: reportID,
	}

	s.logger.Info("Created report", "group_id", req.Msg.GroupId, "report_id", reportID)
	return connect.NewResponse(response), nil
}

// GetReport retrieves a specific report from the group database
func (s *DatabaseService) GetReport(
	ctx context.Context,
	req *connect.Request[snitchv1.DbGetReportRequest],
) (*connect.Response[snitchv1.DbGetReportResponse], error) {
	db, err := s.getOrCreateGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	query := `SELECT report_id, report_text, reporter_id, reported_user_id, origin_server_id, created_at 
			  FROM reports WHERE report_id = ?`

	var report snitchv1.DbGetReportResponse
	var createdAt time.Time
	
	err = db.QueryRowContext(ctx, query, req.Msg.ReportId).Scan(
		&report.Id,
		&report.Reason,
		&report.ReporterId,
		&report.UserId,
		&report.ServerId,
		&createdAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("report not found: %d", req.Msg.ReportId))
		}
		s.logger.Error("Failed to get report", "group_id", req.Msg.GroupId, "report_id", req.Msg.ReportId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get report: %w", err))
	}

	report.CreatedAt = createdAt.Format(time.RFC3339)

	return connect.NewResponse(&report), nil
}

// ListReports lists reports from the group database
func (s *DatabaseService) ListReports(
	ctx context.Context,
	req *connect.Request[snitchv1.DbListReportsRequest],
) (*connect.Response[snitchv1.DbListReportsResponse], error) {
	db, err := s.getOrCreateGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	query := `SELECT report_id, report_text, reporter_id, reported_user_id, origin_server_id, created_at 
			  FROM reports`
	args := []interface{}{}

	if req.Msg.UserId != nil {
		query += ` WHERE reported_user_id = ?`
		args = append(args, *req.Msg.UserId)
	}

	query += ` ORDER BY created_at DESC`

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
		s.logger.Error("Failed to list reports", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list reports: %w", err))
	}
	defer rows.Close()

	var reports []*snitchv1.DbGetReportResponse
	for rows.Next() {
		var report snitchv1.DbGetReportResponse
		var createdAt time.Time
		
		err := rows.Scan(
			&report.Id,
			&report.Reason,
			&report.ReporterId,
			&report.UserId,
			&report.ServerId,
			&createdAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan report", "group_id", req.Msg.GroupId, "error", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan report: %w", err))
		}

		report.CreatedAt = createdAt.Format(time.RFC3339)
		reports = append(reports, &report)
	}

	if err = rows.Err(); err != nil {
		s.logger.Error("Row iteration error", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("row iteration error: %w", err))
	}

	response := &snitchv1.DbListReportsResponse{
		Reports: reports,
	}

	return connect.NewResponse(response), nil
}

// DeleteReport deletes a report from the group database
func (s *DatabaseService) DeleteReport(
	ctx context.Context,
	req *connect.Request[snitchv1.DbDeleteReportRequest],
) (*connect.Response[emptypb.Empty], error) {
	db, err := s.getOrCreateGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	query := `DELETE FROM reports WHERE report_id = ?`
	
	result, err := db.ExecContext(ctx, query, req.Msg.ReportId)
	if err != nil {
		s.logger.Error("Failed to delete report", "group_id", req.Msg.GroupId, "report_id", req.Msg.ReportId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete report: %w", err))
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get affected rows: %w", err))
	}

	if affected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("report not found: %d", req.Msg.ReportId))
	}

	s.logger.Info("Deleted report", "group_id", req.Msg.GroupId, "report_id", req.Msg.ReportId)
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// Helper functions
func (s *DatabaseService) ensureUserExists(ctx context.Context, db *sql.DB, userID string) error {
	query := `INSERT OR IGNORE INTO users (user_id) VALUES (?)`
	_, err := db.ExecContext(ctx, query, userID)
	return err
}

func (s *DatabaseService) ensureServerExists(ctx context.Context, db *sql.DB, serverID string) error {
	query := `INSERT OR IGNORE INTO servers (server_id) VALUES (?)`
	_, err := db.ExecContext(ctx, query, serverID)
	return err
}