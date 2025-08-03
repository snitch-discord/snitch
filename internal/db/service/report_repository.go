package service

import (
	"context"
	"database/sql"
	"fmt"

	"snitch/internal/db/sqlcgen/groupdb"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ReportRepository handles report CRUD operations
type ReportRepository struct {
	service *DatabaseService
}

// NewReportRepository creates a new ReportRepository
func NewReportRepository(service *DatabaseService) *ReportRepository {
	return &ReportRepository{
		service: service,
	}
}

// CreateReport creates a new report in the group database using sqlc
func (r *ReportRepository) CreateReport(
	ctx context.Context,
	req *connect.Request[snitchv1.DbCreateReportRequest],
) (*connect.Response[snitchv1.DbCreateReportResponse], error) {
	db, err := r.service.getGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	queries := groupdb.New(db)

	// Ensure users and servers exist using sqlc
	if err := queries.EnsureUserExists(ctx, req.Msg.UserId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure user exists: %w", err))
	}
	if err := queries.EnsureUserExists(ctx, req.Msg.ReporterId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure reporter exists: %w", err))
	}
	if err := queries.EnsureServerExists(ctx, req.Msg.ServerId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to ensure server exists: %w", err))
	}

	// Create report using sqlc
	reportID, err := queries.CreateReport(ctx, groupdb.CreateReportParams{
		ReportText:       req.Msg.Reason,
		ReporterID:       req.Msg.ReporterId,
		ReportedUserID:   req.Msg.UserId,
		OriginServerID:   req.Msg.ServerId,
	})
	if err != nil {
		r.service.logger.Error("Failed to create report", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create report: %w", err))
	}

	response := &snitchv1.DbCreateReportResponse{
		ReportId: reportID,
	}

	r.service.logger.Info("Created report", "group_id", req.Msg.GroupId, "report_id", reportID)
	return connect.NewResponse(response), nil
}

// GetReport retrieves a specific report from the group database using sqlc
func (r *ReportRepository) GetReport(
	ctx context.Context,
	req *connect.Request[snitchv1.DbGetReportRequest],
) (*connect.Response[snitchv1.DbGetReportResponse], error) {
	db, err := r.service.getGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	queries := groupdb.New(db)

	// Get report using sqlc
	report, err := queries.GetReport(ctx, req.Msg.ReportId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("report not found: %d", req.Msg.ReportId))
		}
		r.service.logger.Error("Failed to get report", "group_id", req.Msg.GroupId, "report_id", req.Msg.ReportId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get report: %w", err))
	}

	response := &snitchv1.DbGetReportResponse{
		Id:         report.ReportID,
		Reason:     report.ReportText,
		ReporterId: report.ReporterID,
		UserId:     report.ReportedUserID,
		ServerId:   report.OriginServerID,
	}

	// Handle nullable CreatedAt field
	if report.CreatedAt.Valid {
		response.CreatedAt = report.CreatedAt.String
	}

	return connect.NewResponse(response), nil
}

// ListReports lists reports from the group database using sqlc
func (r *ReportRepository) ListReports(
	ctx context.Context,
	req *connect.Request[snitchv1.DbListReportsRequest],
) (*connect.Response[snitchv1.DbListReportsResponse], error) {
	db, err := r.service.getGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	queries := groupdb.New(db)

	var reportRows []groupdb.Report
	
	// Use appropriate sqlc query based on whether user_id filter is provided
	if req.Msg.UserId != nil {
		reportRows, err = queries.ListReportsByUser(ctx, *req.Msg.UserId)
	} else {
		reportRows, err = queries.ListReports(ctx)
	}
	
	if err != nil {
		r.service.logger.Error("Failed to list reports", "group_id", req.Msg.GroupId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list reports: %w", err))
	}

	// Apply limit and offset manually since sqlc doesn't support dynamic LIMIT/OFFSET
	start := 0
	if req.Msg.Offset != nil {
		start = int(*req.Msg.Offset)
	}
	
	end := len(reportRows)
	if req.Msg.Limit != nil {
		end = start + int(*req.Msg.Limit)
		if end > len(reportRows) {
			end = len(reportRows)
		}
	}
	
	if start >= len(reportRows) {
		reportRows = []groupdb.Report{}
	} else {
		reportRows = reportRows[start:end]
	}

	var reports []*snitchv1.DbGetReportResponse
	for _, reportRow := range reportRows {
		report := &snitchv1.DbGetReportResponse{
			Id:         reportRow.ReportID,
			Reason:     reportRow.ReportText,
			ReporterId: reportRow.ReporterID,
			UserId:     reportRow.ReportedUserID,
			ServerId:   reportRow.OriginServerID,
		}
		
		// Handle nullable CreatedAt field
		if reportRow.CreatedAt.Valid {
			report.CreatedAt = reportRow.CreatedAt.String
		}
		
		reports = append(reports, report)
	}

	response := &snitchv1.DbListReportsResponse{
		Reports: reports,
	}

	return connect.NewResponse(response), nil
}

// DeleteReport deletes a report from the group database using sqlc
func (r *ReportRepository) DeleteReport(
	ctx context.Context,
	req *connect.Request[snitchv1.DbDeleteReportRequest],
) (*connect.Response[emptypb.Empty], error) {
	db, err := r.service.getGroupDB(ctx, req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get group database: %w", err))
	}

	queries := groupdb.New(db)

	// Delete report using sqlc
	affected, err := queries.DeleteReport(ctx, req.Msg.ReportId)
	if err != nil {
		r.service.logger.Error("Failed to delete report", "group_id", req.Msg.GroupId, "report_id", req.Msg.ReportId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete report: %w", err))
	}

	if affected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("report not found: %d", req.Msg.ReportId))
	}

	r.service.logger.Info("Deleted report", "group_id", req.Msg.GroupId, "report_id", req.Msg.ReportId)
	return connect.NewResponse(&emptypb.Empty{}), nil
}