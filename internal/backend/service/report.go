package service

import (
	"context"
	"fmt"
	"log/slog"

	"snitch/internal/backend/dbclient"
	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"

	"connectrpc.com/connect"
)

type ReportServer struct {
	dbClient     *dbclient.Client
	eventService *EventService
}

func NewReportServer(dbClient *dbclient.Client, eventService *EventService) *ReportServer {
	return &ReportServer{
		dbClient:     dbClient,
		eventService: eventService,
	}
}

func (s *ReportServer) CreateReport(
	ctx context.Context,
	req *connect.Request[snitchv1.CreateReportRequest],
) (*connect.Response[snitchv1.CreateReportResponse], error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	// Get server ID from header
	serverID := req.Header().Get(ServerIDHeader)
	if serverID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, 
			fmt.Errorf("server ID header is required"))
	}

	// Find group ID for this server
	groupID, err := s.dbClient.FindGroupByServer(ctx, serverID)
	if err != nil {
		slogger.Error("Failed to find group for server", "server_id", serverID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Create the report
	reportID, err := s.dbClient.CreateReport(ctx, groupID, req.Msg.ReportedId, req.Msg.ReporterId, serverID, req.Msg.ReportText, nil)
	if err != nil {
		slogger.Error("Failed to create report", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Emit event
	event := &snitchv1.Event{
		Type:     snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		GroupId:  groupID,
		ServerId: serverID,
		Data: &snitchv1.Event_ReportCreated{
			ReportCreated: &snitchv1.ReportCreatedEvent{
				ReportId:   reportID,
				ReportedId: req.Msg.ReportedId,
				ReporterId: req.Msg.ReporterId,
				ReportText: req.Msg.ReportText,
			},
		},
	}
	if err := s.eventService.PublishEvent(ctx, event); err != nil {
		slogger.Warn("Failed to publish event", "error", err)
	}

	slogger.Info("Report created", "report_id", reportID, "group_id", groupID)

	return connect.NewResponse(&snitchv1.CreateReportResponse{
		ReportId: reportID,
	}), nil
}

func (s *ReportServer) ListReports(
	ctx context.Context,
	req *connect.Request[snitchv1.ListReportsRequest],
) (*connect.Response[snitchv1.ListReportsResponse], error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	// Get server ID from header
	serverID := req.Header().Get(ServerIDHeader)
	if serverID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, 
			fmt.Errorf("server ID header is required"))
	}

	// Find group ID for this server
	groupID, err := s.dbClient.FindGroupByServer(ctx, serverID)
	if err != nil {
		slogger.Error("Failed to find group for server", "server_id", serverID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// List reports - convert from old protobuf format to new format for now
	dbReports, err := s.dbClient.ListReports(ctx, groupID, req.Msg.ReportedId, nil, nil)
	if err != nil {
		slogger.Error("Failed to list reports", "group_id", groupID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert from database format to API format
	var reports []*snitchv1.CreateReportRequest
	for _, dbReport := range dbReports {
		reports = append(reports, &snitchv1.CreateReportRequest{
			ReportText: dbReport.Reason,
			ReporterId: dbReport.ReporterId,
			ReportedId: dbReport.UserId,
		})
	}

	return connect.NewResponse(&snitchv1.ListReportsResponse{
		Reports: reports,
	}), nil
}

func (s *ReportServer) DeleteReport(
	ctx context.Context,
	req *connect.Request[snitchv1.DeleteReportRequest],
) (*connect.Response[snitchv1.DeleteReportResponse], error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	// Get server ID from header
	serverID := req.Header().Get(ServerIDHeader)
	if serverID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, 
			fmt.Errorf("server ID header is required"))
	}

	// Find group ID for this server
	groupID, err := s.dbClient.FindGroupByServer(ctx, serverID)
	if err != nil {
		slogger.Error("Failed to find group for server", "server_id", serverID, "error", err)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Delete the report
	if err := s.dbClient.DeleteReport(ctx, groupID, req.Msg.ReportId); err != nil {
		slogger.Error("Failed to delete report", "group_id", groupID, "report_id", req.Msg.ReportId, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Emit event
	event := &snitchv1.Event{
		Type:     snitchv1.EventType_EVENT_TYPE_REPORT_DELETED,
		GroupId:  groupID,
		ServerId: serverID,
		Data: &snitchv1.Event_ReportDeleted{
			ReportDeleted: &snitchv1.ReportDeletedEvent{
				ReportId: req.Msg.ReportId,
			},
		},
	}
	if err := s.eventService.PublishEvent(ctx, event); err != nil {
		slogger.Warn("Failed to publish event", "error", err)
	}

	slogger.Info("Report deleted", "report_id", req.Msg.ReportId, "group_id", groupID)

	return connect.NewResponse(&snitchv1.DeleteReportResponse{
		ReportId: req.Msg.ReportId,
	}), nil
}