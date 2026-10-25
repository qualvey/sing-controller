package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qualvey/sing-controller/internal/models"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing-box/option"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return store.New(filepath.Join(t.TempDir(), "config.json"))
}

// newUnreachableDB 返回一个能构造出 *gorm.DB、但任何查询都会立刻失败的连接。
// 关闭自动 ping 后 gorm 不会在建连阶段报错，因此不需要真实数据库即可
// 驱动「落库失败」这条分支（用于验证生命周期顺序，不验证 SQL）。
func newUnreachableDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(postgres.Open("host=127.0.0.1 port=1 user=none dbname=none sslmode=disable connect_timeout=1"), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               gormlogger.Discard,
	})
	if err != nil {
		t.Fatalf("open unreachable database: %v", err)
	}
	return database
}

func TestTargetsSkipsGroupsAndSpecialOutbounds(t *testing.T) {
	st := newTestStore(t)
	st.Options = option.Options{
		Outbounds: []option.Outbound{
			{Type: "direct", Tag: "direct"},
			{Type: "selector", Tag: "proxy"},
			{Type: "urltest", Tag: "auto"},
			{Type: "vless", Tag: " jp-01 "},
			{Type: "vless", Tag: "jp-01"},
			{Type: "", Tag: "no-type"},
		},
		Endpoints: []option.Endpoint{
			{Type: "wireguard", Tag: "wg-01"},
			{Type: "selector", Tag: "ep-group"},
		},
	}
	m := NewManager(nil, st, nil)
	targets := m.targets()
	got := make(map[string]string, len(targets))
	for _, target := range targets {
		got[target.Tag] = target.Type
	}
	want := map[string]string{
		"jp-01": "vless",
		"wg-01": "wireguard",
	}
	if len(got) != len(want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	for tag, typ := range want {
		if got[tag] != typ {
			t.Errorf("target %q = %q, want %q", tag, got[tag], typ)
		}
	}
}

func TestProbeableType(t *testing.T) {
	skipped := []string{"", "direct", "block", "dns", "selector", "urltest", "fallback", "loadbalance", " DIRECT "}
	for _, typ := range skipped {
		if probeableType(typ) {
			t.Errorf("probeableType(%q) = true, want false", typ)
		}
	}
	for _, typ := range []string{"vless", "trojan", "shadowsocks", "hysteria2", "tuic", "wireguard", "ssh"} {
		if !probeableType(typ) {
			t.Errorf("probeableType(%q) = false, want true", typ)
		}
	}
}

func TestNormalizeResult(t *testing.T) {
	checked := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	result := normalizeResult(Result{
		NodeTag:   "  jp-01 ",
		NodeType:  "vless",
		Status:    "ONLINE",
		LatencyMS: 128,
		CheckedAt: checked,
	})
	if result.NodeTag != "jp-01" || result.Status != StatusOnline || result.LatencyMS != 128 {
		t.Fatalf("normalizeResult = %+v", result)
	}
	if !result.CheckedAt.Equal(checked) {
		t.Errorf("checked_at = %s, want %s", result.CheckedAt, checked)
	}

	offline := normalizeResult(Result{NodeTag: "jp-02", Status: StatusOffline, LatencyMS: 999})
	if offline.Status != StatusOffline || offline.LatencyMS != 0 {
		t.Errorf("offline latency must be cleared: %+v", offline)
	}
	if offline.Trigger != TriggerManual {
		t.Errorf("trigger = %q, want %q", offline.Trigger, TriggerManual)
	}

	unknown := normalizeResult(Result{NodeTag: "jp-03", Status: "weird"})
	if unknown.Status != StatusUnknown {
		t.Errorf("status = %q, want %q", unknown.Status, StatusUnknown)
	}
	if unknown.CheckedAt.IsZero() {
		t.Error("checked_at must be filled")
	}

	long := normalizeResult(Result{NodeTag: "jp-04", Status: StatusOffline, Error: string(make([]byte, 4096))})
	if len(long.Error) != 2048 {
		t.Errorf("error length = %d, want 2048", len(long.Error))
	}
}

func TestMergeResultsFillsMissingTargets(t *testing.T) {
	targets := []Target{
		{Tag: "a", Type: "vless"},
		{Tag: "b", Type: "trojan"},
		{Tag: "c", Type: "shadowsocks"},
	}
	results := mergeResults(targets, []Result{{NodeTag: "b", Status: StatusOnline, LatencyMS: 42}}, TriggerScheduler, nil)
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	byTag := make(map[string]Result, 3)
	for _, result := range results {
		byTag[result.NodeTag] = result
	}
	if byTag["a"].Status != StatusUnknown || byTag["a"].Error == "" {
		t.Errorf("missing result = %+v, want unknown with error", byTag["a"])
	}
	if byTag["b"].Status != StatusOnline || byTag["b"].NodeType != "trojan" {
		t.Errorf("result b = %+v", byTag["b"])
	}
	if byTag["c"].Trigger != TriggerScheduler {
		t.Errorf("trigger = %q, want %q", byTag["c"].Trigger, TriggerScheduler)
	}
}

func TestStatusFromDelay(t *testing.T) {
	if got := statusFromDelay(120); got != StatusOnline {
		t.Errorf("statusFromDelay(120) = %q, want online", got)
	}
	if got := statusFromDelay(0); got != StatusOffline {
		t.Errorf("statusFromDelay(0) = %q, want offline", got)
	}
}

func TestProbeServiceClashDelay(t *testing.T) {
	var mu sync.Mutex
	gotPaths := make(map[string]string)
	gotAuth := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths[r.URL.Path] = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		if r.URL.Path == "/proxies/broken/delay" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"timeout"}`))
			return
		}
		_, _ = w.Write([]byte(`{"delay":88}`))
	}))
	defer server.Close()

	opts := settings.NodeProbeOptions{TestURL: "https://www.gstatic.com/generate_204", Timeout: "3s"}
	results, err := probeClash(context.Background(), server.URL, "secret", []Target{
		{Tag: "ok", Type: "vless"},
		{Tag: "broken", Type: "trojan"},
	}, opts)
	if err != nil {
		t.Fatalf("probeClash: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[0].Status != StatusOnline || results[0].LatencyMS != 88 {
		t.Errorf("first result = %+v", results[0])
	}
	if results[1].Status != StatusOffline || results[1].Error == "" {
		t.Errorf("second result = %+v", results[1])
	}
	if gotAuth != "Bearer secret" {
		t.Errorf("authorization = %q", gotAuth)
	}
	query, ok := gotPaths["/proxies/ok/delay"]
	if !ok {
		t.Fatalf("requests = %v, want /proxies/ok/delay", gotPaths)
	}
	if query == "" {
		t.Error("expected url/timeout query parameters")
	}
	if _, ok := gotPaths["/proxies/broken/delay"]; !ok {
		t.Errorf("requests = %v, want /proxies/broken/delay", gotPaths)
	}
}

func TestClashCycleBudgetScalesWithWaves(t *testing.T) {
	// 单批预算必须按波次放大：一批慢节点各占满 timeout 时，
	// 后续波次拿不到预算就会被误判成离线。
	if got := clashCycleBudget(1, 10*time.Second); got != 10*time.Second {
		t.Errorf("single wave budget = %s, want 10s", got)
	}
	if got := clashCycleBudget(clashProbeConcurrency, 10*time.Second); got != 10*time.Second {
		t.Errorf("one full batch budget = %s, want 10s", got)
	}
	if got := clashCycleBudget(clashProbeConcurrency+1, 10*time.Second); got != 20*time.Second {
		t.Errorf("two waves budget = %s, want 20s", got)
	}
	if got := clashCycleBudget(10000, time.Minute); got != maxClashCycleBudget {
		t.Errorf("budget must be capped, got %s", got)
	}
}

func TestProbeClashProbesAllNodesWithinBudget(t *testing.T) {
	// 40 个节点、单次 25ms、并发 16 → 需要 3 波。
	// 整轮预算若只有 1 波（旧的 timeout 语义），后面的节点会被误判离线。
	const nodeCount = 40
	var (
		mu             sync.Mutex
		inFlight, peak int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight > peak {
			peak = inFlight
		}
		mu.Unlock()
		time.Sleep(25 * time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
		_, _ = w.Write([]byte(`{"delay":42}`))
	}))
	defer server.Close()

	targets := make([]Target, 0, nodeCount)
	for i := 0; i < nodeCount; i++ {
		targets = append(targets, Target{Tag: fmt.Sprintf("node-%02d", i), Type: "vless"})
	}
	opts := settings.NodeProbeOptions{TestURL: "https://www.gstatic.com/generate_204", Timeout: "60ms"}
	results, err := probeClash(context.Background(), server.URL, "", targets, opts)
	if err != nil {
		t.Fatalf("probeClash: %v", err)
	}
	if len(results) != nodeCount {
		t.Fatalf("results = %d, want %d", len(results), nodeCount)
	}
	for _, result := range results {
		if result.Status != StatusOnline {
			t.Fatalf("node %s = %+v, want online", result.NodeTag, result)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if peak > clashProbeConcurrency {
		t.Errorf("peak concurrency = %d, want <= %d", peak, clashProbeConcurrency)
	}
}

// TestPruneTagsForSkipsSingleNodeTriggers 覆盖回归：单节点探测不能拿
// 自己那一个 tag 当白名单，否则会删掉所有其他节点的状态行。
func TestPruneTagsForSkipsSingleNodeTriggers(t *testing.T) {
	all := []Target{{Tag: "jp-01", Type: "vless"}, {Tag: "us-01", Type: "trojan"}}
	if got := pruneTagsFor(all, nil); len(got) != 2 {
		t.Fatalf("full cycle tags = %v, want both nodes", got)
	}
	if got := pruneTagsFor([]Target{{Tag: "jp-01", Type: "vless"}}, []Target{{Tag: "jp-01", Type: "vless"}}); got != nil {
		t.Fatalf("single node cycle tags = %v, want nil (no status prune)", got)
	}
}

func TestServiceRoundBudgetCoversBroadcastSlack(t *testing.T) {
	// 整轮预算必须大于单节点超时：否则「测试已结束、最后一帧广播还在路上」
	// 会被判成整轮未完成并把已在线的节点写成离线。
	if got := serviceRoundBudget(10 * time.Second); got <= 10*time.Second {
		t.Errorf("serviceRoundBudget(10s) = %s, want > 10s", got)
	}
	if got := serviceRoundBudget(0); got <= 10*time.Second {
		t.Errorf("serviceRoundBudget(0) = %s, want > 10s", got)
	}
	// 小超时也要留出广播余量，否则 1s 超时会退化成 1s 整轮。
	if got := serviceRoundBudget(time.Second); got <= time.Second {
		t.Errorf("serviceRoundBudget(1s) = %s, want > 1s", got)
	}
}

// TestFillUnstartedMarksUnknownNotOffline 确认没轮到探测的节点记为 unknown
// 而不是 offline：节点多时前者是「本轮没测到」，后者会每轮触发假离线告警。
func TestFillUnstartedMarksUnknownNotOffline(t *testing.T) {
	targets := []Target{{Tag: "a", Type: "vless"}, {Tag: "b", Type: "trojan"}, {Tag: "c", Type: "hysteria2"}}
	// 模拟整轮预算耗尽：只有前两个下标拿到了结果。
	results := []Result{
		{NodeTag: "a", Status: StatusOnline, LatencyMS: 42},
		{NodeTag: "b", Status: StatusOffline, Error: "timeout"},
		{},
	}
	if filled := fillUnstarted(targets, results); filled != 1 {
		t.Fatalf("fillUnstarted = %d, want 1", filled)
	}
	last := results[2]
	if last.NodeTag != "c" || last.NodeType != "hysteria2" {
		t.Errorf("undispatched node = %+v, want tag/type filled from targets", last)
	}
	if last.Status != StatusUnknown {
		t.Errorf("undispatched node = %q, want %q", last.Status, StatusUnknown)
	}
	if !strings.Contains(last.Error, "not started before cycle deadline") {
		t.Errorf("undispatched node error = %q", last.Error)
	}
	// 已完成的两条不能被覆盖。
	if results[0].Status != StatusOnline || results[1].Status != StatusOffline {
		t.Errorf("completed results were overwritten: %+v %+v", results[0], results[1])
	}
}

// TestProbeClashDetectsUnusableAPI 确认「API 有响应但一个节点都测不出延迟」
// 会被当成后端故障：地址指向别的内核或 tag 全部失配时，
// 所有 offline 都是 API 语义错误，不能当成探测成功的整轮结果。
func TestProbeClashDetectsUnusableAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"proxy not found"}`))
	}))
	defer server.Close()

	targets := []Target{{Tag: "jp-01", Type: "vless"}, {Tag: "us-01", Type: "trojan"}}
	opts := settings.NodeProbeOptions{TestURL: "https://www.gstatic.com/generate_204", Timeout: "1s"}
	results, err := probeClash(context.Background(), server.URL, "", targets, opts)
	if err == nil {
		t.Fatal("expected an error when clash reports no delay for any node")
	}
	if !strings.Contains(err.Error(), "no delay") {
		t.Errorf("error = %v, want a 'no delay' diagnosis", err)
	}
	if len(results) != len(targets) {
		t.Errorf("results = %d, want %d", len(results), len(targets))
	}
}

