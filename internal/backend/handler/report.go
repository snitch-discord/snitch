package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"snitch/internal/backend/dbconfig"
	"snitch/internal/backend/group"
	groupSQLc "snitch/internal/backend/group/sqlc"
	"snitch/internal/backend/jwt"
	"snitch/internal/backend/middleware"
	"snitch/internal/shared/ctxutil"
	"time"
)

type reportRequest struct {
	ReportText     string `json:"reportText"`
	ReporterID     int    `json:"reporterId,string"`
	ReportedUserID int    `json:"reportedUserId,string"`
}

func CreateReportHandler(tokenCache *jwt.TokenCache, libSqlConfig dbconfig.LibSQLConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slogger, ok := ctxutil.Value[*slog.Logger](r.Context())
		if !ok {
			slogger = slog.Default()
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*5)
		defer cancel()

		serverID, err := middleware.GetServerID(ctx)
		if err != nil {
			slogger.Error("Couldn't get server id", "Error", err)
			http.Error(w, "Server ID not available", http.StatusInternalServerError)
			return
		}
		groupID, err := middleware.GetGroupID(ctx)
		if err != nil {
			slogger.Error("Couldn't get group id", "Error", err)
			http.Error(w, "Group ID not available", http.StatusInternalServerError)
			return
		}

		switch r.Method {
		case "GET":
			db, err := group.NewGroupDB(ctx, tokenCache.Get(), libSqlConfig, groupID)
			if err != nil {
				slogger.Error("Failed creating group db", "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			queries := groupSQLc.New(db)

			reports, err := queries.GetAllReports(r.Context())

			if err != nil {
				slogger.Error("failed to get reports", "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := json.NewEncoder(w).Encode(reports); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case "POST":

			var reportRequest reportRequest
			err := json.NewDecoder(r.Body).Decode(&reportRequest)
			defer r.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			db, err := group.NewGroupDB(ctx, tokenCache.Get(), libSqlConfig, groupID)
			if err != nil {
				slogger.Error("Failed creating group db", "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			queries := groupSQLc.New(db)

			if err := queries.AddUser(ctx, reportRequest.ReportedUserID); err != nil {
				slogger.Error(fmt.Sprintf("failed to add user %d", reportRequest.ReportedUserID), "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := queries.AddUser(ctx, reportRequest.ReporterID); err != nil {
				slogger.Error(fmt.Sprintf("failed to add user %d", reportRequest.ReporterID), "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			reportID, err := queries.CreateReport(r.Context(), groupSQLc.CreateReportParams{
				OriginServerID: serverID,
				ReportText:     reportRequest.ReportText,
				ReporterID:     reportRequest.ReporterID,
				ReportedUserID: reportRequest.ReportedUserID,
			})

			if err != nil {
				slogger.Error("failed to create report", "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")

			if err := json.NewEncoder(w).Encode(map[string]interface{}{"id": reportID}); err != nil {
				slogger.Error("failed to encode response", "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		default:
			http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
