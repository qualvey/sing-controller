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
