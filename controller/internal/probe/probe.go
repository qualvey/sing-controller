package probe

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/models"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"gorm.io/gorm"
)

const (
	StatusUnknown  = "unknown"
	StatusOnline   = "online"
	StatusOffline  = "offline"
	StatusDegraded = "degraded"

	TriggerScheduler = "scheduler"
	TriggerManual    = "manual"
)

var (
	ErrUnavailable = errors.New("node probe persistence is unavailable")
	ErrNoTargets   = errors.New("no probeable outbound nodes found")
	// ErrNodeNotFound 表示请求的出站节点不在可探测列表中（已删除或类型不可探测）。
	ErrNodeNotFound = errors.New("outbound node not found")
)

// Target is a value-copy snapshot of an outbound. It is safe to retain after
// the store read lock has been released.
type Target struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
}

// Result is the result of one node probe.
type Result struct {
	NodeTag   string    `json:"node_tag"`
	NodeType  string    `json:"node_type"`
	Status    string    `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
	Trigger   string    `json:"trigger"`
	CheckedAt time.Time `json:"checked_at"`
}

// StatusView is the API representation of the current node state.
type StatusView struct {
	NodeTag             string     `json:"node_tag"`
	NodeType            string     `json:"node_type"`
	Status              string     `json:"status"`
	LatencyMS           int64      `json:"latency_ms"`
	LastError           string     `json:"last_error,omitempty"`
	LastCheckedAt       time.Time  `json:"last_checked_at,omitempty"`
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
}

// HistoryPoint is the API representation of a persisted probe record.
type HistoryPoint struct {
	ID        uint64    `json:"id"`
	NodeTag   string    `json:"node_tag"`
	NodeType  string    `json:"node_type"`
	Status    string    `json:"status"`
	LatencyMS int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
	Trigger   string    `json:"trigger"`
	CheckedAt time.Time `json:"checked_at"`
}

// ProbeFunc performs one backend round. It is injectable to keep the manager
// testable without a live sing-box service.
type ProbeFunc func(context.Context, []Target, settings.NodeProbeOptions) ([]Result, error)

// Manager owns the periodic loop, persistence, and asynchronous webhook
// delivery for node probes.
type Manager struct {
	settings *settings.Manager
	store    *store.Store
	database *gorm.DB
	prober   ProbeFunc

	// gate 只允许一轮探测同时运行（周期任务与手动触发共享），
	// 用 channel 而非 Mutex 是为了等锁时能被 ctx 取消，
	// 否则 Engine 关闭时调度协程会永久阻塞在锁上。
	gate chan struct{}

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	stopped bool
	// cycles 统计正在执行的探测轮次（含 HTTP 手动触发）。
	// 手动触发跑在请求协程上，不在 wg 里，Stop 必须单独等待，
	// 否则 Engine 关闭数据库时会有事务写到已关闭的连接池上。
	cycles sync.WaitGroup
	// cycleCancels 保存在途轮次的取消函数，Stop 时统一取消，
	// 避免关闭流程被一轮最长 2 分钟的探测拖住。
	cycleCancels map[uint64]context.CancelFunc
	cycleSeq     uint64
	// lastCleanup 记录上次历史清理时间，避免每轮探测都执行大范围 DELETE。
	lastCleanup time.Time
	// lastDeliveryCleanup 记录上次投递记录清理时间。
	lastDeliveryCleanup time.Time
}

// NewManager creates a node probe manager. A nil database keeps the manager in
// an unavailable state, while the rest of Controller can continue running.
func NewManager(cfg *settings.Manager, st *store.Store, database *gorm.DB) *Manager {
	return &Manager{settings: cfg, store: st, database: database, gate: make(chan struct{}, 1)}
}

// NewManagerWithProber is primarily useful for tests and embedding applications.
func NewManagerWithProber(cfg *settings.Manager, st *store.Store, database *gorm.DB, prober ProbeFunc) *Manager {
	m := NewManager(cfg, st, database)
	m.prober = prober
	return m
}

// Available reports whether persistence is configured and available.
func (m *Manager) Available() bool {
	return m.conn() != nil
}

// conn 返回当前数据库连接快照。读路径统一走这里，
// 避免与 SetDatabase / Invalidate 的写入产生数据竞争。
func (m *Manager) conn() *gorm.DB {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.database
}

// SetDatabase 绑定（或重新绑定）数据库连接。
// 连接在 Engine 启动时建立，关闭流程里需要显式解绑，
// 否则 Manager 会继续持有一个已关闭的连接池并对外报告可用。
func (m *Manager) SetDatabase(database *gorm.DB) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.database = database
	m.mu.Unlock()
}

// Invalidate 解绑数据库连接。关闭数据库后调用，
// 使后续 API 请求返回 503 而不是写入已关闭的连接池。
func (m *Manager) Invalidate() {
	m.SetDatabase(nil)
}

// Start starts the periodic probe and webhook delivery workers. Calling Start
// more than once is harmless.
func (m *Manager) Start(parent context.Context) error {
	if m == nil || m.settings == nil || m.store == nil {
		return ErrUnavailable
	}
	if !m.Available() {
		return ErrUnavailable
	}
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	m.ctx = ctx
	m.cancel = cancel
	m.running = true
	m.stopped = false
	m.wg.Add(2)
	m.mu.Unlock()

	go m.run(ctx)
	go m.deliveryLoop(ctx)
	return nil
}

// Stop stops all workers and waits for in-flight operations to finish.
func (m *Manager) Stop() {
	if m == nil {
		return
	}
	m.mu.Lock()
	cancel := m.cancel
	m.cancel = nil
	m.ctx = nil
	m.running = false
	m.stopped = true
	// 先摘出在途轮次的取消函数：它们的 ctx 来自 HTTP 请求，
	// 不受 Engine 的 cancel 影响，不主动取消会把关闭拖到整轮超时。
	pending := make([]context.CancelFunc, 0, len(m.cycleCancels))
	for _, cycleCancel := range m.cycleCancels {
		pending = append(pending, cycleCancel)
	}
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	for _, cycleCancel := range pending {
		cycleCancel()
	}
	m.wg.Wait()
	// 手动触发不在 wg 中：等待它们收尾，避免使用已关闭的数据库连接。
	m.cycles.Wait()
}

// acquire 串行化探测轮次，等待期间响应 ctx 取消。
func (m *Manager) acquire(ctx context.Context) error {
	m.mu.RLock()
	gate := m.gate
	m.mu.RUnlock()
	if gate == nil {
		return ErrUnavailable
	}
	select {
	case gate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) release() {
	m.mu.RLock()
	gate := m.gate
	m.mu.RUnlock()
	if gate == nil {
		return
	}
	select {
	case <-gate:
	default:
	}
}

// cycle 是一次在途探测轮次的句柄。手动触发跑在 HTTP 请求协程上，
// 请求 ctx 不受 Engine 关闭影响，因此每轮都派生一个可被 Stop 取消的 ctx，
// 否则关闭流程最多要等一整轮探测超时。
type cycle struct {
	id     uint64
	ctx    context.Context
	cancel context.CancelFunc
}

// beginCycle 登记一轮探测。检查 stopped 与计数递增必须在同一把锁内完成，
// 否则 Stop 之后的 Wait 可能看不到刚启动的轮次。
func (m *Manager) beginCycle(ctx context.Context) (cycle, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return cycle{}, ErrUnavailable
	}
	cycleCtx, cancel := context.WithCancel(ctx)
	m.cycleSeq++
	current := cycle{id: m.cycleSeq, ctx: cycleCtx, cancel: cancel}
	if m.cycleCancels == nil {
		m.cycleCancels = make(map[uint64]context.CancelFunc)
	}
	m.cycleCancels[current.id] = cancel
	m.cycles.Add(1)
	return current, nil
}

func (m *Manager) endCycle(current cycle) {
	m.mu.Lock()
	delete(m.cycleCancels, current.id)
	m.mu.Unlock()
	current.cancel()
	m.cycles.Done()
}

// Trigger runs one probe immediately. The trigger is recorded in history and
// is independent from whether periodic probing is enabled.
func (m *Manager) Trigger(ctx context.Context, trigger string) ([]Result, error) {
	if !m.Available() {
		return nil, ErrUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return m.runCycle(ctx, trigger)
}

// TriggerOne runs a probe for one currently configured outbound tag.
func (m *Manager) TriggerOne(ctx context.Context, tag, trigger string) ([]Result, error) {
	if !m.Available() {
		return nil, ErrUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	target, ok := m.findTarget(tag)
	if !ok {
		return nil, fmt.Errorf("%w: outbound node %q", ErrNodeNotFound, tag)
	}
	return m.runCycle(ctx, trigger, target)
}

// ListStatuses returns persisted states, adding unknown entries for configured
// nodes that have not been probed yet.
func (m *Manager) ListStatuses(ctx context.Context) ([]StatusView, error) {
	database := m.conn()
	if database == nil {
		return nil, ErrUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var rows []models.NodeProbeStatus
	if err := database.WithContext(ctx).Order("node_tag ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	byTag := make(map[string]StatusView, len(rows))
	for _, row := range rows {
		byTag[row.NodeTag] = statusView(row)
	}
	for _, target := range m.targets() {
		if _, ok := byTag[target.Tag]; !ok {
			byTag[target.Tag] = StatusView{
				NodeTag:  target.Tag,
				NodeType: target.Type,
				Status:   StatusUnknown,
			}
		}
	}
	result := make([]StatusView, 0, len(byTag))
	for _, row := range byTag {
		result = append(result, row)
	}
	sortStatusViews(result)
	return result, nil
}

// History returns the newest records for a node. A non-positive limit uses a
// safe default; the hard cap prevents accidental unbounded API responses.
func (m *Manager) History(ctx context.Context, tag string, limit int) ([]HistoryPoint, error) {
	database := m.conn()
	if database == nil {
		return nil, ErrUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	var rows []models.NodeProbeRecord
	query := database.WithContext(ctx).Where("node_tag = ?", tag).
		Order("checked_at DESC, id DESC").Limit(limit)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]HistoryPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, HistoryPoint{
			ID:        row.ID,
			NodeTag:   row.NodeTag,
			NodeType:  row.NodeType,
			Status:    row.Status,
			LatencyMS: row.LatencyMS,
			Error:     row.Error,
			Trigger:   row.Trigger,
			CheckedAt: row.CheckedAt,
		})
	}
	return result, nil
}

func (m *Manager) run(ctx context.Context) {
	defer m.wg.Done()
	// Probe once at startup, then use the configured interval. A missing or
	// disabled configuration is rechecked periodically so API changes take
	// effect without restarting Controller.
	for {
		if m.settings.Values().NodeProbe.Enabled && m.Available() {
			if _, err := m.runCycle(ctx, TriggerScheduler); err != nil && ctx.Err() == nil {
				logger.Warn("[Probe] scheduled probe failed: %v", err)
			}
		}
		wait := 5 * time.Second
		if m.settings.Values().NodeProbe.Enabled {
			wait = m.settings.Values().NodeProbe.IntervalDuration()
		}
		if wait <= 0 {
			wait = 120 * time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func (m *Manager) runCycle(ctx context.Context, trigger string, only ...Target) ([]Result, error) {
	if m.conn() == nil {
		return nil, ErrUnavailable
	}
	if m.settings == nil {
		return nil, ErrUnavailable
	}
	if err := m.acquire(ctx); err != nil {
		return nil, err
	}
	defer m.release()
	// 登记后再取快照：Stop 之后的触发必须被拒绝，
	// 且 Stop 会等待这里登记的轮次结束再关闭数据库。
	cycle, err := m.beginCycle(ctx)
	if err != nil {
		return nil, err
	}
	defer m.endCycle(cycle)
	targets := m.targets()
	if len(only) > 0 {
		targets = only
	}
	if len(targets) == 0 {
		return nil, ErrNoTargets
	}
	if strings.TrimSpace(trigger) == "" {
		trigger = TriggerManual
	}
	opts := m.settings.Values().NodeProbe
	prober := m.prober
	if prober == nil {
		prober = m.probeBackends
	}
	results, probeErr := prober(cycle.ctx, targets, opts)
	// Controller shutdown cancels the probe context while sing-box may still
	// be tearing down. Do not turn that normal disconnect into node failures.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cycle.ctx.Err(); err != nil {
		return nil, err
	}
	results = mergeResults(targets, results, trigger, probeErr)
	if err := m.persistResults(ctx, results); err != nil {
		return results, err
	}
	// 只有整轮探测才有资格清理失效状态行。
	if err := m.cleanupHistory(ctx, opts.HistoryRetentionDays, pruneTagsFor(targets, only)); err != nil {
		logger.Warn("[Probe] history cleanup failed: %v", err)
	}
	return results, probeErr
}

// tagsOf 提取目标 tag 列表，用于清理失效状态行。
func tagsOf(targets []Target) []string {
	tags := make([]string, 0, len(targets))
	for _, target := range targets {
		tags = append(tags, target.Tag)
	}
	return tags
}

// pruneTagsFor 返回本轮允许清理的「仍存在的节点」白名单。
// 单节点触发时 targets 只含被探测的那一个 tag，
// 拿它当白名单等于 DELETE ... WHERE node_tag NOT IN ('jp-01')，
// 会把其余所有节点的状态行清空并触发一轮全量 Webhook。
// 因此单节点触发不参与失效行清理。
//
// 复用本轮 targets 快照而不是重读 store：配置切换的中间态
// 可能给出不完整列表，把真实存在的状态行误删。
func pruneTagsFor(targets []Target, only []Target) []string {
	if len(only) > 0 {
		return nil
	}
	return tagsOf(targets)
}

func mergeResults(targets []Target, results []Result, trigger string, probeErr error) []Result {
	byTag := make(map[string]Result, len(results))
	for _, result := range results {
		result.NodeTag = strings.TrimSpace(result.NodeTag)
		if result.NodeTag == "" {
			continue
		}
		if result.Trigger == "" {
			result.Trigger = trigger
		}
		byTag[result.NodeTag] = result
	}
	now := time.Now()
	merged := make([]Result, 0, len(targets))
	for _, target := range targets {
		if result, ok := byTag[target.Tag]; ok {
			if result.NodeType == "" {
				result.NodeType = target.Type
			}
			if result.Trigger == "" {
				result.Trigger = trigger
			}
			if result.CheckedAt.IsZero() {
				result.CheckedAt = now
			}
			merged = append(merged, normalizeResult(result))
			continue
		}
		status := StatusUnknown
		message := "probe result missing"
		if probeErr != nil {
			message = probeErr.Error()
		}
		merged = append(merged, normalizeResult(Result{
			NodeTag:   target.Tag,
			NodeType:  target.Type,
			Status:    status,
			Error:     message,
			Trigger:   trigger,
			CheckedAt: now,
		}))
	}
	return merged
}

func normalizeResult(result Result) Result {
	result.NodeTag = strings.TrimSpace(result.NodeTag)
	result.Status = strings.ToLower(strings.TrimSpace(result.Status))
	if result.Status != StatusOnline && result.Status != StatusOffline && result.Status != StatusDegraded {
		result.Status = StatusUnknown
	}
	if result.LatencyMS < 0 {
		result.LatencyMS = 0
	}
	if result.Status != StatusOnline {
		result.LatencyMS = 0
	}
	result.Error = truncateError(result.Error)
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now()
	}
	if result.Trigger == "" {
		result.Trigger = TriggerManual
	}
	return result
}

func truncateError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 2048 {
		return message[:2048]
	}
	return message
}

func (m *Manager) findTarget(tag string) (Target, bool) {
	for _, target := range m.targets() {
		if target.Tag == tag {
			return target, true
		}
	}
	return Target{}, false
}

func statusView(row models.NodeProbeStatus) StatusView {
	return StatusView{
		NodeTag:             row.NodeTag,
		NodeType:            row.NodeType,
		Status:              row.Status,
		LatencyMS:           row.LatencyMS,
		LastError:           row.LastError,
		LastCheckedAt:       row.LastCheckedAt,
		LastSuccessAt:       row.LastSuccessAt,
		ConsecutiveFailures: row.ConsecutiveFailures,
	}
}

func sortStatusViews(items []StatusView) {
	// A small insertion sort keeps this package independent of a UI-specific
	// comparator and is more than sufficient for the usual node count.
	for i := 1; i < len(items); i++ {
		value := items[i]
		j := i - 1
		for j >= 0 && items[j].NodeTag > value.NodeTag {
			items[j+1] = items[j]
			j--
		}
		items[j+1] = value
	}
}
