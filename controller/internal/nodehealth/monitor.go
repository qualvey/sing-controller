package nodehealth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
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
	Enabled  bool              `json:"enabled"`
	Statuses map[string]Status `json:"statuses"`
}

type Event struct {
	Event            string    `json:"event"`
	Node             Status    `json:"node"`
	ConsecutiveFails int       `json:"consecutive_failures"`
	CheckedAt        time.Time `json:"checked_at"`
}

type Runner struct {
	cfg   *settings.Manager
	store *store.Store

	mu       sync.RWMutex
	statuses map[string]Status
	nativeMu sync.Mutex
	http     *http.Client
}

func New(cfg *settings.Manager, st *store.Store) *Runner {
	return &Runner{
		cfg:      cfg,
		store:    st,
		statuses: make(map[string]Status),
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
	r.mu.RUnlock()
	return Snapshot{Enabled: values.NodeHealth.Enabled, Statuses: statuses}
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
		case now := <-ticker.C:
			options := r.Config()
			if !options.Enabled {
				continue
			}
			interval, err := time.ParseDuration(options.Interval)
			if err != nil {
				interval = 5 * time.Minute
			}
			if !lastRun.IsZero() && now.Sub(lastRun) < interval {
				continue
			}
			lastRun = now
			r.checkAll(ctx, options)
		}
	}
}

func (r *Runner) checkAll(ctx context.Context, options settings.NodeHealthOptions) {
	proxies := r.proxyTargets()
	active := make(map[string]struct{}, len(proxies))
	for _, proxy := range proxies {
		active[proxy.Tag] = struct{}{}
	}
	r.mu.Lock()
	for tag := range r.statuses {
		if _, ok := active[tag]; !ok {
			delete(r.statuses, tag)
		}
	}
	r.mu.Unlock()
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
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-semaphore }()
			latency, err := r.probe(ctx, options, proxy.Tag)
			r.record(options, proxy, latency, err)
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

func (r *Runner) record(options settings.NodeHealthOptions, target proxyTarget, latency int64, probeErr error) {
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
	shouldNotify := (status.State == "unavailable" && oldState != "unavailable") ||
		(status.State == "available" && oldState == "unavailable")
	r.mu.Unlock()

	if probeErr != nil {
		logger.Debug("[NodeHealth] probe %s failed (%d/%d): %v", target.Tag, status.ConsecutiveFails, options.FailureThreshold, probeErr)
	}
	if !shouldNotify || options.WebhookURL == "" {
		return
	}
	eventName := "node.available"
	if status.State == "unavailable" {
		eventName = "node.unavailable"
	}
	go r.sendWebhook(options, Event{
		Event:            eventName,
		Node:             status,
		ConsecutiveFails: status.ConsecutiveFails,
		CheckedAt:        now,
	})
}

func (r *Runner) sendWebhook(options settings.NodeHealthOptions, event Event) {
	body, useBearerToken, err := buildWebhookPayload(options, event)
	if err != nil {
		return
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
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
			err = fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
		}
		if attempt == 0 {
			time.Sleep(500 * time.Millisecond)
		}
		logger.Warn("[NodeHealth] webhook delivery failed (attempt %d/2): %v", attempt+1, err)
	}
}
