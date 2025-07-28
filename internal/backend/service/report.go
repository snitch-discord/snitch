package service

import (
	"context"
	"fmt"
	"log/slog"
	"snitch/internal/backend/dbconfig"
	"snitch/internal/backend/group"
	groupSQLc "snitch/internal/backend/group/gen/sqlc"
	"snitch/internal/backend/jwt"
	"snitch/internal/backend/service/interceptor"
	"snitch/internal/shared/ctxutil"
	snitchv1 "snitch/pkg/proto/gen/snitch/v1"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReportServer struct {
	tokenCache   *jwt.TokenCache
	libSQLConfig dbconfig.LibSQLConfig
	eventService *EventService
}

func NewReportServer(tokenCache *jwt.TokenCache, libSQLConfig dbconfig.LibSQLConfig, eventService *EventService) *ReportServer {
	return &ReportServer{tokenCache: tokenCache, libSQLConfig: libSQLConfig, eventService: eventService}
}

func reportDBtoRPC(reportRow groupSQLc.GetAllReportsRow) *snitchv1.CreateReportRequest {
	return &snitchv1.CreateReportRequest{
		ReportText: reportRow.ReportText,
		ReporterId: reportRow.ReporterID,
		ReportedId: reportRow.ReportedUserID,
	}
}

func newReportCreatedEvent(serverID string, reportID int, reporterID, reportedID, reportText, groupID string) *snitchv1.Event {
	return &snitchv1.Event{
		Type:      snitchv1.EventType_EVENT_TYPE_REPORT_CREATED,
		Timestamp: timestamppb.New(time.Now()),
		ServerId:  serverID,
		GroupId:   groupID,
		Data: &snitchv1.Event_ReportCreated{
			ReportCreated: &snitchv1.ReportCreatedEvent{
				ReportId:   int64(reportID),
				ReporterId: reporterID,
				ReportedId: reportedID,
				ReportText: reportText,
			},
		},
	}
}

func newReportDeletedEvent(serverID string, reportID int, groupID string) *snitchv1.Event {
	return &snitchv1.Event{
		Type:      snitchv1.EventType_EVENT_TYPE_REPORT_DELETED,
		Timestamp: timestamppb.New(time.Now()),
		ServerId:  serverID,
		GroupId:   groupID,
		Data: &snitchv1.Event_ReportDeleted{
			ReportDeleted: &snitchv1.ReportDeletedEvent{
				ReportId: int64(reportID),
			},
		},
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
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	serverID, err := interceptor.GetServerID(ctx)
	if err != nil {
		slogger.Error("Couldn't get server id", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	groupID, err := interceptor.GetGroupID(ctx)
	if err != nil {
		slogger.Error("Couldn't get group id", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	db, err := group.GetGroupDB(ctx, s.tokenCache.Get(), s.libSQLConfig, groupID)
	if err != nil {
		slogger.Error("Failed creating group db", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	queries := groupSQLc.New(db)

	if err := queries.AddUser(ctx, req.Msg.ReportedId); err != nil {
		slogger.Error(fmt.Sprintf("failed to add user %s", req.Msg.ReportedId), "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := queries.AddUser(ctx, req.Msg.ReporterId); err != nil {
		slogger.Error(fmt.Sprintf("failed to add user %s", req.Msg.ReportedId), "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	reportID, err := queries.CreateReport(ctx, groupSQLc.CreateReportParams{
		OriginServerID: serverID,
		ReportText:     req.Msg.ReportText,
		ReporterID:     req.Msg.ReporterId,
		ReportedUserID: req.Msg.ReportedId,
	})

	if err != nil {
		slogger.Error("failed to create report", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if s.eventService != nil {
		event := newReportCreatedEvent(serverID, reportID, req.Msg.ReporterId, req.Msg.ReportedId, req.Msg.ReportText, groupID)
		if err := s.eventService.PublishEvent(event); err != nil {
			slogger.Error("Failed to publish report created event", "error", err, "report_id", reportID)
			// Continue with request - event failure shouldn't fail the report creation
		}
	}

	return connect.NewResponse(&snitchv1.CreateReportResponse{
		ReportId: int64(reportID),
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

	groupID, err := interceptor.GetGroupID(ctx)
	if err != nil {
		slogger.Error("Couldn't get group id", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	db, err := group.GetGroupDB(ctx, s.tokenCache.Get(), s.libSQLConfig, groupID)
	if err != nil {
		slogger.Error("Failed getting group db", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	queries := groupSQLc.New(db)

	dbReports, err := queries.GetAllReports(ctx)

	if err != nil {
		slogger.Error("failed to get reports", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	rpcReports := make([]*snitchv1.CreateReportRequest, 0, len(dbReports))

	for _, dbReport := range dbReports {
		if req.Msg.ReporterId != nil {
			if dbReport.ReporterID != *req.Msg.ReporterId {
				continue
			}
		}

		if req.Msg.ReportedId != nil {
			if dbReport.ReportedUserID != *req.Msg.ReportedId {
				continue
			}
		}

		rpcReport := reportDBtoRPC(dbReport)
		rpcReports = append(rpcReports, rpcReport)
	}

	return connect.NewResponse(&snitchv1.ListReportsResponse{
		Reports: rpcReports,
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

	groupID, err := interceptor.GetGroupID(ctx)
	if err != nil {
		slogger.Error("Couldn't get group id", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	db, err := group.GetGroupDB(ctx, s.tokenCache.Get(), s.libSQLConfig, groupID)
	if err != nil {
		slogger.Error("Failed getting group db", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	queries := groupSQLc.New(db)
	deletedReportID, err := queries.DeleteReport(ctx, int(req.Msg.ReportId))
	if err != nil {
		slogger.Error("failed to delete report", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if s.eventService != nil {
		serverID, err := interceptor.GetServerID(ctx)
		if err != nil {
			slogger.Error("Failed to get server ID for report deleted event", "error", err, "report_id", deletedReportID)
		} else {
			event := newReportDeletedEvent(serverID, deletedReportID, groupID)
			if err := s.eventService.PublishEvent(event); err != nil {
				slogger.Error("Failed to publish report deleted event", "error", err, "report_id", deletedReportID)
				// Continue with request - event failure shouldn't fail the report deletion
			}
		}
	}

	return connect.NewResponse(&snitchv1.DeleteReportResponse{ReportId: int64(deletedReportID)}), nil
}
