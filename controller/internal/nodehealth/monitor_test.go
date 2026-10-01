package nodehealth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing-box/option"
)

func newTestRunner(t *testing.T) *Runner {
	t.Helper()
	cfg := settings.New(filepath.Join(t.TempDir(), "controller.json"))
	if err := cfg.Load(); err != nil {
		t.Fatal(err)
	}
	return New(cfg, nil)
}

func TestProbeClashUsesExistingDelayAPI(t *testing.T) {
	var receivedPath string
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.RequestURI
		receivedAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"delay":42}`))
	}))
	defer server.Close()

	runner := newTestRunner(t)
	options := settings.DefaultNodeHealthOptions()
	latency, err := runner.probeClash(context.Background(), options, server.URL, "probe-secret", "group/node")
	if err != nil {
		t.Fatal(err)
	}
	if latency != 42 {
		t.Fatalf("latency = %d, want 42", latency)
	}
	if !strings.Contains(receivedPath, "/proxies/group%2Fnode/delay") {
		t.Fatalf("request path = %q, expected escaped outbound tag", receivedPath)
	}
	if receivedAuth != "Bearer probe-secret" {
		t.Fatalf("authorization = %q", receivedAuth)
	}
}

func TestProxyTargetsExcludesGroupsAndSystemOutbounds(t *testing.T) {
	runner := newTestRunner(t)
	st := store.New(filepath.Join(t.TempDir(), "sing-box.json"))
	st.Options.Outbounds = []option.Outbound{
		{Type: "direct", Tag: "direct"},
		{Type: "selector", Tag: "Proxy"},
		{Type: "urltest", Tag: "Auto"},
		{Type: "vless", Tag: "edge-1"},
	}
	st.Options.Endpoints = []option.Endpoint{{Type: "wireguard", Tag: "wg-1"}}
	runner.store = st

	targets := runner.proxyTargets()
	if len(targets) != 2 || targets[0].Tag != "edge-1" || targets[1].Tag != "wg-1" {
		t.Fatalf("proxy targets = %#v", targets)
	}
}

func TestTriggerScanQueuesSingleRunAndRecordsCompletion(t *testing.T) {
	runner := newTestRunner(t)
	runner.store = store.New(filepath.Join(t.TempDir(), "sing-box.json"))
	if !runner.TriggerScan() {
		t.Fatal("first manual scan should be queued")
	}
	if !runner.Snapshot().ScanQueued {
		t.Fatal("snapshot should report queued scan")
	}
	if runner.TriggerScan() {
		t.Fatal("duplicate manual scan should be rejected")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := runner.Snapshot()
		if !snapshot.Scanning && !snapshot.ScanQueued && !snapshot.LastScanEnd.IsZero() {
			if snapshot.LastScanStart.IsZero() {
				t.Fatal("scan start timestamp was not recorded")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("manual scan did not complete")
}

func TestFeishuWebhookPayloadIsSignedWhenSecretIsConfigured(t *testing.T) {
	options := settings.DefaultNodeHealthOptions()
	options.WebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/test"
	options.WebhookToken = "generic-bearer-must-not-be-used"
	options.WebhookSignSecret = "feishu-sign-secret"
	event := Event{
		Event: "node.unavailable",
		Node: Status{
			Tag:   "edge-1",
			Type:  "vless",
			State: "unavailable",
			Error: "timeout",
		},
		ConsecutiveFails: 2,
		CheckedAt:        time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
	}

	body, useBearerToken, err := buildWebhookPayload(options, event)
	if err != nil {
		t.Fatal(err)
	}
	if useBearerToken {
		t.Fatal("Feishu webhook should not use generic bearer authorization")
	}
	var payload feishuMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.MsgType != "text" || !strings.Contains(payload.Content.Text, "edge-1") {
		t.Fatalf("unexpected Feishu payload: %#v", payload)
	}
	var timestamp int64
	if _, err := fmt.Sscan(payload.Timestamp, &timestamp); err != nil {
		t.Fatalf("invalid Feishu timestamp %q: %v", payload.Timestamp, err)
	}
	key := []byte(fmt.Sprintf("%d\n%s", timestamp, options.WebhookSignSecret))
	mac := hmac.New(sha256.New, key)
	wantSign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if payload.Sign != wantSign {
		t.Fatalf("signature = %q, want %q", payload.Sign, wantSign)
	}
}

func TestFeishuDetectionUsesDomainOnly(t *testing.T) {
	if !isFeishuWebhook("https://OPEN.FEISHU.CN/open-apis/bot/v2/hook/id") {
		t.Fatal("expected Feishu domain to be detected case-insensitively")
	}
	if isFeishuWebhook("https://hooks.example.com/feishu/webhook") {
		t.Fatal("Feishu keyword in URL path must not select Feishu mode")
	}
}

func TestFeishuSignatureMatchesOfficialExample(t *testing.T) {
	const want = "l1N0gAcBjdwBvGm1xMjOF0XSyaLRpR7tuO5dHfhAYc8="
	if got := signFeishuWebhook("demo", 1599360473); got != want {
		t.Fatalf("Feishu signature = %q, want %q", got, want)
	}
}

func TestWebhookResponseChecksFeishuApplicationCode(t *testing.T) {
	if err := webhookResponseError(http.StatusOK, []byte(`{"code":0,"msg":"success"}`), true); err != nil {
		t.Fatalf("successful Feishu response rejected: %v", err)
	}
	if err := webhookResponseError(http.StatusOK, []byte(`{"code":19024,"msg":"sign match fail"}`), true); err == nil || !strings.Contains(err.Error(), "19024") {
		t.Fatalf("Feishu application error was not reported: %v", err)
	}
	if err := webhookResponseError(http.StatusOK, []byte(`{"code":19024}`), false); err != nil {
		t.Fatalf("generic webhook response should only use HTTP status: %v", err)
	}
}

func TestWebhookOnlySendsAvailabilityTransitions(t *testing.T) {
	events := make(chan Event, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Errorf("decode event: %v", err)
		}
		events <- event
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	runner := newTestRunner(t)
	options := settings.DefaultNodeHealthOptions()
	options.FailureThreshold = 2
	options.WebhookURL = server.URL
	target := proxyTarget{Tag: "edge-1", Type: "vless"}

	runner.record(options, target, 30, nil) // Initial baseline should not notify.
	runner.record(options, target, 0, context.DeadlineExceeded)
	select {
	case event := <-events:
		t.Fatalf("unexpected notification before failure threshold: %#v", event)
	case <-time.After(50 * time.Millisecond):
	}
	runner.record(options, target, 0, context.DeadlineExceeded)
	assertEvent := func(want string) {
		t.Helper()
		select {
		case event := <-events:
			if event.Event != want || event.Node.Tag != target.Tag {
				t.Fatalf("event = %#v, want %s for %s", event, want, target.Tag)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for %s webhook", want)
		}
	}
	assertEvent("node.unavailable")

	runner.record(options, target, 0, context.DeadlineExceeded)
	select {
	case event := <-events:
		t.Fatalf("unexpected repeated outage notification: %#v", event)
	case <-time.After(50 * time.Millisecond):
	}
	runner.record(options, target, 25, nil)
	assertEvent("node.available")
}

func TestProbeFailureSchedulesOneMinuteRetry(t *testing.T) {
	runner := newTestRunner(t)
	st := store.New(filepath.Join(t.TempDir(), "sing-box.json"))
	st.Options.Outbounds = []option.Outbound{{Type: "vless", Tag: "edge-1"}}
	runner.store = st
	options := settings.DefaultNodeHealthOptions()
	target := proxyTarget{Tag: "edge-1", Type: "vless"}

	runner.record(options, target, 0, context.DeadlineExceeded)
	runner.mu.RLock()
	status := runner.statuses[target.Tag]
	retry, queued := runner.retries[target.Tag]
	runner.mu.RUnlock()
	if !queued || !retry.retryAt.Equal(status.CheckedAt.Add(time.Minute)) {
		t.Fatalf("retry was not scheduled one minute after failure: status=%#v retry=%#v queued=%v", status, retry, queued)
	}
	if status.State != "unknown" || status.ConsecutiveFails != 1 {
		t.Fatalf("first failure should be pending threshold: %#v", status)
	}
	if targets := runner.targetsForScan([]proxyTarget{target}, "scheduled", retry.retryAt.Add(-time.Nanosecond)); len(targets) != 0 {
		t.Fatalf("scheduled scan should not retry node before one minute: %#v", targets)
	}
	if targets := runner.targetsForScan([]proxyTarget{target}, "manual", retry.retryAt.Add(-time.Nanosecond)); len(targets) != 1 {
		t.Fatalf("manual scan should still include configured node: %#v", targets)
	}
	if due := runner.takeDueRetries(retry.retryAt.Add(-time.Nanosecond)); len(due) != 0 {
		t.Fatalf("retry ran early: %#v", due)
	}
	if due := runner.takeDueRetries(retry.retryAt); len(due) != 1 || due[0].Tag != target.Tag {
		t.Fatalf("retry not released at deadline: %#v", due)
	}

	runner.record(options, target, 0, context.DeadlineExceeded)
	runner.record(options, target, 20, nil)
	runner.mu.RLock()
	_, queued = runner.retries[target.Tag]
	status = runner.statuses[target.Tag]
	runner.mu.RUnlock()
	if queued || status.State != "available" || status.ConsecutiveFails != 0 {
		t.Fatalf("successful retry did not clear failures: status=%#v queued=%v", status, queued)
	}
}
