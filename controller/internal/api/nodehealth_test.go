package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qualvey/sing-controller/internal/nodehealth"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
)

func newNodeHealthHandler(t *testing.T) (*Handler, *nodehealth.Runner) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "sing-box.json")
	if err := os.WriteFile(configPath, []byte(`{"outbounds":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	controllerSettings := settings.New(filepath.Join(dir, "controller.json"))
	if err := controllerSettings.Load(); err != nil {
		t.Fatal(err)
	}
	st := store.New(configPath)
	runner := nodehealth.New(controllerSettings, st)
	return NewHandler(HandlerOptions{
		Store:      st,
		Settings:   controllerSettings,
		NodeHealth: runner,
	}), runner
}

func TestNodeHealthConfigMasksAndPreservesWebhookToken(t *testing.T) {
	h, runner := newNodeHealthHandler(t)
	body := `{"enabled":true,"interval":"1m","test_url":"https://example.com/ping","timeout_ms":3000,"failure_threshold":3,"concurrency":4,"webhook_url":"https://hooks.example.com/event","webhook_token":"webhook-token-sentinel","webhook_sign_secret":"feishu-sign-sentinel"}`
	put := httptest.NewRecorder()
	h.ServeHTTP(put, httptest.NewRequest(http.MethodPut, "/api/node-health/config", strings.NewReader(body)))
	if put.Code != http.StatusOK {
		t.Fatalf("PUT status = %d: %s", put.Code, put.Body.String())
	}

	get := httptest.NewRecorder()
	h.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/node-health/config", nil))
	if get.Code != http.StatusOK {
		t.Fatalf("GET status = %d: %s", get.Code, get.Body.String())
	}
	if strings.Contains(get.Body.String(), "webhook-token-sentinel") || strings.Contains(get.Body.String(), "feishu-sign-sentinel") ||
		!strings.Contains(get.Body.String(), `"webhook_token_configured":true`) ||
		!strings.Contains(get.Body.String(), `"webhook_sign_secret_configured":true`) {
		t.Fatalf("GET should mask the saved webhook token: %s", get.Body.String())
	}
	settingsGet := httptest.NewRecorder()
	h.ServeHTTP(settingsGet, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
	if strings.Contains(settingsGet.Body.String(), "webhook-token-sentinel") || strings.Contains(settingsGet.Body.String(), "feishu-sign-sentinel") {
		t.Fatalf("GET /api/settings should mask the saved webhook token: %s", settingsGet.Body.String())
	}

	put = httptest.NewRecorder()
	h.ServeHTTP(put, httptest.NewRequest(http.MethodPut, "/api/node-health/config", strings.NewReader(`{"enabled":true,"interval":"1m","test_url":"https://example.com/ping","timeout_ms":3000,"failure_threshold":3,"concurrency":4,"webhook_url":"https://hooks.example.com/event"}`)))
	if put.Code != http.StatusOK {
		t.Fatalf("PUT without token status = %d: %s", put.Code, put.Body.String())
	}
	if got := runner.Config().WebhookToken; got != "webhook-token-sentinel" {
		t.Fatalf("saved webhook token = %q, want preserved secret", got)
	}
	if got := runner.Config().WebhookSignSecret; got != "feishu-sign-sentinel" {
		t.Fatalf("saved Feishu signing secret = %q, want preserved secret", got)
	}

	put = httptest.NewRecorder()
	h.ServeHTTP(put, httptest.NewRequest(http.MethodPut, "/api/node-health/config", strings.NewReader(`{"enabled":true,"interval":"1m","test_url":"https://example.com/ping","timeout_ms":3000,"failure_threshold":3,"concurrency":4,"webhook_url":"https://hooks.example.com/event","clear_webhook_token":true,"clear_webhook_sign_secret":true}`)))
	if put.Code != http.StatusOK {
		t.Fatalf("PUT clearing token status = %d: %s", put.Code, put.Body.String())
	}
	if got := runner.Config().WebhookToken; got != "" {
		t.Fatalf("saved webhook token = %q, want cleared", got)
	}
	if got := runner.Config().WebhookSignSecret; got != "" {
		t.Fatalf("saved Feishu signing secret = %q, want cleared", got)
	}
}

func TestNodeHealthConfigRejectsInvalidInterval(t *testing.T) {
	h, _ := newNodeHealthHandler(t)
	body := `{"enabled":true,"interval":"1s","test_url":"https://example.com/ping","timeout_ms":3000,"failure_threshold":2,"concurrency":3}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/node-health/config", strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func TestManualNodeHealthScanAPIQueuesOnce(t *testing.T) {
	h, _ := newNodeHealthHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/node-health/scan", nil))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("first scan status = %d, want 202: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/node-health/scan", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate scan status = %d, want 409: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/node-health", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"scan_queued":true`) {
		t.Fatalf("health status should show queued scan: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
