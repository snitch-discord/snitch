package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"snitch/internal/backend/dbconfig"
	groupSQLc "snitch/internal/backend/group/sqlc"
	"snitch/internal/backend/jwt"
	"snitch/internal/backend/libsqladmin"
	metadataSQLc "snitch/internal/backend/metadata/sqlc"
	"snitch/internal/shared/ctxutil"
	snitchpb "snitch/pkg/proto/gen/snitch/v1"
	"strconv"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type RegisterServer struct {
	tokenCache   *jwt.TokenCache
	metadataDB   *sql.DB
	libSQLConfig dbconfig.LibSQLConfig
}

func NewRegisterServer(tokenCache *jwt.TokenCache, metadataDB *sql.DB, libSQLConfig dbconfig.LibSQLConfig) *RegisterServer {
	return &RegisterServer{tokenCache: tokenCache, metadataDB: metadataDB, libSQLConfig: libSQLConfig}
}

const ServerIDHeader = "X-Server-ID"

func getServerIDFromHeader(r *connect.Request[snitchpb.RegisterRequest]) (int, error) {
	serverIDStr := r.Header().Get(ServerIDHeader)
	if serverIDStr == "" {
		return 0, fmt.Errorf("server ID header is required")
	}

	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid server ID format")
	}

	return serverID, nil
}

func (s *RegisterServer) Register(
	ctx context.Context,
	req *connect.Request[snitchpb.RegisterRequest],
) (*connect.Response[snitchpb.RegisterResponse], error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}
	serverID, err := getServerIDFromHeader(req)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	metadataTx, err := s.metadataDB.BeginTx(ctx, nil)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed to start metadata transaction", "Error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	defer func() {
		if err := metadataTx.Rollback(); !errors.Is(err, sql.ErrTxDone) {
			slogger.ErrorContext(ctx, "Failed to rollback transaction metadata", "Error", err)
		}
	}()

	metadataQueries := metadataSQLc.New(metadataTx)
	metadataQueries.WithTx(metadataTx)
	var groupID uuid.UUID

	previousGroupID, err := metadataQueries.FindGroupIDByServerID(ctx, serverID)
	if err == nil {
		slogger.ErrorContext(ctx, "Server is already registered to group: "+previousGroupID.String())
		return nil, connect.NewError(connect.CodeAlreadyExists, err)
	}

	slogger.ErrorContext(ctx, "Msg", "Request", req.Msg)

	if req.Msg.GroupId != nil {
		// Join group flow
		groupID, err = uuid.Parse(*req.Msg.GroupId)
		if err != nil {
			slogger.ErrorContext(ctx, "Invalid group ID format")
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		exists, err := libsqladmin.DoesNamespaceExist(groupID.String(), ctx, s.tokenCache.Get(), s.libSQLConfig)
		if err != nil {
			slogger.ErrorContext(ctx, "Failed checking if namespace exists")
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if !exists {
			slogger.ErrorContext(ctx, "Group does not exist")
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		dbURL, err := s.libSQLConfig.NamespaceURL(groupID.String(), s.tokenCache.Get())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		newDB, err := sql.Open("libsql", dbURL.String())
		if err != nil {
			slogger.ErrorContext(ctx, "Failed to connect to db", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		defer newDB.Close()

		groupTx, err := newDB.BeginTx(ctx, nil)
		if err != nil {
			slogger.ErrorContext(ctx, "Failed to start group transaction", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		defer func() {
			if err := groupTx.Rollback(); !errors.Is(err, sql.ErrTxDone) {
				slogger.ErrorContext(ctx, "Failed to rollback transaction group", "Error", err)
			}
		}()

		groupQueries := groupSQLc.New(groupTx)
		groupQueries.WithTx(groupTx)

		if err := metadataQueries.AddServerToGroup(ctx, metadataSQLc.AddServerToGroupParams{
			GroupID:  groupID,
			ServerID: serverID,
		}); err != nil {
			slogger.ErrorContext(ctx, "Failed adding server to group metadata", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := groupQueries.AddServer(ctx, serverID); err != nil {
			slogger.ErrorContext(ctx, "Failed adding server to group", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := groupTx.Commit(); err != nil {
			slogger.ErrorContext(ctx, "Failed to commit group transaction", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := metadataTx.Commit(); err != nil {
			slogger.ErrorContext(ctx, "Failed to commit metadata transaction", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else {
		// Create new group flow
		if *req.Msg.GroupName == "" {
			slogger.ErrorContext(ctx, "Group name is required when creating a new group")
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		groupID = uuid.New()
		exists, err := libsqladmin.DoesNamespaceExist(groupID.String(), ctx, s.tokenCache.Get(), s.libSQLConfig)
		if err != nil {
			slogger.ErrorContext(ctx, "Failed checking if namespace exists", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if !exists {
			if err := libsqladmin.CreateNamespace(groupID.String(), ctx, s.tokenCache.Get(), s.libSQLConfig); err != nil {
				slogger.ErrorContext(ctx, "Failed creating namespace", "Error", err)
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}

		dbURL, err := s.libSQLConfig.NamespaceURL(groupID.String(), s.tokenCache.Get())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		slogger.InfoContext(ctx, "DB URL", "URL", dbURL.String())

		newDB, err := sql.Open("libsql", dbURL.String())
		if err != nil {
			slogger.ErrorContext(ctx, "Failed to connect to db", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		defer newDB.Close()

		groupTx, err := newDB.BeginTx(ctx, nil)
		if err != nil {
			slogger.ErrorContext(ctx, "Failed to start group transaction", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		defer func() {
			if err := groupTx.Rollback(); !errors.Is(err, sql.ErrTxDone) {
				slogger.ErrorContext(ctx, "Failed to rollback transaction group", "Error", err)
			}
		}()

		groupQueries := groupSQLc.New(groupTx)
		groupQueries.WithTx(groupTx)

		if err := newDB.PingContext(ctx); err != nil {
			slogger.Error("Ping Database", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := groupQueries.CreateUserTable(ctx); err != nil {
			slogger.Error("Create User Table", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := groupQueries.CreateServerTable(ctx); err != nil {
			slogger.Error("Create Server Table", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := groupQueries.CreateReportTable(ctx); err != nil {
			slogger.Error("Create Group Table", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := metadataQueries.InsertGroup(ctx, metadataSQLc.InsertGroupParams{
			GroupID:   groupID,
			GroupName: *req.Msg.GroupName,
		}); err != nil {
			slogger.ErrorContext(ctx, "Insert Group to Metadata", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := metadataQueries.AddServerToGroup(ctx, metadataSQLc.AddServerToGroupParams{
			GroupID:  groupID,
			ServerID: serverID,
		}); err != nil {
			slogger.ErrorContext(ctx, "Failed adding server to group metadata", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := groupQueries.AddServer(ctx, serverID); err != nil {
			slogger.ErrorContext(ctx, "Failed adding server to group", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := groupTx.Commit(); err != nil {
			slogger.ErrorContext(ctx, "Failed to commit group transaction", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if err := metadataTx.Commit(); err != nil {
			slogger.ErrorContext(ctx, "Failed to commit metadata transaction", "Error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	slogger.InfoContext(ctx, "Registration completed",
		"groupID", groupID.String(),
		"serverID", serverID,
		"isNewGroup", *req.Msg.GroupId == "")

	return connect.NewResponse(&snitchpb.RegisterResponse{
		ServerId: int32(serverID),
		GroupId:  groupID.String(),
	}), nil
}
