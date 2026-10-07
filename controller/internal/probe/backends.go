package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/sagernet/sing-box/daemon"
	"github.com/sagernet/sing-box/option"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (m *Manager) targets() []Target {
	if m == nil || m.store == nil {
		return nil
	}
	m.store.RLock()
	defer m.store.RUnlock()

	result := make([]Target, 0, len(m.store.Options.Outbounds)+len(m.store.Options.Endpoints))
	seen := make(map[string]struct{}, cap(result))
	appendTarget := func(tag, typ string) {
		tag = strings.TrimSpace(tag)
		if tag == "" || !probeableType(typ) {
			return
		}
		if _, ok := seen[tag]; ok {
			return
		}
		seen[tag] = struct{}{}
		result = append(result, Target{Tag: tag, Type: typ})
	}
	for _, outbound := range m.store.Options.Outbounds {
		appendTarget(outbound.Tag, outbound.Type)
	}
	for _, endpoint := range m.store.Options.Endpoints {
		appendTarget(endpoint.Tag, endpoint.Type)
	}
	return result
}

func probeableType(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "direct", "block", "dns", "selector", "urltest", "fallback", "loadbalance":
		return false
	default:
		return true
	}
}

// probeBackends 按 service → clash 的顺序尝试探测后端。
// 关键约定：后端返回「部分结果 + 错误」时必须把结果一并向上传递。
// 整轮超时是最常见的错误（个别节点慢），若把已知结果丢掉，
// mergeResults 会把整批节点写成 unknown，并触发全量状态变化 Webhook。
func (m *Manager) probeBackends(ctx context.Context, targets []Target, opts settings.NodeProbeOptions) ([]Result, error) {
	var serviceErr error
	var serviceResults []Result
	if address, secret, ok := m.serviceTarget(); ok {
		results, err := probeService(ctx, address, secret, targets, opts)
		if err == nil {
			return results, nil
		}
		serviceErr, serviceResults = err, results
	}

	if address, secret, ok := m.clashTarget(); ok {
		results, err := probeClash(ctx, address, secret, targets, opts)
		if err == nil {
			return results, nil
		}
		switch {
		case len(serviceResults) > 0:
			// clash 本轮不可用（地址不匹配 / tag 全部失效 / 整轮超时），
			// 而 gRPC 侧仍有真实测量值：宁可采用 service 的部分结果，
			// 也不要把「clash 答不上来」当成所有节点离线。
			return serviceResults, fmt.Errorf("clash probe: %v; service probe: %w", err, serviceErr)
		case len(results) > 0:
			// clash 有部分结果：优先采用，错误仅作为附加信息。
			return results, err
		case serviceErr != nil:
			return nil, fmt.Errorf("service probe: %v; clash probe: %w", serviceErr, err)
		default:
			return nil, err
		}
	}
	if serviceErr != nil {
		return serviceResults, serviceErr
	}
	return nil, errors.New("no sing-box service or clash API is configured")
}

func (m *Manager) serviceTarget() (string, string, bool) {
	if m.settings != nil {
		values := m.settings.Values()
		if values.API.Service != nil && strings.TrimSpace(values.API.Service.Address) != "" {
			return normalizeAddress(values.API.Service.Address), values.API.Service.Secret, true
		}
	}
	if m.store == nil {
		return "", "", false
	}
	m.store.RLock()
	defer m.store.RUnlock()
	for i := range m.store.Options.Services {
		service := &m.store.Options.Services[i]
		if service.Type != "api" {
			continue
		}
		serviceOptions, ok := service.Options.(*option.APIServiceOptions)
		if !ok || serviceOptions == nil {
			continue
		}
		host := ""
		if serviceOptions.Listen != nil {
			host = serviceOptions.Listen.Build(netip.IPv4Unspecified()).String()
		}
		if host == "" || host == "::" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		port := serviceOptions.ListenPort
		if port == 0 {
			port = 9900
		}
		return "http://" + net.JoinHostPort(host, strconv.Itoa(int(port))), serviceOptions.Secret, true
	}
	return "", "", false
}

func (m *Manager) clashTarget() (string, string, bool) {
	if m.settings != nil {
		values := m.settings.Values()
		if values.API.Clash != nil && strings.TrimSpace(values.API.Clash.Address) != "" {
			return normalizeAddress(values.API.Clash.Address), values.API.Clash.Secret, true
		}
	}
	if m.store == nil {
		return "", "", false
	}
	m.store.RLock()
	defer m.store.RUnlock()
	if m.store.Options.Experimental == nil || m.store.Options.Experimental.ClashAPI == nil {
		return "", "", false
	}
	clash := m.store.Options.Experimental.ClashAPI
	if strings.TrimSpace(clash.ExternalController) == "" {
		return "", "", false
	}
	return normalizeAddress(clash.ExternalController), clash.Secret, true
}

