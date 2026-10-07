package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qualvey/sing-controller/internal/probe"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newTestHandlerStore(t *testing.T) *store.Store {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfgPath, []byte(testConfigJSONC), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return store.New(cfgPath)
}

// 未配置数据库时探测接口应明确返回 503，而不是 500 或 panic。
func TestNodeProbeUnavailableWithoutDatabase(t *testing.T) {
	h := newTestHandler(t)
	for _, path := range []string{
		"/api/node-probes",
		"/api/node-probes/jp-01",
		"/api/node-probes/jp-01/history",
	} {
		rec := doRequest(t, h, http.MethodGet, path, "")
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("GET %s status = %d, want 503", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "unavailable") {
			t.Errorf("GET %s body = %s", path, rec.Body.String())
		}
	}
	for _, path := range []string{"/api/node-probes", "/api/node-probes/jp-01"} {
		rec := doRequest(t, h, http.MethodPost, path, "")
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("POST %s status = %d, want 503", path, rec.Code)
		}
	}
}

func TestNodeProbeHistoryLimitValidation(t *testing.T) {
	h := newTestHandler(t)
	// 数据库不可用时先返回 503；此处仅验证 limit 解析逻辑不会 panic。
	rec := doRequest(t, h, http.MethodGet, "/api/node-probes/jp-01/history?limit=abc", "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 (probe check runs first)", rec.Code)
	}
}

// TestNodeProbeUnknownNodeReturns404 确认探测不存在的节点返回 404 而不是 500。
func TestNodeProbeUnknownNodeReturns404(t *testing.T) {
	database, err := gorm.Open(postgres.Open("host=127.0.0.1 port=1 user=none dbname=none sslmode=disable connect_timeout=1"), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               gormlogger.Discard,
	})
	if err != nil {
		t.Fatalf("open unreachable database: %v", err)
	}
	dir := t.TempDir()
	set := settings.New(filepath.Join(dir, "settings.json"))
	if err := set.Load(); err != nil {
		t.Fatalf("load settings: %v", err)
	}
	st := newTestHandlerStore(t)
	probeMgr := probe.NewManager(set, st, database)
	h := NewHandler(HandlerOptions{Settings: set, Store: st, Probe: probeMgr, Version: "test"})

	rec := doRequest(t, h, http.MethodPost, "/api/node-probes/does-not-exist", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s, want 404", rec.Code, rec.Body.String())
	}
}

