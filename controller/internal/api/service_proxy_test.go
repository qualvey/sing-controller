package api

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
)

// grpcBackend 模拟 sing-box service API（gRPC-Web POST + WS 升级路径）
func grpcBackend(t *testing.T, secret string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if secret != "" {
			if got := r.Header.Get("Authorization"); got != "Bearer "+secret {
				t.Errorf("missing/wrong secret: got %q, want Bearer %s", got, secret)
			}
		}
		switch {
		case r.URL.Path == "/daemon.StartedService/GetVersion":
			w.Header().Set("Content-Type", "application/grpc-web+proto")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("grpc-web-bytes"))
		case r.URL.Path == "/daemon.StartedService/SubscribeConnections" && r.Header.Get("Upgrade") == "websocket":
			w.Header().Set("Upgrade", "websocket")
			w.Header().Set("Connection", "Upgrade")
			w.WriteHeader(http.StatusSwitchingProtocols)
			_, _ = w.Write([]byte("ws"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestStoreWithServices(t *testing.T, servicesJSON string) *store.Store {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{"log":{"level":"info"},"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}],"services":[` + servicesJSON + `]}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	st := store.New(cfgPath)
	if err := st.Load(t.Context(), store.DefaultConfig{}); err != nil {
		t.Fatalf("store load: %v", err)
	}
	return st
}

func handlerWith(t *testing.T, st *store.Store, set *settings.Manager) http.Handler {
	t.Helper()
	return NewHandler(HandlerOptions{Store: st, Settings: set, Version: "test"})
}

func TestServiceProxyFromConfigInference(t *testing.T) {
	backend := grpcBackend(t, "svc-secret")
	// sing-box 配置里 services[type=api] 指向 mock 地址（端口不同，用 settings 覆盖更直接，
	// 这里验证「从配置推断」路径：直接构造 listen=127.0.0.1 + 端口指向 backend）
	port := backend.Listener.Addr().(*net.TCPAddr).Port
	st := newTestStoreWithServices(t, `{"type":"api","listen":"127.0.0.1","listen_port":`+itoa(port)+`,"secret":"svc-secret"}`)
	dir := t.TempDir()
	set := settings.New(filepath.Join(dir, "settings.json"))
	_ = set.Load()
	h := handlerWith(t, st, set)

	rec := doRequest(t, h, http.MethodPost, "/api/grpc/daemon.StartedService/GetVersion", "body")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "grpc-web-bytes") {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestServiceProxySettingsOverride(t *testing.T) {
	backend := grpcBackend(t, "s3cret")
	dir := t.TempDir()
	set := settings.New(filepath.Join(dir, "settings.json"))
	_ = set.Load()
	_ = set.Update(func(s *settings.Settings) error {
		s.ServiceAPI = &settings.ServiceAPIOptions{Address: backend.URL, Secret: "s3cret"}
		return nil
	})
	// 配置里没有 services 段
	st := newTestStoreWithServices(t, ``)
	h := handlerWith(t, st, set)

	rec := doRequest(t, h, http.MethodPost, "/api/grpc/daemon.StartedService/GetVersion", "body")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestServiceProxyUnavailable(t *testing.T) {
	st := newTestStoreWithServices(t, ``)
	dir := t.TempDir()
	set := settings.New(filepath.Join(dir, "settings.json"))
	_ = set.Load()
	h := handlerWith(t, st, set)

	rec := doRequest(t, h, http.MethodPost, "/api/grpc/daemon.StartedService/GetVersion", "body")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