func normalizeAddress(address string) string {
	address = strings.TrimSpace(address)
	if !strings.Contains(address, "://") {
		return "http://" + address
	}
	return address
}

// urlTestSample 是 sing-box 广播里某个出站的 URLTest 历史快照。
// sing-box 用「成功写入 history / 失败删除 history」表达探测结果，
// 因此必须与探测前的基线做差分才能判断某条广播是本轮结果还是上一轮的残留。
type urlTestSample struct {
	delay int32
	at    int64
}

func sampleOf(list *daemon.OutboundList) map[string]urlTestSample {
	samples := make(map[string]urlTestSample, len(list.GetOutbounds()))
	for _, item := range list.GetOutbounds() {
		samples[item.GetTag()] = urlTestSample{delay: item.GetUrlTestDelay(), at: item.GetUrlTestTime()}
	}
	return samples
}

// serviceRoundBudget 计算 gRPC 探测的整轮时间上限。
// sing-box 的 URLTest 是 fire-and-forget：每个节点各自起协程并发测试，
// 广播按 250ms 节流重推全量列表。因此整轮耗时 ≈ 最慢的单个节点 + 广播余量，
// 不需要像 clash 那样按波次放大；但也不能只留一个节点的超时，
// 否则「测试已结束、最后一帧广播还在路上」会被误判成整轮离线。
func serviceRoundBudget(perNode time.Duration) time.Duration {
	if perNode <= 0 {
		perNode = 10 * time.Second
	}
	slack := perNode / 2
	if slack < 2*time.Second {
		slack = 2 * time.Second
	}
	return perNode + slack
}

func probeService(ctx context.Context, address, secret string, targets []Target, opts settings.NodeProbeOptions) ([]Result, error) {
	connection, err := daemon.NewRemoteClient(daemon.RemoteClientOptions{
		ServerURL: normalizeAddress(address),
		Secret:    secret,
	})
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	client := daemon.NewStartedServiceClient(connection)
	roundBudget := serviceRoundBudget(opts.TimeoutDuration())
	probeCtx, cancel := context.WithTimeout(ctx, roundBudget)
	defer cancel()

	// 必须先订阅再触发：URLTest 是一元 fire-and-forget RPC，结果只通过
	// SubscribeOutbounds 广播回来。首帧同时作为「探测前基线」，
	// 用来过滤上一轮遗留的历史值（否则会把上一轮的延迟当成本轮结果）。
	stream, err := client.SubscribeOutbounds(probeCtx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	initial, err := stream.Recv()
	if err != nil {
		return nil, err
	}
	baseline := sampleOf(initial)

	updates := make(chan *daemon.OutboundList, 4)
	streamErr := make(chan error, 1)
	go func() {
		defer close(updates)
		for {
			update, recvErr := stream.Recv()
			if recvErr != nil {
				select {
				case streamErr <- recvErr:
				default:
				}
				return
			}
			select {
			case updates <- update:
			case <-probeCtx.Done():
				return
			}
		}
	}()

	targetSet := make(map[string]Target, len(targets))
	for _, target := range targets {
		targetSet[target.Tag] = target
	}
	results := make(map[string]Result, len(targets))
	pending := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		pending[target.Tag] = struct{}{}
		if _, err := client.URLTest(probeCtx, &daemon.URLTestRequest{OutboundTag: target.Tag}); err != nil {
			results[target.Tag] = Result{
				NodeTag:   target.Tag,
				NodeType:  target.Type,
				Status:    StatusOffline,
				Error:     err.Error(),
				CheckedAt: time.Now(),
			}
			delete(pending, target.Tag)
		}
	}

	timer := time.NewTimer(roundBudget)
	defer timer.Stop()
	for len(pending) > 0 {
		select {
		case update, ok := <-updates:
			if !ok {
				updates = nil
				continue
			}
			if update == nil {
				continue
			}
			// sing-box 每次推送都会重建整个出站列表，其中未完成的节点
			// 携带的是历史值。与基线相同即视为尚未完成本轮探测。
			// 基线无历史且本轮又失败的节点无法与「仍在测试中」区分，
			// 只能等整轮超时后由 markPendingOffline 判为 offline。
			samples := sampleOf(update)
			for _, item := range update.GetOutbounds() {
				tag := item.GetTag()
				target, ok := targetSet[tag]
				if !ok {
					continue
				}
				if _, stillPending := pending[tag]; !stillPending {
					continue
				}
				if samples[tag] == baseline[tag] {
					continue
				}
				results[tag] = Result{
					NodeTag:   target.Tag,
					NodeType:  target.Type,
					Status:    statusFromDelay(int64(item.GetUrlTestDelay())),
					LatencyMS: int64(item.GetUrlTestDelay()),
					CheckedAt: time.Now(),
				}
				delete(pending, tag)
			}
		case err := <-streamErr:
			if probeCtx.Err() != nil {
				if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
					return markPendingOffline(targets, results, pending, opts.TimeoutDuration()), context.DeadlineExceeded
				}
				// controller 正在关闭：runCycle 会因 ctx 取消而整体放弃本轮，
				// 不把正常断连写成节点失败。
				return orderedResults(targets, results), probeCtx.Err()
			}
			// 流中途断开：已完成的节点照常返回，未完成的留给 mergeResults
			// 记为 unknown（宁可「未知」也不误报离线），下一轮自愈。
			return orderedResults(targets, results), err
		case <-timer.C:
			timeoutErr := fmt.Errorf("node probe timeout after %s", roundBudget)
			return markPendingOffline(targets, results, pending, roundBudget), timeoutErr
		}
	}
	return orderedResults(targets, results), nil
}

