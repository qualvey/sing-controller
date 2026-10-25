package nodehealth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing-box/daemon"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

const failureRetryDelay = time.Minute
const webhookBatchWindow = 2 * time.Second

type Status struct {
	Tag              string    `json:"tag"`
	Type             string    `json:"type"`
	State            string    `json:"state"`
	LatencyMS        int64     `json:"latency_ms,omitempty"`
	ConsecutiveFails int       `json:"consecutive_failures"`
	CheckedAt        time.Time `json:"checked_at,omitempty"`
	Error            string    `json:"error,omitempty"`
}

type Snapshot struct {
	Enabled       bool              `json:"enabled"`
	Scanning      bool              `json:"scanning"`
	ScanQueued    bool              `json:"scan_queued"`
	LastScanStart time.Time         `json:"last_scan_started,omitempty"`
	LastScanEnd   time.Time         `json:"last_scan_completed,omitempty"`
	Statuses      map[string]Status `json:"statuses"`
}

type Event struct {
	Event            string    `json:"event"`
	Node             Status    `json:"node"`
	ConsecutiveFails int       `json:"consecutive_failures"`
	CheckedAt        time.Time `json:"checked_at"`
}

type webhookBatch struct {
	options settings.NodeHealthOptions
	events  []Event
	timer   *time.Timer
}

type retryTask struct {
	target  proxyTarget
	retryAt time.Time
}

type Runner struct {
	cfg   *settings.Manager
	store *store.Store

	mu        sync.RWMutex
	statuses  map[string]Status
	nativeMu  sync.Mutex
	scanCh    chan struct{}
	scanning  bool
	queued    bool
	lastStart time.Time
	lastEnd   time.Time
	retries   map[string]retryTask
	webhookMu sync.Mutex
	webhook   *webhookBatch
	http      *http.Client
}

func New(cfg *settings.Manager, st *store.Store) *Runner {
	return &Runner{
		cfg:      cfg,
		store:    st,
		statuses: make(map[string]Status),
		scanCh:   make(chan struct{}, 1),
		retries:  make(map[string]retryTask),
		http:     &http.Client{Timeout: 65 * time.Second},
	}
}

func (r *Runner) Snapshot() Snapshot {
	values := r.cfg.Values()
	r.mu.RLock()
	statuses := make(map[string]Status, len(r.statuses))
	for tag, status := range r.statuses {
		statuses[tag] = status
	}
	scanning := r.scanning
	queued := r.queued
	lastStart := r.lastStart
	lastEnd := r.lastEnd
	r.mu.RUnlock()
	return Snapshot{
		Enabled:       values.NodeHealth.Enabled,
		Scanning:      scanning,
		ScanQueued:    queued,
		LastScanStart: lastStart,
		LastScanEnd:   lastEnd,
		Statuses:      statuses,
	}
}

// TriggerScan queues one immediate scan. It returns false while a scan is
// already queued or running.
func (r *Runner) TriggerScan() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.scanning || r.queued {
		return false
	}
	r.queued = true
	select {
	case r.scanCh <- struct{}{}:
		return true
	default:
		r.queued = false
		return false
	}
}

func (r *Runner) Config() settings.NodeHealthOptions {
	options := r.cfg.Values().NodeHealth
	options.Normalize()
	return options
}

func (r *Runner) UpdateConfig(options settings.NodeHealthOptions) error {
	options.FillDefaults()
	if err := options.Validate(); err != nil {
		return err
	}
	return r.cfg.Update(func(values *settings.Settings) error {
		values.NodeHealth = options
		return nil
	})
}

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastRun time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.scanCh:
			r.runScan(ctx, r.Config(), "manual")
			lastRun = time.Now()
		case now := <-ticker.C:
			options := r.Config()
			if !options.Enabled {
				r.runDueRetries(ctx, options, now)
				continue
			}
			select {
			case <-r.scanCh:
				r.runScan(ctx, options, "manual")
				lastRun = time.Now()
				continue
			default:
			}
			interval, err := time.ParseDuration(options.Interval)
			if err != nil {
				interval = 5 * time.Minute
			}
			if lastRun.IsZero() || now.Sub(lastRun) >= interval {
				r.runScan(ctx, options, "scheduled")
				lastRun = time.Now()
				continue
			}
			r.runDueRetries(ctx, options, now)
		}
	}
}

func (r *Runner) runScan(ctx context.Context, options settings.NodeHealthOptions, kind string) {
	r.mu.Lock()
	wasQueued := r.queued
	r.queued = false
	r.scanning = true
	r.lastStart = time.Now().UTC()
	r.mu.Unlock()
	if wasQueued {
		select {
		case <-r.scanCh:
		default:
		}
	}

	r.checkAll(ctx, options, kind)

	r.mu.Lock()
	r.scanning = false
	r.lastEnd = time.Now().UTC()
	r.mu.Unlock()
}

