package libsqladmin

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"snitch/internal/backend/dbconfig"
	"snitch/internal/shared/ctxutil"
)

func CreateNamespace(name string, ctx context.Context, token string, config dbconfig.LibSQLConfig) error {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	adminURL, err := config.AdminURL()
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, "POST",
		adminURL.JoinPath(fmt.Sprintf("v1/namespaces/%s/create", name)).String(),
		bytes.NewReader([]byte(`{"dump_url": null}`)))

	if err != nil {
		slogger.ErrorContext(ctx, "Failed creating metadata namespace create request", "Error", err)
		return err
	}

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed executing metadata namespace create request", "Error", err)
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slogger.ErrorContext(ctx, "Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusConflict {
		slogger.ErrorContext(ctx, "Unexpected status code", "Status", resp.Status)
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
}

func DoesNamespaceExist(name string, ctx context.Context, token string, config dbconfig.LibSQLConfig) (bool, error) {
	slogger, ok := ctxutil.Value[*slog.Logger](ctx)
	if !ok {
		slogger = slog.Default()
	}

	adminURL, err := config.AdminURL()
	if err != nil {
		return false, err
	}

	request, err := http.NewRequestWithContext(ctx, "GET",
		adminURL.JoinPath(fmt.Sprintf("v1/namespaces/%s/config", name)).String(),
		nil)

	if err != nil {
		slogger.ErrorContext(ctx, "Failed creating metadata namespace request", "Error", err)
		return false, err
	}

	request.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		slogger.ErrorContext(ctx, "Failed executing metadata namespace request", "Error", err)
		return false, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slogger.ErrorContext(ctx, "Failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode >= 300 {
		slogger.ErrorContext(ctx, "Unexpected status code", "Status", resp.Status)
		return false, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return true, nil
}
