package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"snitch/internal/backend/dbconfig"
	"snitch/internal/backend/handler"
	"snitch/internal/backend/jwt"
	"testing"

	_ "github.com/tursodatabase/go-libsql"
)

func TestCreateRegistrationHandler(t *testing.T) {
	tests := []struct {
		name           string
		serverID       string
		requestBody    map[string]interface{}
		expectedStatus int
		validateResp   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "missing server ID header",
			serverID: "",
			requestBody: map[string]interface{}{
				"userId":    "123",
				"groupName": "test group",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid server ID format",
			serverID: "not-a-number",
			requestBody: map[string]interface{}{
				"userId":    "123",
				"groupName": "test group",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "missing required fields",
			serverID: "1",
			requestBody: map[string]interface{}{
				"userId": "123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:     "invalid group ID format",
			serverID: "1",
			requestBody: map[string]interface{}{
				"userId":  "123",
				"groupId": "not-a-uuid",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenCache := &jwt.TokenCache{}
			db, err := sql.Open("libsql", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			config := dbconfig.LibSQLConfig{
				Host:      "localhost",
				Port:      "8080",
				AdminPort: "9090",
				AuthKey:   "test-key",
			}

			handler := handler.CreateRegistrationHandler(tokenCache, db, config)

			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodPost, "/databases", bytes.NewReader(body))
			if tt.serverID != "" {
				req.Header.Set("X-Server-ID", tt.serverID)
			}
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, recorder.Code)
			}

			if tt.validateResp != nil {
				tt.validateResp(t, recorder)
			}
		})
	}
}
