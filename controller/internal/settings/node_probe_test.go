package settings

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNodeProbeDefaults(t *testing.T) {
	s := defaultSettings()
	if !s.NodeProbe.Enabled {
		t.Error("node_probe.enabled should default to true")
	}
	if s.NodeProbe.Interval != "120s" {
		t.Errorf("interval = %q, want 120s", s.NodeProbe.Interval)
	}
	if got := s.NodeProbe.IntervalDuration(); got != 120*time.Second {
		t.Errorf("IntervalDuration = %s, want 120s", got)
	}
	if got := s.NodeProbe.TimeoutDuration(); got != 10*time.Second {
		t.Errorf("TimeoutDuration = %s, want 10s", got)
	}
	if s.NodeProbe.HistoryRetentionDays != 30 {
		t.Errorf("retention = %d, want 30", s.NodeProbe.HistoryRetentionDays)
	}
	if s.Database.DSN != "" {
		t.Errorf("database.dsn should default to empty, got %q", s.Database.DSN)
	}
}

func TestNodeProbeDurationFallback(t *testing.T) {
	cases := []struct {
		interval string
		want     time.Duration
	}{
		{"", 120 * time.Second},
		{"  ", 120 * time.Second},
		{"0s", 120 * time.Second},
		{"-5m", 120 * time.Second},
		{"not-a-duration", 120 * time.Second},
		{"45s", 45 * time.Second},
		{"2m", 2 * time.Minute},
	}
	for _, tc := range cases {
		o := NodeProbeOptions{Interval: tc.interval}
		if got := o.IntervalDuration(); got != tc.want {
			t.Errorf("IntervalDuration(%q) = %s, want %s", tc.interval, got, tc.want)
		}
	}
}

func TestNodeProbeNormalizeAndValidate(t *testing.T) {
	o := &NodeProbeOptions{}
	o.Normalize()
	if o.Interval != "120s" || o.TestURL == "" || o.Timeout == "" || o.HistoryRetentionDays != 30 {
		t.Fatalf("normalize = %+v", o)
	}
	if err := o.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	o.Interval = "10"
	if err := o.Validate(); err == nil {
		t.Error("expected error for invalid interval")
	}
	o.Interval = "60s"

	o.Timeout = "0s"
	if err := o.Validate(); err == nil {
		t.Error("expected error for non-positive timeout")
	}
	o.Timeout = "5s"

	o.TestURL = "not a url"
	if err := o.Validate(); err == nil {
		t.Error("expected error for invalid test_url")
	}
	o.TestURL = "https://example.com/generate_204"

	o.HistoryRetentionDays = -1
	if err := o.Validate(); err == nil {
		t.Error("expected error for negative retention")
	}
	o.HistoryRetentionDays = 7

	o.Webhooks = []WebhookOptions{{URL: "ftp://example.com/hook"}}
	if err := o.Validate(); err == nil {
		t.Error("expected error for non-http webhook URL")
	}
	o.Webhooks = []WebhookOptions{{URL: "https://example.com/hook", Secret: "s"}}
	if err := o.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestLoadFillsNodeProbeDefaultsForExistingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	content := `{"server":{"listen":"127.0.0.1","listen_port":9999}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	m := New(path)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	values := m.Values()
	if values.NodeProbe.Interval != "120s" {
		t.Errorf("interval = %q, want 120s (normalized and persisted)", values.NodeProbe.Interval)
	}
	if !values.NodeProbe.Enabled {
		t.Error("enabled should default to true for existing configs")
	}
	if err := m.Update(func(s *Settings) error {
		s.Database.DSN = "postgres://user:pass@127.0.0.1:5432/db?sslmode=disable"
		s.NodeProbe.Webhooks = []WebhookOptions{{URL: "https://example.com/hook", Secret: "s"}}
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated := m.Values()
	if updated.Database.DSN == "" || len(updated.NodeProbe.Webhooks) != 1 {
		t.Fatalf("update not applied: %+v", updated)
	}
	// A client that omits node_probe entirely must not silently disable probing.
	if err := m.Update(func(s *Settings) error {
		s.NodeProbe = NodeProbeOptions{}
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	final := m.Values()
	if final.NodeProbe.Interval != "120s" {
		t.Errorf("interval after empty replace = %q, want 120s", final.NodeProbe.Interval)
	}
}

// TestUpdateKeepsPreviousValuesOnFailure 确认写盘/校验失败不会污染内存配置。
func TestUpdateKeepsPreviousValuesOnFailure(t *testing.T) {
	m := New(filepath.Join(t.TempDir(), "settings.json"))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := m.Update(func(s *Settings) error {
		s.NodeProbe.Interval = "60s"
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// 回调自身返回错误：内存配置应保持不变。
	err := m.Update(func(s *Settings) error {
		s.NodeProbe.Interval = "5m"
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("Update error = %v, want %v", err, errBoom)
	}
	if got := m.Values().NodeProbe.Interval; got != "60s" {
		t.Errorf("interval = %q, want 60s", got)
	}

	// 校验失败：内存配置同样保持不变。
	if err := m.Update(func(s *Settings) error {
		s.NodeProbe.Interval = "not-a-duration"
		return nil
	}); err == nil {
		t.Fatal("expected validation error")
	}
	if got := m.Values().NodeProbe.Interval; got != "60s" {
		t.Errorf("interval after failed validation = %q, want 60s", got)
	}
}

var errBoom = errors.New("boom")

// TestValuesDeepCopiesWebhooks 确认 Values() 不泄露内部切片：
// 调用方在锁外改写返回值不应影响后续读取（否则会与探测后台任务形成数据竞争）。
func TestValuesDeepCopiesWebhooks(t *testing.T) {
	m := New(filepath.Join(t.TempDir(), "settings.json"))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := m.Update(func(s *Settings) error {
		s.NodeProbe.Webhooks = []WebhookOptions{{URL: " https://a.example/hook ", Secret: "s1"}}
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	snapshot := m.Values()
	snapshot.NodeProbe.Webhooks[0].URL = "https://tampered.example/"
	snapshot.NodeProbe.Webhooks[0].Secret = "tampered"
	fresh := m.Values()
	if len(fresh.NodeProbe.Webhooks) != 1 {
		t.Fatalf("webhooks = %+v", fresh.NodeProbe.Webhooks)
	}
	if fresh.NodeProbe.Webhooks[0].URL != "https://a.example/hook" || fresh.NodeProbe.Webhooks[0].Secret != "s1" {
		t.Errorf("internal webhooks were mutated through Values(): %+v", fresh.NodeProbe.Webhooks[0])
	}
}

// TestUpdateNormalizeDoesNotMutatePreviousWebhooks 确认 Update 里的
// Normalize 不会写到上一次配置共享的底层数组上。
func TestUpdateNormalizeDoesNotMutatePreviousWebhooks(t *testing.T) {
	m := New(filepath.Join(t.TempDir(), "settings.json"))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := m.Update(func(s *Settings) error {
		s.NodeProbe.Webhooks = []WebhookOptions{{URL: "https://a.example/hook"}}
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// 模拟 API 层「缺 node_probe 键 → 回填当前值」后再触发 Normalize 的路径。
	replacement := m.Values()
	replacement.NodeProbe.Webhooks = []WebhookOptions{{URL: "  https://b.example/hook  "}}
	replacement.NodeProbe.Normalize()
	if got := m.Values().NodeProbe.Webhooks[0].URL; got != "https://a.example/hook" {
		t.Errorf("stored webhook url = %q, want untouched", got)
	}
}
