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
)

type ReportServer struct {
	tokenCache   *jwt.TokenCache
	libSQLConfig dbconfig.LibSQLConfig
}

func NewReportServer(tokenCache *jwt.TokenCache, libSQLConfig dbconfig.LibSQLConfig) *ReportServer {
	return &ReportServer{tokenCache: tokenCache, libSQLConfig: libSQLConfig}
}

func reportDBtoRPC(reportRow groupSQLc.GetAllReportsRow) *snitchv1.CreateReportRequest {
	return &snitchv1.CreateReportRequest{
		ReportText: reportRow.ReportText,
		ReporterId: reportRow.ReporterID,
		ReportedId: reportRow.ReportedUserID,
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
	deletedReportID, err := queries.DeleteReport(ctx, req.Msg.ReportId)
	if err != nil {
		slogger.Error("failed to delete report", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&snitchv1.DeleteReportResponse{ReportId: deletedReportID}), nil
}