func (r *Runner) checkAll(ctx context.Context, options settings.NodeHealthOptions, kind string) {
	configured := r.proxyTargets()
	active := make(map[string]struct{}, len(configured))
	for _, proxy := range configured {
		active[proxy.Tag] = struct{}{}
	}
	r.mu.Lock()
	for tag := range r.statuses {
		if _, ok := active[tag]; !ok {
			delete(r.statuses, tag)
			delete(r.retries, tag)
		}
	}
	for tag := range r.retries {
		if _, ok := active[tag]; !ok {
			delete(r.retries, tag)
		}
	}
	r.mu.Unlock()
	r.checkTargets(ctx, options, r.targetsForScan(configured, kind, time.Now()), kind)
}

func (r *Runner) targetsForScan(targets []proxyTarget, kind string, now time.Time) []proxyTarget {
	if kind != "scheduled" {
		return targets
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	filtered := make([]proxyTarget, 0, len(targets))
	for _, target := range targets {
		if retry, pending := r.retries[target.Tag]; pending && now.Before(retry.retryAt) {
			continue
		}
		filtered = append(filtered, target)
	}
	return filtered
}

func (r *Runner) runDueRetries(ctx context.Context, options settings.NodeHealthOptions, now time.Time) {
	targets := r.takeDueRetries(now)
	if len(targets) > 0 {
		r.checkTargets(ctx, options, targets, "retry")
	}
}

func (r *Runner) takeDueRetries(now time.Time) []proxyTarget {
	activeTargets := r.proxyTargets()
	active := make(map[string]proxyTarget, len(activeTargets))
	for _, target := range activeTargets {
		active[target.Tag] = target
	}
	r.mu.Lock()
	targets := make([]proxyTarget, 0, len(r.retries))
	for tag, task := range r.retries {
		target, exists := active[tag]
		if !exists {
			delete(r.retries, tag)
			delete(r.statuses, tag)
			continue
		}
		if !now.Before(task.retryAt) {
			targets = append(targets, target)
			delete(r.retries, tag)
		}
	}
	r.mu.Unlock()
	return targets
}

func (r *Runner) checkTargets(ctx context.Context, options settings.NodeHealthOptions, proxies []proxyTarget, kind string) {
	startedAt := time.Now()
	logger.Info("[NodeHealth] %s probe cycle started (nodes=%d, interval=%s)", kind, len(proxies), options.Interval)
	var succeeded atomic.Int64
	var failed atomic.Int64
	defer func() {
		logger.Info("[NodeHealth] %s probe cycle completed (nodes=%d, succeeded=%d, failed=%d, duration=%s)",
			kind, len(proxies), succeeded.Load(), failed.Load(), time.Since(startedAt).Round(time.Millisecond))
	}()
	if len(proxies) == 0 {
		return
	}
	semaphore := make(chan struct{}, options.Concurrency)
	var wg sync.WaitGroup
	for _, proxy := range proxies {
		proxy := proxy
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()
			latency, err := r.probe(ctx, options, proxy.Tag)
			if err == nil {
				succeeded.Add(1)
			} else {
				failed.Add(1)
			}
			r.record(options, proxy, latency, err, kind)
		}()
	}
	wg.Wait()
}

type proxyTarget struct {
	Tag  string
	Type string
}

func (r *Runner) proxyTargets() []proxyTarget {
	r.store.RLock()
	defer r.store.RUnlock()

	blocked := map[string]bool{
		"direct": true, "block": true, "dns": true, "selector": true, "urltest": true,
	}
	result := make([]proxyTarget, 0, len(r.store.Options.Outbounds)+len(r.store.Options.Endpoints))
	for _, outbound := range r.store.Options.Outbounds {
		if outbound.Tag == "" || blocked[strings.ToLower(outbound.Type)] {
			continue
		}
		result = append(result, proxyTarget{Tag: outbound.Tag, Type: outbound.Type})
	}
	for _, endpoint := range r.store.Options.Endpoints {
		if endpoint.Tag == "" || blocked[strings.ToLower(endpoint.Type)] {
			continue
		}
		result = append(result, proxyTarget{Tag: endpoint.Tag, Type: endpoint.Type})
	}
	return result
}

