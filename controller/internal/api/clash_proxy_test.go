package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
)

// clashBackend 模拟 sing-box clash API：校验 Bearer secret + 返回固定 JSON
func clashBackend(t *testing.T, secret string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret != "" {
			if got := r.Header.Get("Authorization"); got != "Bearer "+secret {
				t.Errorf("missing/wrong secret: got %q, want Bearer %s", got, secret)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/version":
			_ = json.NewEncoder(w).Encode(map[string]string{"version": "1.14.0-test"})
		case "/proxies":
			_ = json.NewEncoder(w).Encode(map[string]any{"proxies": map[string]any{"Proxy": map[string]string{"type": "selector"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestHandlerWithClash(t *testing.T, cfgJSON string) http.Handler {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	st := store.New(cfgPath)
	if err := st.Load(t.Context(), store.DefaultConfig{}); err != nil {
		t.Fatalf("store load: %v", err)
	}
	set := settings.New(filepath.Join(dir, "settings.json"))
	if err := set.Load(); err != nil {
		t.Fatalf("settings load: %v", err)
	}
	return NewHandler(HandlerOptions{Store: st, Settings: set, Version: "test"})
}

func TestClashProxySettingsConfig(t *testing.T) {
	backend := clashBackend(t, "s3cret")
	cfg := `{"log":{"level":"info"},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`
	h := newTestHandlerWithClash(t, cfg)

	// settings 指向 mock 后端（显式配置优先路径）
	dir := t.TempDir()
	set := settings.New(filepath.Join(dir, "settings.json"))
	_ = set.Load()
	_ = set.Update(func(s *settings.Settings) error {
		s.ClashAPI = &settings.ClashAPIOptions{Address: backend.URL, Secret: "s3cret"}
		return nil
	})

	// 重建 handler（settings 变化后）
	dir2 := t.TempDir()
	cfg2 := filepath.Join(dir2, "config.json")
	_ = os.WriteFile(cfg2, []byte(cfg), 0o644)
	st := store.New(cfg2)
	_ = st.Load(t.Context(), store.DefaultConfig{})
	h = NewHandler(HandlerOptions{Store: st, Settings: set, Version: "test"})

	rec := doRequest(t, h, http.MethodGet, "/api/clash/version", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "1.14.0-test") {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestClashProxyUnavailable(t *testing.T) {
	h := newTestHandlerWithClash(t, `{"log":{"level":"info"},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)
	rec := doRequest(t, h, http.MethodGet, "/api/clash/version", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (clash API not configured)", rec.Code)
	}
}
