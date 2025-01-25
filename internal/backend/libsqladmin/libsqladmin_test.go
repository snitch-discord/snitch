package libsqladmin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"snitch/internal/backend/dbconfig"
)

func getHostAndPort(serverURL string) (host string, port string) {
	u, _ := url.Parse(serverURL)
	parts := strings.Split(u.Host, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], "80"
}

func TestCreateNamespace(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST request, got %s", r.Method)
			}
			if r.URL.Path != "/v1/namespaces/metadata/create" {
				t.Errorf("expected /v1/namespaces/metadata/create path, got %s", r.URL.Path)
			}
			if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
				t.Errorf("expected Bearer test-token, got %s", auth)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected application/json content-type, got %s", ct)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		host, port := getHostAndPort(server.URL)
		config := dbconfig.LibSQLConfig{
			Host:      host,
			AdminPort: port,
			AuthKey:   "test-token",
		}

		err := CreateNamespace("metadata", context.Background(), "test-token", config)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		host, port := getHostAndPort(server.URL)
		config := dbconfig.LibSQLConfig{
			Host:      host,
			Port:      port,
			AdminPort: port,
			AuthKey:   "test-token",
		}

		err := CreateNamespace("test", context.Background(), "test-token", config)

		if err == nil {
			t.Error("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unexpected status") {
			t.Errorf("expected 'unexpected status' error, got %v", err)
		}
	})
}