// TestStopCancelsInFlightTrigger 覆盖关闭期的取消：手动触发的 ctx 来自 HTTP
// 请求，Engine 的 cancel 触达不到，不主动取消会把关闭拖到整轮探测超时。
func TestStopCancelsInFlightTrigger(t *testing.T) {
	entered := make(chan struct{})
	prober := func(ctx context.Context, _ []Target, _ settings.NodeProbeOptions) ([]Result, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	st := newTestStore(t)
	st.Options = option.Options{Outbounds: []option.Outbound{{Type: "vless", Tag: "a"}}}
	m := NewManagerWithProber(settings.New(filepath.Join(t.TempDir(), "settings.json")), st, newUnreachableDB(t), prober)

	triggerDone := make(chan struct{})
	go func() {
		defer close(triggerDone)
		_, _ = m.Trigger(context.Background(), TriggerManual)
	}()
	<-entered

	stopped := make(chan struct{})
	go func() {
		m.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop did not cancel the in-flight cycle")
	}
	<-triggerDone
}

func TestPostWebhook(t *testing.T) {
	type received struct {
		payload map[string]any
		eventID string
		secret  string
	}
	got := make(chan received, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		got <- received{payload: payload, eventID: r.Header.Get("X-Sing-Controller-Event-ID"), secret: r.Header.Get("X-Secret")}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	delivery := models.WebhookDelivery{
		EventID:   "event-1",
		EventType: "node.status_changed",
		URL:       server.URL,
		Secret:    "shh",
		Payload:   []byte(`{"event":"node.status_changed"}`),
	}
	code, err := postWebhook(context.Background(), delivery)
	if err != nil {
		t.Fatalf("postWebhook: %v", err)
	}
	if code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", code)
	}
	select {
	case item := <-got:
		if item.eventID != "event-1" || item.secret != "shh" {
			t.Errorf("headers = %+v", item)
		}
		if item.payload["event"] != "node.status_changed" {
			t.Errorf("payload = %+v", item.payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("webhook was not received")
	}

	delivery.URL = "http://127.0.0.1:1/hook"
	if _, err := postWebhook(context.Background(), delivery); err == nil {
		t.Error("expected transport error")
	}
}

// TestManagerUnavailableWithoutDatabase 确认未配置数据库时探测整体降级而不 panic。
func TestManagerUnavailableWithoutDatabase(t *testing.T) {
	st := newTestStore(t)
	st.Options = option.Options{Outbounds: []option.Outbound{{Type: "vless", Tag: "jp-01"}}}
	m := NewManager(settings.New(filepath.Join(t.TempDir(), "settings.json")), st, nil)
	if m.Available() {
		t.Fatal("Available() = true without database")
	}
	if err := m.Start(context.Background()); err != ErrUnavailable {
		t.Fatalf("Start = %v, want %v", err, ErrUnavailable)
	}
	if _, err := m.Trigger(context.Background(), TriggerManual); err != ErrUnavailable {
		t.Fatalf("Trigger = %v, want %v", err, ErrUnavailable)
	}
	if _, err := m.ListStatuses(context.Background()); err != ErrUnavailable {
		t.Fatalf("ListStatuses = %v, want %v", err, ErrUnavailable)
	}
	if _, err := m.History(context.Background(), "jp-01", 10); err != ErrUnavailable {
		t.Fatalf("History = %v, want %v", err, ErrUnavailable)
	}
	m.Stop()
}

// TestManagerStopWithoutStart 保证重复 Stop 不会阻塞或 panic。
func TestManagerStopWithoutStart(t *testing.T) {
	var m *Manager
	m.Stop()
	NewManager(nil, nil, nil).Stop()
}

func TestFindTargetMissing(t *testing.T) {
	st := newTestStore(t)
	m := NewManager(nil, st, nil)
	if _, ok := m.findTarget("nope"); ok {
		t.Fatal("findTarget found unknown tag")
	}
}

func TestSortStatusViews(t *testing.T) {
	items := []StatusView{
		{NodeTag: "c"},
		{NodeTag: "a"},
		{NodeTag: "b"},
	}
	sortStatusViews(items)
	if items[0].NodeTag != "a" || items[2].NodeTag != "c" {
		t.Fatalf("sorted = %+v", items)
	}
}

func TestNextStatusRowFirstInsert(t *testing.T) {
	checked := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	row, previous, changed := nextStatusRow(nil, Result{
		NodeTag:   "jp-01",
		NodeType:  "vless",
		Status:    StatusOnline,
		LatencyMS: 100,
		CheckedAt: checked,
	})
	if !changed {
		t.Error("first insert must be treated as a status change")
	}
	if previous.Status != StatusUnknown {
		t.Errorf("previous status = %q, want unknown", previous.Status)
	}
	if row.Status != StatusOnline || row.ConsecutiveFailures != 0 {
		t.Errorf("row = %+v", row)
	}
	if row.LastSuccessAt == nil || !row.LastSuccessAt.Equal(checked) {
		t.Errorf("last_success_at = %v, want %s", row.LastSuccessAt, checked)
	}
	if !row.LastCheckedAt.Equal(checked) {
		t.Errorf("last_checked_at = %v", row.LastCheckedAt)
	}
}

func TestNextStatusRowOnlyChangesOnStatusFlip(t *testing.T) {
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	existing := &models.NodeProbeStatus{
		NodeTag:             "jp-01",
		NodeType:            "vless",
		Status:              StatusOnline,
		LatencyMS:           100,
		LastSuccessAt:       &base,
		LastCheckedAt:       base,
		ConsecutiveFailures: 0,
	}

	same, previous, changed := nextStatusRow(existing, Result{
		NodeTag:   "jp-01",
		NodeType:  "vless",
		Status:    StatusOnline,
		LatencyMS: 130,
		CheckedAt: base.Add(120 * time.Second),
	})
	if changed {
		t.Error("latency-only update must not count as a status change")
	}
	if same.LatencyMS != 130 {
		t.Errorf("latency = %d, want 130", same.LatencyMS)
	}
	if same.LastSuccessAt == nil || !same.LastSuccessAt.Equal(base.Add(120*time.Second)) {
		t.Errorf("last_success_at = %v, want %s", same.LastSuccessAt, base.Add(120*time.Second))
	}
	if previous.Status != StatusOnline {
		t.Errorf("previous = %+v", previous)
	}

	flipTime := base.Add(240 * time.Second)
	flipped, previous, changed := nextStatusRow(existing, Result{
		NodeTag:   "jp-01",
		NodeType:  "vless",
		Status:    StatusOffline,
		Error:     "connection refused",
		CheckedAt: flipTime,
	})
	if !changed {
		t.Error("offline transition must be reported as a change")
	}
	if previous.Status != StatusOnline {
		t.Errorf("previous = %+v, want online", previous)
	}
	if flipped.ConsecutiveFailures != 1 {
		t.Errorf("consecutive_failures = %d, want 1", flipped.ConsecutiveFailures)
	}
	if flipped.LastError != "connection refused" {
		t.Errorf("last_error = %q", flipped.LastError)
	}
	if flipped.LastSuccessAt == nil || !flipped.LastSuccessAt.Equal(base) {
		t.Errorf("last_success_at must be preserved on failure: %v", flipped.LastSuccessAt)
	}
	again, _, changed := nextStatusRow(&flipped, Result{
		NodeTag:   "jp-01",
		NodeType:  "vless",
		Status:    StatusOffline,
		CheckedAt: base.Add(360 * time.Second),
	})
	if changed {
		t.Error("repeated offline must not report a change")
	}
	if again.ConsecutiveFailures != 2 {
		t.Errorf("consecutive_failures = %d, want 2", again.ConsecutiveFailures)
	}
}

func TestWebhookPayloadShape(t *testing.T) {
	content, err := json.Marshal(webhookPayload{
		Event:      "node.status_changed",
		Source:     "sing-controller",
		OccurredAt: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		Node:       webhookState{NodeTag: "jp-01", NodeType: "vless"},
		Previous:   webhookState{NodeTag: "jp-01", NodeType: "vless", Status: StatusOnline},
		Current:    webhookState{NodeTag: "jp-01", NodeType: "vless", Status: StatusOffline, Error: "timeout"},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["event"] != "node.status_changed" || decoded["source"] != "sing-controller" {
		t.Errorf("payload = %s", content)
	}
	current, ok := decoded["current"].(map[string]any)
	if !ok || current["status"] != StatusOffline || current["error"] != "timeout" {
		t.Errorf("current = %v", decoded["current"])
	}
}

func TestWebhookBatchPayloadShape(t *testing.T) {
	transitions := []webhookPayload{
		newWebhookPayload(Result{
			NodeTag:   "jp-01",
			NodeType:  "vless",
			Status:    StatusOffline,
			CheckedAt: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		}, webhookState{NodeTag: "jp-01", NodeType: "vless", Status: StatusOnline}),
		newWebhookPayload(Result{
			NodeTag:   "jp-02",
			NodeType:  "trojan",
			Status:    StatusOnline,
			CheckedAt: time.Date(2026, 3, 1, 10, 0, 1, 0, time.UTC),
		}, webhookState{NodeTag: "jp-02", NodeType: "trojan", Status: StatusOffline}),
	}
	body, err := json.Marshal(webhookBatchPayload{
		Event:      webhookEventStatusChanged,
		Source:     "sing-controller",
		OccurredAt: transitions[1].OccurredAt,
		Events:     transitions,
	})
	if err != nil {
		t.Fatalf("marshal batch: %v", err)
	}
	var decoded struct {
		Event  string           `json:"event"`
		Events []webhookPayload `json:"events"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal batch: %v", err)
	}
	if decoded.Event != webhookEventStatusChanged || len(decoded.Events) != 2 {
		t.Fatalf("batch payload = %s", body)
	}
	if decoded.Events[0].Node.NodeTag != "jp-01" || decoded.Events[1].Node.NodeTag != "jp-02" {
		t.Fatalf("batch nodes = %#v", decoded.Events)
	}
}

func TestConcurrentTriggerSerialization(t *testing.T) {
	// 并发触发必须被串行化，避免探测结果交叉写入数据库。
	m := NewManager(nil, newTestStore(t), nil)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.acquire(ctx); err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			m.release()
		}()
	}
	wg.Wait()
	select {
	case m.gate <- struct{}{}:
		m.release()
	default:
		t.Fatal("probe gate was not released")
	}
}

// TestProbeGateRespectsContext 确认等锁时能被 ctx 取消，
// 否则 Engine 关闭会永久阻塞在调度协程上。
func TestProbeGateRespectsContext(t *testing.T) {
	m := NewManager(nil, newTestStore(t), nil)
	if err := m.acquire(context.Background()); err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.acquire(ctx) }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected context error while gate is held")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("acquire did not return after context cancellation")
	}
	m.release()
}

func TestTriggerRefusedAfterStop(t *testing.T) {
	m := NewManager(settings.New(filepath.Join(t.TempDir(), "settings.json")), newTestStore(t), &gorm.DB{Config: &gorm.Config{}})
	m.Stop()
	if _, err := m.Trigger(context.Background(), TriggerManual); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Trigger after Stop = %v, want %v", err, ErrUnavailable)
	}
}

// TestStopWaitsForInFlightTrigger 覆盖 P1-3：手动触发跑在 HTTP 请求协程上，
// 不在 Manager 的 wg 中。若 Stop 不等待它，Engine 关闭数据库时
// 正在写库的事务会撞上 "sql: database is closed"。
func TestStopWaitsForInFlightTrigger(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	prober := func(context.Context, []Target, settings.NodeProbeOptions) ([]Result, error) {
		close(entered)
		<-release
		return []Result{{NodeTag: "a", Status: StatusOnline, CheckedAt: time.Now()}}, nil
	}
	st := newTestStore(t)
	st.Options = option.Options{Outbounds: []option.Outbound{{Type: "vless", Tag: "a"}}}
	m := NewManagerWithProber(settings.New(filepath.Join(t.TempDir(), "settings.json")), st, newUnreachableDB(t), prober)

	triggerDone := make(chan struct{})
	go func() {
		defer close(triggerDone)
		// 落库必然失败（连接不可达），这里只关心 Stop 的等待语义。
		_, _ = m.Trigger(context.Background(), TriggerManual)
	}()
	<-entered

	stopped := make(chan struct{})
	go func() {
		m.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
		t.Fatal("Stop returned while a probe cycle was still running")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return after the in-flight cycle finished")
	}
	<-triggerDone
}

// TestInvalidateDisablesProbes 覆盖 P1-4：数据库关闭后 Manager
// 必须报告不可用，否则 API 会一直对着已关闭的连接池返回 500。
func TestInvalidateDisablesProbes(t *testing.T) {
	m := NewManager(settings.New(filepath.Join(t.TempDir(), "settings.json")), newTestStore(t), &gorm.DB{Config: &gorm.Config{}})
	if !m.Available() {
		t.Fatal("manager with a database must be available")
	}
	m.Invalidate()
	if m.Available() {
		t.Fatal("manager must be unavailable after Invalidate")
	}
	if _, err := m.ListStatuses(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListStatuses after Invalidate = %v, want %v", err, ErrUnavailable)
	}
	if _, err := m.Trigger(context.Background(), TriggerManual); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Trigger after Invalidate = %v, want %v", err, ErrUnavailable)
	}
}

func TestTriggerOneUnknownNode(t *testing.T) {
	m := NewManager(settings.New(filepath.Join(t.TempDir(), "settings.json")), newTestStore(t), &gorm.DB{Config: &gorm.Config{}})
	if _, err := m.TriggerOne(context.Background(), "ghost", TriggerManual); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("TriggerOne unknown = %v, want %v", err, ErrNodeNotFound)
	}
}
