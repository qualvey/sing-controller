package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		s.API.Clash = &settings.EndpointAPI{Address: backend.URL, Secret: "s3cret"}
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

func TestChannelAvailabilityUsesClashWhenGRPCIsNotConfigured(t *testing.T) {
	backend := clashBackend(t, "")
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	config := `{"log":{"level":"info"},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	coreStore := store.New(configPath)
	if err := coreStore.Load(t.Context(), store.DefaultConfig{}); err != nil {
		t.Fatal(err)
	}
	controllerSettings := settings.New(filepath.Join(dir, "settings.json"))
	if err := controllerSettings.Load(); err != nil {
		t.Fatal(err)
	}
	if err := controllerSettings.Update(func(s *settings.Settings) error {
		s.API.Clash = &settings.EndpointAPI{Address: backend.URL}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(HandlerOptions{Store: coreStore, Settings: controllerSettings, Version: "test"})

	rec := doRequest(t, h, http.MethodGet, "/api/channels", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var got channelAvailability
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.GRPC.Configured || got.GRPC.Available {
		t.Fatalf("gRPC channel = %+v, want unconfigured and unavailable", got.GRPC)
	}
	if !got.Clash.Configured || !got.Clash.Available {
		t.Fatalf("Clash channel = %+v, want configured and available", got.Clash)
	}
}

func TestCoreHealthFallsBackToServiceAPIWhenClashAPIIsUnavailable(t *testing.T) {
	clashListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for unavailable Clash endpoint: %v", err)
	}
	clashAddress := clashListener.Addr().String()
	if err := clashListener.Close(); err != nil {
		t.Fatalf("close Clash endpoint: %v", err)
	}

	serviceListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for service API endpoint: %v", err)
	}
	defer serviceListener.Close()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	config := `{
		"services": [{"type":"api","listen":"127.0.0.1","listen_port":` + portFromAddress(t, serviceListener.Addr().String()) + `}]
	}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	coreStore := store.New(configPath)
	if err := coreStore.Load(t.Context(), store.DefaultConfig{}); err != nil {
		t.Fatalf("load config: %v", err)
	}

	controllerSettings := settings.New(filepath.Join(dir, "settings.json"))
	if err := controllerSettings.Load(); err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if err := controllerSettings.Update(func(s *settings.Settings) error {
		s.API.Clash = &settings.EndpointAPI{Address: "http://" + clashAddress}
		return nil
	}); err != nil {
		t.Fatalf("set unavailable Clash API: %v", err)
	}

	h := NewHandler(HandlerOptions{Store: coreStore, Settings: controllerSettings})
	status := h.probeCoreHealth(time.Second)
	if !status.Alive {
		t.Fatalf("core should be alive through service API fallback, got %+v", status)
	}
	if status.Version != "sing-box" {
		t.Fatalf("version = %q, want sing-box", status.Version)
	}
}

func portFromAddress(t *testing.T, address string) string {
	t.Helper()
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("split listener address %q: %v", address, err)
	}
	return port
}