func (r *Runner) probe(ctx context.Context, options settings.NodeHealthOptions, tag string) (int64, error) {
	values := r.cfg.Values()
	var clashErr error
	if endpoint := values.API.Clash; endpoint != nil && strings.TrimSpace(endpoint.Address) != "" {
		latency, err := r.probeClash(ctx, options, endpoint.Address, endpoint.Secret, tag)
		if err == nil {
			return latency, nil
		}
		clashErr = err
		if !isClashAPIFallbackError(err) {
			return 0, err
		}
	}
	if endpoint := values.API.Service; endpoint != nil && strings.TrimSpace(endpoint.Address) != "" {
		return r.probeService(ctx, options, endpoint.Address, endpoint.Secret, tag)
	}
	if clashErr != nil {
		return 0, clashErr
	}
	return 0, errors.New("未配置 Clash API 或 sing-box service API，无法探测节点")
}

type clashStatusError struct {
	statusCode int
}

func (e clashStatusError) Error() string {
	return fmt.Sprintf("Clash delay endpoint returned HTTP %d", e.statusCode)
}

func isClashAPIFallbackError(err error) bool {
	var statusErr clashStatusError
	if errors.As(err, &statusErr) {
		return statusErr.statusCode == http.StatusNotFound || statusErr.statusCode == http.StatusUnauthorized || statusErr.statusCode == http.StatusForbidden
	}
	return true
}