func statusFromDelay(delay int64) string {
	if delay > 0 {
		return StatusOnline
	}
	return StatusOffline
}

func orderedResults(targets []Target, results map[string]Result) []Result {
	ordered := make([]Result, 0, len(targets))
	for _, target := range targets {
		if result, ok := results[target.Tag]; ok {
			ordered = append(ordered, result)
		}
	}
	return ordered
}

func markPendingOffline(targets []Target, results map[string]Result, pending map[string]struct{}, timeout time.Duration) []Result {
	for tag := range pending {
		for _, target := range targets {
			if target.Tag != tag {
				continue
			}
			results[tag] = Result{
				NodeTag:   target.Tag,
				NodeType:  target.Type,
				Status:    StatusOffline,
				Error:     fmt.Sprintf("node probe timeout after %s", timeout),
				CheckedAt: time.Now(),
			}
			break
		}
	}
	return orderedResults(targets, results)
}

const (
	// clashProbeConcurrency 限制同时打到 clash API 的探测请求数。
	// 串行探测会让一轮耗时 = N × timeout，节点多时直接拖垮探测周期。
	clashProbeConcurrency = 16
	// maxClashCycleBudget 是整轮探测的时间上限，防止节点数很多时
	// 探测周期被无限拉长（下一轮会与本轮重叠）。
	// 10s 超时 + 16 并发下相当于 480 个节点的完整覆盖，
	// 超出部分记为 unknown（未测到），而不是谎报离线。
	maxClashCycleBudget = 5 * time.Minute
)

// clashCycleBudget 计算整轮探测的时间预算。
// 预算必须按「波次」放大：一批慢节点各占满 timeout 时，
// 后续波次拿不到预算就会被误判为离线。
func clashCycleBudget(nodeCount int, perNode time.Duration) time.Duration {
	if perNode <= 0 {
		perNode = 10 * time.Second
	}
	waves := (nodeCount + clashProbeConcurrency - 1) / clashProbeConcurrency
	if waves < 1 {
		waves = 1
	}
	budget := perNode * time.Duration(waves)
	if budget > maxClashCycleBudget {
		return maxClashCycleBudget
	}
	return budget
}