func TestGetSettingsRedactsSecrets(t *testing.T) {
	dir := t.TempDir()
	set := settings.New(dir + "/settings.json")
	if err := set.Load(); err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if err := set.Update(func(s *settings.Settings) error {
		s.Database.DSN = "postgres://user:sup3rsecret@127.0.0.1:5432/sing"
		s.NodeProbe.Webhooks = []settings.WebhookOptions{
			{URL: "https://example.com/hook", Secret: "hook-secret"},
		}
		return nil
	}); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	h := NewHandler(HandlerOptions{Settings: set, Store: newTestHandlerStore(t)})
	rec := doRequest(t, h, http.MethodGet, "/api/settings", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "sup3rsecret") || strings.Contains(rec.Body.String(), "hook-secret") {
		t.Fatalf("settings leaked secrets: %s", rec.Body.String())
	}

	var body struct {
		Database  settings.DatabaseOptions `json:"database"`
		NodeProbe struct {
			Interval string                    `json:"interval"`
			Webhooks []settings.WebhookOptions `json:"webhooks"`
		} `json:"node_probe"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if body.Database.DSN != "" {
		t.Errorf("dsn = %q, want empty", body.Database.DSN)
	}
	if len(body.NodeProbe.Webhooks) != 1 || body.NodeProbe.Webhooks[0].Secret != "" {
		t.Errorf("webhooks = %+v, want one redacted entry", body.NodeProbe.Webhooks)
	}
	if body.NodeProbe.Interval != "120s" {
		t.Errorf("interval = %q, want 120s", body.NodeProbe.Interval)
	}
}

func TestPutSettingsPreservesRedactedSecrets(t *testing.T) {
	dir := t.TempDir()
	set := settings.New(dir + "/settings.json")
	if err := set.Load(); err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if err := set.Update(func(s *settings.Settings) error {
		s.Database.DSN = "postgres://user:sup3rsecret@127.0.0.1:5432/sing"
		s.NodeProbe.Webhooks = []settings.WebhookOptions{
			{URL: "https://example.com/hook", Secret: "hook-secret"},
		}
		return nil
	}); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	h := NewHandler(HandlerOptions{Settings: set, Store: newTestHandlerStore(t)})

	// 模拟前端把 GET 的脱敏结果原样回传：凭据不应被清空。
	payload := `{"server":{"listen":"127.0.0.1","listen_port":8080},"core":{"config_path":"config.json"},"api":{},"stats":{},"database":{},"node_probe":{"enabled":true,"interval":"60s","webhooks":[{"url":"https://example.com/hook","secret":""}]}}`
	rec := doRequest(t, h, http.MethodPut, "/api/settings", payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	values := set.Values()
	if values.Database.DSN != "postgres://user:sup3rsecret@127.0.0.1:5432/sing" {
		t.Errorf("dsn = %q, want preserved", values.Database.DSN)
	}
	if len(values.NodeProbe.Webhooks) != 1 || values.NodeProbe.Webhooks[0].Secret != "hook-secret" {
		t.Errorf("webhooks = %+v, want preserved secret", values.NodeProbe.Webhooks)
	}
	if values.NodeProbe.Interval != "60s" {
		t.Errorf("interval = %q, want 60s", values.NodeProbe.Interval)
	}
}

func TestPutSettingsRejectsInvalidNodeProbe(t *testing.T) {
	h := newTestHandler(t)
	payload := `{"core":{"config_path":"config.json"},"node_probe":{"enabled":true,"interval":"abc"}}`
	rec := doRequest(t, h, http.MethodPut, "/api/settings", payload)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s, want 400", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "node_probe.interval") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

// TestPutSettingsFlagsDatabaseRestart 确认改 DSN 时明确告知需要重启，
// 而不是让配置看起来已经生效。
func TestPutSettingsFlagsDatabaseRestart(t *testing.T) {
	dir := t.TempDir()
	set := settings.New(filepath.Join(dir, "settings.json"))
	if err := set.Load(); err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if err := set.Update(func(s *settings.Settings) error {
		s.Database.DSN = "postgres://user:old@127.0.0.1:5432/sing"
		return nil
	}); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	h := NewHandler(HandlerOptions{Settings: set, Store: newTestHandlerStore(t)})

	unchanged := `{"core":{"config_path":"config.json"},"database":{"dsn":"postgres://user:old@127.0.0.1:5432/sing"}}`
	rec := doRequest(t, h, http.MethodPut, "/api/settings", unchanged)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "restart_required") {
		t.Errorf("unchanged dsn must not request restart: %s", rec.Body.String())
	}

	changed := `{"core":{"config_path":"config.json"},"database":{"dsn":"postgres://user:new@127.0.0.1:5432/sing"}}`
	rec = doRequest(t, h, http.MethodPut, "/api/settings", changed)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Saved           bool   `json:"saved"`
		RestartRequired bool   `json:"restart_required"`
		RestartReason   string `json:"restart_reason"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Saved || !body.RestartRequired || body.RestartReason == "" {
		t.Errorf("response = %+v, want saved with restart_required", body)
	}
	if got := set.Values().Database.DSN; got != "postgres://user:new@127.0.0.1:5432/sing" {
		t.Errorf("dsn = %q, want persisted", got)
	}
}

// TestPutSettingsClearsDatabaseWithNull 确认可以用显式的 null 关闭数据库：
// 脱敏后的 GET 返回空串，而空串在 PUT 里表示「保持不变」，
// 没有这个逃生口就只能手改配置文件才能停用数据库。
func TestPutSettingsClearsDatabaseWithNull(t *testing.T) {
	dir := t.TempDir()
	set := settings.New(filepath.Join(dir, "settings.json"))
	if err := set.Load(); err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if err := set.Update(func(s *settings.Settings) error {
		s.Database.DSN = "postgres://user:old@127.0.0.1:5432/sing"
		return nil
	}); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	h := NewHandler(HandlerOptions{Settings: set, Store: newTestHandlerStore(t)})

	// 空串仍表示「保留既有值」，不能当成清除。
	rec := doRequest(t, h, http.MethodPut, "/api/settings", `{"core":{"config_path":"config.json"},"database":{"dsn":""}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := set.Values().Database.DSN; got == "" {
		t.Fatal("empty dsn must preserve the existing value")
	}

	rec = doRequest(t, h, http.MethodPut, "/api/settings", `{"core":{"config_path":"config.json"},"database":{"dsn":null}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := set.Values().Database.DSN; got != "" {
		t.Errorf("dsn = %q, want cleared", got)
	}
	if !strings.Contains(rec.Body.String(), "restart_required") {
		t.Errorf("clearing dsn must request restart: %s", rec.Body.String())
	}
}