func (r *Runner) probeClash(parent context.Context, options settings.NodeHealthOptions, address, secret, tag string) (int64, error) {
	base := strings.TrimRight(address, "/")
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	endpoint, err := url.Parse(base)
	if err != nil || endpoint.Host == "" {
		return 0, errors.New("Clash API 地址无效")
	}
	pathPrefix := strings.TrimRight(endpoint.Path, "/") + "/proxies/"
	endpoint.Path = pathPrefix + tag + "/delay"
	endpoint.RawPath = pathPrefix + url.PathEscape(tag) + "/delay"
	query := endpoint.Query()
	query.Set("url", options.TestURL)
	query.Set("timeout", fmt.Sprint(options.TimeoutMS))
	endpoint.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(parent, time.Duration(options.TimeoutMS+1000)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, err
	}
	if secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := r.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, clashStatusError{statusCode: resp.StatusCode}
	}
	var result struct {
		Delay int64 `json:"delay"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode Clash delay response: %w", err)
	}
	if result.Delay <= 0 {
		return 0, errors.New("Clash delay endpoint returned a non-positive delay")
	}
	return result.Delay, nil
}

func (r *Runner) probeService(parent context.Context, options settings.NodeHealthOptions, address, secret, tag string) (int64, error) {
	r.nativeMu.Lock()
	defer r.nativeMu.Unlock()

	target := strings.TrimSpace(address)
	if strings.Contains(target, "://") {
		parsed, err := url.Parse(target)
		if err != nil {
			return 0, fmt.Errorf("sing-box service API 地址无效: %w", err)
		}
		target = parsed.Host
	}
	if target == "" {
		return 0, errors.New("sing-box service API 地址为空")
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	timeout := time.Duration(options.TimeoutMS+3000) * time.Millisecond
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	if secret != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+secret)
	}
	client := daemon.NewStartedServiceClient(conn)
	stream, err := client.SubscribeOutbounds(ctx, &emptypb.Empty{})
	if err != nil {
		return 0, fmt.Errorf("subscribe sing-box outbounds: %w", err)
	}
	if _, err := stream.Recv(); err != nil {
		return 0, fmt.Errorf("read sing-box outbound snapshot: %w", err)
	}
	probeStartedAt := time.Now().Unix()
	if _, err := client.URLTest(ctx, &daemon.URLTestRequest{OutboundTag: tag}); err != nil {
		return 0, fmt.Errorf("start sing-box URLTest: %w", err)
	}
	for {
		update, err := stream.Recv()
		if err != nil {
			return 0, fmt.Errorf("wait for sing-box URLTest result: %w", err)
		}
		for _, item := range update.Outbounds {
			if item.Tag != tag {
				continue
			}
			if item.UrlTestDelay > 0 && item.UrlTestTime >= probeStartedAt {
				return int64(item.UrlTestDelay), nil
			}
			if item.UrlTestDelay > 0 {
				continue
			}
			// The native service clears history on a failed URLTest and emits an
			// outbound update. A zero delay in that update means the node is down.
			return 0, errors.New("sing-box URLTest failed")
		}
	}
}

func (r *Runner) record(options settings.NodeHealthOptions, target proxyTarget, latency int64, probeErr error, kind ...string) {
	now := time.Now().UTC()
	r.mu.Lock()
	status := r.statuses[target.Tag]
	status.Tag = target.Tag
	status.Type = target.Type
	status.CheckedAt = now
	if status.State == "" {
		status.State = "unknown"
	}
	oldState := status.State
	if probeErr == nil {
		status.State = "available"
		status.LatencyMS = latency
		status.ConsecutiveFails = 0
		status.Error = ""
	} else {
		status.ConsecutiveFails++
		status.LatencyMS = 0
		status.Error = probeErr.Error()
		if status.ConsecutiveFails >= options.FailureThreshold {
			status.State = "unavailable"
		}
	}
	r.statuses[target.Tag] = status
	if probeErr != nil && status.ConsecutiveFails < options.FailureThreshold {
		r.retries[target.Tag] = retryTask{target: target, retryAt: now.Add(failureRetryDelay)}
	} else {
		delete(r.retries, target.Tag)
	}
	shouldNotify := (status.State == "unavailable" && oldState != "unavailable") ||
		(status.State == "available" && oldState == "unavailable")
	// Scheduled probes notify only after a failed retry has confirmed the state.
	// Manual probes are intentionally immediate so operators can verify recovery.
	if probeErr != nil && len(kind) > 0 && kind[0] == "scheduled" && status.ConsecutiveFails < options.FailureThreshold {
		shouldNotify = false
	}
	r.mu.Unlock()

	if probeErr != nil {
		if status.ConsecutiveFails < options.FailureThreshold {
			logger.Info("[NodeHealth] probe failed; retry scheduled (tag=%s, failures=%d/%d, retry_after=%s, error=%v)",
				target.Tag, status.ConsecutiveFails, options.FailureThreshold, failureRetryDelay, probeErr)
		} else {
			logger.Debug("[NodeHealth] probe %s failed (%d/%d): %v", target.Tag, status.ConsecutiveFails, options.FailureThreshold, probeErr)
		}
	}
	if !shouldNotify {
		return
	}
	logger.Info("[NodeHealth] node status changed (tag=%s, state=%s, consecutive_failures=%d)",
		status.Tag, status.State, status.ConsecutiveFails)
	if options.WebhookURL == "" {
		logger.Info("[NodeHealth] webhook skipped for %s: no webhook URL configured", status.Tag)
		return
	}
	eventName := "node.available"
	if status.State == "unavailable" {
		eventName = "node.unavailable"
	}
	adapter := "generic"
	signed := false
	if isFeishuWebhook(options.WebhookURL) {
		adapter = "feishu"
		signed = options.WebhookSignSecret != ""
	}
	logger.Info("[NodeHealth] dispatching webhook (event=%s, tag=%s, adapter=%s, signed=%t)",
		eventName, status.Tag, adapter, signed)
	r.queueWebhook(options, Event{
		Event:            eventName,
		Node:             status,
		ConsecutiveFails: status.ConsecutiveFails,
		CheckedAt:        now,
	})
}

func (r *Runner) queueWebhook(options settings.NodeHealthOptions, event Event) {
	r.webhookMu.Lock()
	defer r.webhookMu.Unlock()
	if r.webhook == nil {
		r.webhook = &webhookBatch{options: options, events: []Event{event}}
		r.webhook.timer = time.AfterFunc(webhookBatchWindow, r.flushWebhook)
		return
	}
	r.webhook.events = append(r.webhook.events, event)
}

func (r *Runner) flushWebhook() {
	r.webhookMu.Lock()
	batch := r.webhook
	r.webhook = nil
	r.webhookMu.Unlock()
	if batch == nil || len(batch.events) == 0 {
		return
	}
	go r.sendWebhookBatch(batch.options, batch.events)
}

func (r *Runner) sendWebhook(options settings.NodeHealthOptions, event Event) {
	r.sendWebhookBatch(options, []Event{event})
}

func (r *Runner) sendWebhookBatch(options settings.NodeHealthOptions, events []Event) {
	if len(events) == 0 {
		return
	}
	event := events[0]
	body, useBearerToken, err := buildWebhookBatchPayload(options, events)
	if err != nil {
		logger.Warn("[NodeHealth] build webhook payload failed (event=%s, tag=%s): %v", event.Event, event.Node.Tag, err)
		return
	}
	if isFeishuWebhook(options.WebhookURL) {
		var payload struct {
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal(body, &payload); err == nil && payload.Timestamp != "" {
			logger.Info("[NodeHealth] Feishu signing timestamp (tag=%s, timestamp=%s)", event.Node.Tag, payload.Timestamp)
		}
	}
	r.postWebhook(options, body, useBearerToken)
}

func buildWebhookPayload(options settings.NodeHealthOptions, event Event) ([]byte, bool, error) {
	var body []byte
	var err error
	feishu := isFeishuWebhook(options.WebhookURL)
	if feishu {
		message := feishuMessage{
			MsgType: "text",
			Content: feishuTextContent{Text: formatFeishuEvent(event)},
		}
		if options.WebhookSignSecret != "" {
			timestamp := time.Now().Unix()
			message.Timestamp = fmt.Sprint(timestamp)
			message.Sign = signFeishuWebhook(options.WebhookSignSecret, timestamp)
		}
		body, err = json.Marshal(message)
	} else {
		body, err = json.Marshal(event)
	}
	if err != nil {
		return nil, false, err
	}
	return body, !feishu, nil
}

func buildWebhookBatchPayload(options settings.NodeHealthOptions, events []Event) ([]byte, bool, error) {
	if len(events) == 1 {
		return buildWebhookPayload(options, events[0])
	}
	texts := make([]string, 0, len(events))
	for _, event := range events {
		texts = append(texts, formatFeishuEvent(event))
	}
	if isFeishuWebhook(options.WebhookURL) {
		message := feishuMessage{
			MsgType: "text",
			Content: feishuTextContent{Text: strings.Join(texts, "\n\n---\n\n")},
		}
		if options.WebhookSignSecret != "" {
			timestamp := time.Now().Unix()
			message.Timestamp = fmt.Sprint(timestamp)
			message.Sign = signFeishuWebhook(options.WebhookSignSecret, timestamp)
		}
		body, err := json.Marshal(message)
		return body, false, err
	}
	body, err := json.Marshal(map[string]any{"events": events})
	return body, true, err
}

type feishuMessage struct {
	Timestamp string            `json:"timestamp,omitempty"`
	Sign      string            `json:"sign,omitempty"`
	MsgType   string            `json:"msg_type"`
	Content   feishuTextContent `json:"content"`
}

type feishuTextContent struct {
	Text string `json:"text"`
}

func isFeishuWebhook(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && strings.Contains(strings.ToLower(u.Hostname()), "feishu")
}

func signFeishuWebhook(secret string, timestamp int64) string {
	key := []byte(fmt.Sprintf("%d\n%s", timestamp, secret))
	signer := hmac.New(sha256.New, key)
	return base64.StdEncoding.EncodeToString(signer.Sum(nil))
}

func formatFeishuEvent(event Event) string {
	state := "节点已恢复"
	if event.Event == "node.unavailable" {
		state = "节点不可用"
	}
	lines := []string{
		state,
		"节点: " + event.Node.Tag,
		"协议: " + event.Node.Type,
		"状态: " + event.Node.State,
	}
	if event.Node.LatencyMS > 0 {
		lines = append(lines, fmt.Sprintf("延迟: %d ms", event.Node.LatencyMS))
	}
	if event.ConsecutiveFails > 0 {
		lines = append(lines, fmt.Sprintf("连续失败: %d", event.ConsecutiveFails))
	}
	if event.Node.Error != "" {
		lines = append(lines, "错误: "+event.Node.Error)
	}
	lines = append(lines, "探测时间: "+event.CheckedAt.Format(time.RFC3339))
	return strings.Join(lines, "\n")
}

func (r *Runner) postWebhook(options settings.NodeHealthOptions, body []byte, useBearerToken bool) {
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest(http.MethodPost, options.WebhookURL, strings.NewReader(string(body)))
		if err != nil {
			logger.Warn("[NodeHealth] create webhook request failed: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if useBearerToken && options.WebhookToken != "" {
			req.Header.Set("Authorization", "Bearer "+options.WebhookToken)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req = req.WithContext(ctx)
		resp, err := r.http.Do(req)
		cancel()
		if err == nil {
			responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
			resp.Body.Close()
			if readErr != nil {
				err = fmt.Errorf("read webhook response: %w", readErr)
			} else {
				err = webhookResponseError(resp.StatusCode, responseBody, isFeishuWebhook(options.WebhookURL))
			}
			if err == nil {
				return
			}
		}
		if attempt == 0 {
			time.Sleep(500 * time.Millisecond)
		}
		logger.Warn("[NodeHealth] webhook delivery failed (attempt %d/2): %v", attempt+1, err)
	}
}

func webhookResponseError(statusCode int, body []byte, feishu bool) error {
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("webhook returned HTTP %d", statusCode)
	}
	if !feishu || len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode Feishu webhook response: %w", err)
	}
	if response.Code != 0 {
		return fmt.Errorf("Feishu webhook rejected message (code=%d, msg=%s)", response.Code, response.Msg)
	}
	return nil
}