func probeClash(ctx context.Context, address, secret string, targets []Target, opts settings.NodeProbeOptions) ([]Result, error) {
	base, err := url.Parse(normalizeAddress(address))
	if err != nil || base.Host == "" {
		if err == nil {
			err = errors.New("clash API address has no host")
		}
		return nil, err
	}
	if len(targets) == 0 {
		return nil, nil
	}
	timeout := opts.TimeoutDuration()
	cycleCtx, cancel := context.WithTimeout(ctx, clashCycleBudget(len(targets), timeout))
	defer cancel()
	client := &http.Client{Timeout: timeout}

	results := make([]Result, len(targets))
	reachable := make([]bool, len(targets))
	delayed := make([]bool, len(targets))

	// 固定数量的 worker 拉取任务：不会为每个节点都起一个协程，
	// 节点数很大时也不会把内存和连接数一起放大。
	workerCount := min(clashProbeConcurrency, len(targets))
	queue := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for w := 0; w < workerCount; w++ {
		go func() {
			defer wg.Done()
			for index := range queue {
				result, apiReachable, hasDelay := probeClashOne(cycleCtx, client, base, secret, targets[index], opts)
				results[index] = result
				reachable[index] = apiReachable
				delayed[index] = hasDelay
			}
		}()
	}
dispatch:
	for i := range targets {
		select {
		case queue <- i:
		case <-cycleCtx.Done():
			break dispatch
		}
	}
	close(queue)
	wg.Wait()

	unstarted := fillUnstarted(targets, results)
	completed := len(targets) - unstarted
	// reachable / delayed 只在被真正派发过的下标上为真，
	// 因此未派发的节点天然被排除在「后端可用」判断之外。
	reachableCount, delayedCount := 0, 0
	for i := range results {
		if reachable[i] {
			reachableCount++
		}
		if delayed[i] {
			delayedCount++
		}
	}
	if reachableCount == 0 {
		return results, errors.New("all clash API requests failed")
	}
	if delayedCount == 0 {
		// API 有响应但一个节点都测不出延迟：地址指向了别的内核、
		// tag 与 clash 侧不匹配，或该地址根本不提供 /delay 接口。
		// 此时所有 offline 都是 API 语义错误，不是节点真的不通。
		return results, errors.New("clash API answered but reported no delay for any node")
	}
	if completed < len(targets) {
		return results, fmt.Errorf("clash probe cycle budget exhausted, %d/%d nodes completed", completed, len(targets))
	}
	return results, nil
}

// fillUnstarted 为本轮没轮到探测的节点写入 unknown 结果，并返回其数量。
// 「没测到」不等于「探测失败」：节点数超出整轮预算时，
// 写 offline 会让这些节点每轮都被判离线并持续触发 Webhook 告警。
func fillUnstarted(targets []Target, results []Result) int {
	filled := 0
	for i := range results {
		if results[i].NodeTag != "" {
			continue
		}
		results[i] = Result{
			NodeTag:   targets[i].Tag,
			NodeType:  targets[i].Type,
			Status:    StatusUnknown,
			Error:     "node probe not started before cycle deadline",
			CheckedAt: time.Now(),
		}
		filled++
	}
	return filled
}

// probeClashOne 返回单节点探测结果、clash API 自身是否可达
// （HTTP 层有响应即视为可达，用于区分「内核没起」与「节点不通」），
// 以及是否真的拿到了可解析的延迟值。
// 前两个值都为真但第三个别为真，说明「答得上」不等于「这条 tag 存在」：
// 地址指向别的内核或 tag 失配时，所有请求都会返回 4xx/5xx。
func probeClashOne(ctx context.Context, client *http.Client, base *url.URL, secret string, target Target, opts settings.NodeProbeOptions) (Result, bool, bool) {
	offline := func(reason string) (Result, bool) {
		return Result{
			NodeTag:   target.Tag,
			NodeType:  target.Type,
			Status:    StatusOffline,
			Error:     reason,
			CheckedAt: time.Now(),
		}, false
	}
	endpoint := strings.TrimRight(base.String(), "/") + "/proxies/" + url.PathEscape(target.Tag) + "/delay"
	requestURL, err := url.Parse(endpoint)
	if err != nil {
		result, _ := offline(err.Error())
		return result, false, false
	}
	query := requestURL.Query()
	query.Set("timeout", strconv.FormatInt(maxInt64(opts.TimeoutDuration().Milliseconds(), 1), 10))
	query.Set("url", opts.TestURL)
	requestURL.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		result, _ := offline(err.Error())
		return result, false, false
	}
	if secret != "" {
		request.Header.Set("Authorization", "Bearer "+secret)
	}
	response, err := client.Do(request)
	if err != nil {
		result, _ := offline(err.Error())
		return result, false, false
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 16*1024))
	if err != nil {
		result, _ := offline(err.Error())
		return result, true, false
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		result, _ := offline(fmt.Sprintf("clash API returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body))))
		return result, true, false
	}
	var payload struct {
		Delay int64 `json:"delay"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		result, _ := offline("invalid clash delay response: " + err.Error())
		return result, true, false
	}
	return Result{
		NodeTag:   target.Tag,
		NodeType:  target.Type,
		Status:    statusFromDelay(payload.Delay),
		LatencyMS: payload.Delay,
		Error:     delayFailureMessage(payload.Delay),
		CheckedAt: time.Now(),
	}, true, true
}

func delayFailureMessage(delay int64) string {
	if delay > 0 {
		return ""
	}
	return "clash API reported zero delay"
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
