package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"snitch/internal/backend/dbconfig"
	groupSQLc "snitch/internal/backend/group/sqlc"
	"snitch/internal/backend/libsqladmin"
	"strconv"

	"snitch/internal/backend/jwt"
	metadataSQLc "snitch/internal/backend/metadata/sqlc"
	"snitch/internal/shared/ctxutil"

	"github.com/google/uuid"
	_ "github.com/tursodatabase/go-libsql"
)

type registrationRequest struct {
	UserID    int    `json:"userId,string"`
	GroupID   string `json:"groupId,omitempty"`
	GroupName string `json:"groupName,omitempty"`
}

type registrationResponse struct {
	ServerID int    `json:"serverId,string"`
	GroupID  string `json:"groupId"`
}

const ServerIDHeader = "X-Server-ID"

func getServerIDFromHeader(r *http.Request) (int, error) {
	serverIDStr := r.Header.Get(ServerIDHeader)
	if serverIDStr == "" {
		return 0, fmt.Errorf("server ID header is required")
	}

	serverID, err := strconv.Atoi(serverIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid server ID format")
	}

	return serverID, nil
}

func CreateRegistrationHandler(tokenCache *jwt.TokenCache, metadataDB *sql.DB, libSqlConfig dbconfig.LibSQLConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slogger, ok := ctxutil.Value[*slog.Logger](r.Context())
		if !ok {
			slogger = slog.Default()
		}

		switch r.Method {
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")

			serverID, err := getServerIDFromHeader(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var registrationRequest registrationRequest
			err = json.NewDecoder(r.Body).Decode(&registrationRequest)
			defer r.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			metadataTx, err := metadataDB.BeginTx(r.Context(), nil)
			if err != nil {
				slogger.ErrorContext(r.Context(), "Failed to start metadata transaction", "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			defer func() {
				if err := metadataTx.Rollback(); err != nil {
					slogger.ErrorContext(r.Context(), "Failed to rollback transaction metadata", "Error", err)
				}
			}()

			metadataQueries := metadataSQLc.New(metadataTx)
			metadataQueries.WithTx(metadataTx)
			var groupID uuid.UUID

			previousGroupID, err := metadataQueries.FindGroupIDByServerID(r.Context(), serverID)
			if err == nil {
				http.Error(w, "Server is already registered to group: "+previousGroupID.String(), http.StatusConflict)
				return
			}

			if registrationRequest.GroupID != "" {
				// Join group flow
				groupID, err = uuid.Parse(registrationRequest.GroupID)
				if err != nil {
					http.Error(w, "Invalid group ID format", http.StatusBadRequest)
					return
				}

				exists, err := libsqladmin.DoesNamespaceExist(groupID.String(), r.Context(), tokenCache.Get(), libSqlConfig)
				if err != nil {
					slogger.ErrorContext(r.Context(), "Failed checking if namespace exists", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if !exists {
					http.Error(w, "Group does not exist", http.StatusNotFound)
					return
				}

				dbURL, err := libSqlConfig.NamespaceURL(groupID.String(), tokenCache.Get())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				newDB, err := sql.Open("libsql", dbURL.String())
				if err != nil {
					slogger.ErrorContext(r.Context(), "Failed to connect to db", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer newDB.Close()

				groupTx, err := newDB.BeginTx(r.Context(), nil)
				if err != nil {
					slogger.ErrorContext(r.Context(), "Failed to start group transaction", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer func() {
					if err := groupTx.Rollback(); err != nil {
						slogger.ErrorContext(r.Context(), "Failed to rollback transaction group", "Error", err)
					}
				}()

				groupQueries := groupSQLc.New(groupTx)
				groupQueries.WithTx(groupTx)

				if err := metadataQueries.AddServerToGroup(r.Context(), metadataSQLc.AddServerToGroupParams{
					GroupID:  groupID,
					ServerID: serverID,
				}); err != nil {
					slogger.ErrorContext(r.Context(), "Failed adding server to group metadata", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := groupQueries.AddServer(r.Context(), serverID); err != nil {
					slogger.ErrorContext(r.Context(), "Failed adding server to group", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := groupTx.Commit(); err != nil {
					slogger.ErrorContext(r.Context(), "Failed to commit group transaction", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := metadataTx.Commit(); err != nil {
					slogger.ErrorContext(r.Context(), "Failed to commit metadata transaction", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			} else {
				// Create new group flow
				if registrationRequest.GroupName == "" {
					http.Error(w, "Group name is required when creating a new group", http.StatusBadRequest)
					return
				}

				groupID = uuid.New()
				exists, err := libsqladmin.DoesNamespaceExist(groupID.String(), r.Context(), tokenCache.Get(), libSqlConfig)
				if err != nil {
					slogger.ErrorContext(r.Context(), "Failed checking if namespace exists", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if !exists {
					if err := libsqladmin.CreateNamespace(groupID.String(), r.Context(), tokenCache.Get(), libSqlConfig); err != nil {
						slogger.ErrorContext(r.Context(), "Failed creating namespace", "Error", err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
				}

				dbURL, err := libSqlConfig.NamespaceURL(groupID.String(), tokenCache.Get())
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				slogger.InfoContext(r.Context(), "DB URL", "URL", dbURL.String())

				newDB, err := sql.Open("libsql", dbURL.String())
				if err != nil {
					slogger.ErrorContext(r.Context(), "Failed to connect to db", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer newDB.Close()

				groupTx, err := newDB.BeginTx(r.Context(), nil)
				if err != nil {
					slogger.ErrorContext(r.Context(), "Failed to start group transaction", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				defer func() {
					if err := groupTx.Rollback(); err != nil {
						slogger.ErrorContext(r.Context(), "Failed to rollback transaction group", "Error", err)
					}
				}()

				groupQueries := groupSQLc.New(groupTx)
				groupQueries.WithTx(groupTx)

				if err := newDB.PingContext(r.Context()); err != nil {
					slogger.Error("Ping Database", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := groupQueries.CreateUserTable(r.Context()); err != nil {
					slogger.Error("Create User Table", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := groupQueries.CreateServerTable(r.Context()); err != nil {
					slogger.Error("Create Server Table", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := groupQueries.CreateReportTable(r.Context()); err != nil {
					slogger.Error("Create Group Table", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := metadataQueries.InsertGroup(r.Context(), metadataSQLc.InsertGroupParams{
					GroupID:   groupID,
					GroupName: registrationRequest.GroupName,
				}); err != nil {
					slogger.ErrorContext(r.Context(), "Insert Group to Metadata", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := metadataQueries.AddServerToGroup(r.Context(), metadataSQLc.AddServerToGroupParams{
					GroupID:  groupID,
					ServerID: serverID,
				}); err != nil {
					slogger.ErrorContext(r.Context(), "Failed adding server to group metadata", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := groupQueries.AddServer(r.Context(), serverID); err != nil {
					slogger.ErrorContext(r.Context(), "Failed adding server to group", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := groupTx.Commit(); err != nil {
					slogger.ErrorContext(r.Context(), "Failed to commit group transaction", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				if err := metadataTx.Commit(); err != nil {
					slogger.ErrorContext(r.Context(), "Failed to commit metadata transaction", "Error", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			slogger.InfoContext(r.Context(), "Registration completed",
				"groupID", groupID.String(),
				"serverID", serverID,
				"isNewGroup", registrationRequest.GroupID == "")

			if err = json.NewEncoder(w).Encode(registrationResponse{
				ServerID: serverID,
				GroupID:  groupID.String(),
			}); err != nil {
				slogger.Error("Encode Response", "Error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		default:
			http.Error(w, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}
