package probe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/models"
	"github.com/qualvey/sing-controller/internal/settings"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	deliveryPending  = "pending"
	deliveryRetry    = "retrying"
	deliverySent     = "delivered"
	deliveryFailed   = "failed"
	maxDeliveryTries = 5

	// webhookEventStatusChanged 是节点状态变化事件的类型标识。
	webhookEventStatusChanged = "node.status_changed"
)

type webhookState struct {
	NodeTag   string `json:"node_tag"`
	NodeType  string `json:"node_type"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

type webhookPayload struct {
	Event      string       `json:"event"`
	Source     string       `json:"source"`
	OccurredAt time.Time    `json:"occurred_at"`
	Node       webhookState `json:"node"`
	Previous   webhookState `json:"previous"`
	Current    webhookState `json:"current"`
}

type webhookBatchPayload struct {
	Event      string           `json:"event"`
	Source     string           `json:"source"`
	OccurredAt time.Time        `json:"occurred_at"`
	Events     []webhookPayload `json:"events"`
}

// nextStatusRow 计算探测结果落库后的当前状态行、变化前状态以及是否发生状态
// 变化（只有变化才派发 Webhook）。existing 为 nil 表示该节点首次入库。
func nextStatusRow(existing *models.NodeProbeStatus, result Result) (models.NodeProbeStatus, webhookState, bool) {
	if existing == nil {
		row := models.NodeProbeStatus{
			NodeTag:   result.NodeTag,
			NodeType:  result.NodeType,
			Status:    result.Status,
			LatencyMS: result.LatencyMS,
			LastError: result.Error,
		}
		previous := webhookState{
			NodeTag:  result.NodeTag,
			NodeType: result.NodeType,
			Status:   StatusUnknown,
		}
		if result.Status == StatusOnline {
			row.LastSuccessAt = &result.CheckedAt
		} else {
			row.ConsecutiveFailures = 1
		}
		row.LastCheckedAt = result.CheckedAt
		return row, previous, true
	}

	row := *existing
	previous := stateFromModel(row)
	changed := previous.Status != result.Status
	row.NodeType = result.NodeType
	row.Status = result.Status
	row.LatencyMS = result.LatencyMS
	row.LastError = result.Error
	row.LastCheckedAt = result.CheckedAt
	if result.Status == StatusOnline {
		successAt := result.CheckedAt
		row.LastSuccessAt = &successAt
		row.ConsecutiveFailures = 0
	} else {
		row.ConsecutiveFailures++
	}
	return row, previous, changed
}

// persistResults 把一轮探测结果整体写入数据库。
// 整轮放在同一个事务里：状态表、历史表与 Webhook 投递队列要么一起提交，
// 要么一起回滚，避免出现「状态已变化但通知丢失」的窗口。
func (m *Manager) persistResults(ctx context.Context, results []Result) error {
	database := m.conn()
	if database == nil {
		return ErrUnavailable
	}
	webhooks := m.webhookTargets()
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		transitions := make([]webhookPayload, 0, len(results))
		for _, raw := range results {
			result := normalizeResult(raw)
			var existing models.NodeProbeStatus
			queryErr := tx.Where("node_tag = ?", result.NodeTag).First(&existing).Error
			found := queryErr == nil
			if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
				return queryErr
			}
			var current models.NodeProbeStatus
			var previous webhookState
			var changed bool
			if found {
				current, previous, changed = nextStatusRow(&existing, result)
				if err := tx.Save(&current).Error; err != nil {
					return err
				}
			} else {
				current, previous, changed = nextStatusRow(nil, result)
				if err := tx.Create(&current).Error; err != nil {
					return err
				}
			}
			if err := tx.Create(&models.NodeProbeRecord{
				NodeTag:   result.NodeTag,
				NodeType:  result.NodeType,
				Status:    result.Status,
				LatencyMS: result.LatencyMS,
				Error:     result.Error,
				Trigger:   result.Trigger,
				CheckedAt: result.CheckedAt,
			}).Error; err != nil {
				return err
			}
			// 只有状态真正变化才通知；延迟抖动不应打扰用户。
			if changed {
				transitions = append(transitions, newWebhookPayload(result, previous))
			}
		}
		// A whole probe cycle is one notification boundary. Keep one delivery
		// per endpoint and include all node transitions in that delivery.
		if len(transitions) > 0 {
			return enqueueTransitionBatch(tx, transitions, webhooks)
		}
		return nil
	})
}

func stateFromModel(row models.NodeProbeStatus) webhookState {
	return webhookState{
		NodeTag:   row.NodeTag,
		NodeType:  row.NodeType,
		Status:    row.Status,
		LatencyMS: row.LatencyMS,
		Error:     row.LastError,
	}
}

func stateFromResult(result Result) webhookState {
	return webhookState{
		NodeTag:   result.NodeTag,
		NodeType:  result.NodeType,
		Status:    result.Status,
		LatencyMS: result.LatencyMS,
		Error:     result.Error,
	}
}

// webhookTargets 返回配置了 URL 的通知端点（已去除空 URL 项）。
func (m *Manager) webhookTargets() []settings.WebhookOptions {
	if m.settings == nil {
		return nil
	}
	configured := m.settings.Values().NodeProbe.Webhooks
	targets := make([]settings.WebhookOptions, 0, len(configured))
	for _, webhook := range configured {
		if strings.TrimSpace(webhook.URL) == "" {
			continue
		}
		targets = append(targets, webhook)
	}
	return targets
}

func newWebhookPayload(result Result, previous webhookState) webhookPayload {
	return webhookPayload{
		Event:      webhookEventStatusChanged,
		Source:     "sing-controller",
		OccurredAt: result.CheckedAt,
		Node:       webhookState{NodeTag: result.NodeTag, NodeType: result.NodeType},
		Previous:   previous,
		Current:    stateFromResult(result),
	}
}

// enqueueTransitions 保留单事件调用语义，供兼容调用方使用。
func enqueueTransitions(tx *gorm.DB, result Result, previous webhookState, webhooks []settings.WebhookOptions) error {
	return enqueueTransitionBatch(tx, []webhookPayload{newWebhookPayload(result, previous)}, webhooks)
}

// enqueueTransitionBatch 在事务内为每个端点排入一条投递记录。
// 任一端点排入失败都不应影响其他端点：记录已经入库，
// 丢失的只是单个端点的通知，不能因此回滚整轮探测。
func enqueueTransitionBatch(tx *gorm.DB, transitions []webhookPayload, webhooks []settings.WebhookOptions) error {
	if len(webhooks) == 0 || len(transitions) == 0 {
		return nil
	}
	var payload any = transitions[0]
	if len(transitions) > 1 {
		payload = webhookBatchPayload{
			Event:      webhookEventStatusChanged,
			Source:     "sing-controller",
			OccurredAt: transitions[len(transitions)-1].OccurredAt,
			Events:     transitions,
		}
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for _, webhook := range webhooks {
		nodeTag := "*"
		if len(transitions) == 1 {
			nodeTag = transitions[0].Node.NodeTag
		}
		delivery := models.WebhookDelivery{
			EventID:   uuid.NewString(),
			EventType: webhookEventStatusChanged,
			NodeTag:   nodeTag,
			URL:       strings.TrimSpace(webhook.URL),
			Secret:    webhook.Secret,
			Payload:   datatypes.JSON(content),
			Status:    deliveryPending,
		}
		if err := tx.Create(&delivery).Error; err != nil {
			logger.Warn("[Probe] enqueue webhook batch (%d nodes) to %s failed: %v", len(transitions), delivery.URL, err)
		}
	}
	return nil
}

const (
	// cleanupInterval 是历史清理的最小间隔，避免每轮探测都扫全表。
	cleanupInterval = time.Hour
	// cleanupBatchSize 限制单条 DELETE 的行数，避免长事务锁表。
	cleanupBatchSize = 5000
	// deliveryBatchSize 是单轮最多投递的通知条数。
	deliveryBatchSize = 20
	// deliveryConcurrency 控制并行投递数：单个挂死的端点
	// 不能把整批通知堵在队头（每条最多 10s 超时）。
	deliveryConcurrency = 4
	// deliveryRetentionDays 是终态投递记录的保留天数，防止该表无限增长。
	deliveryRetentionDays = 7
)

// cleanupHistory 分批删除过期历史，并清掉出站已被删除后残留的状态行。
// 使用 m.lastCleanup 做节流：清理是周期性维护，不该占用每轮探测的事务。
// 只有清理成功才推进节流时间戳，否则一次临时数据库抖动会让清理停摆一小时。
func (m *Manager) cleanupHistory(ctx context.Context, retentionDays int, tags []string) error {
	database := m.conn()
	if database == nil {
		return nil
	}
	m.mu.RLock()
	last := m.lastCleanup
	m.mu.RUnlock()
	if !last.IsZero() && time.Since(last) < cleanupInterval {
		return nil
	}
	if err := m.pruneRecords(ctx, database, retentionDays, tags); err != nil {
		return err
	}
	m.mu.Lock()
	m.lastCleanup = time.Now()
	m.mu.Unlock()
	return nil
}

func (m *Manager) pruneRecords(ctx context.Context, database *gorm.DB, retentionDays int, tags []string) error {
	db := database.WithContext(ctx)
	if retentionDays > 0 {
		cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
		for {
			// Delete 会忽略 Limit，这里用子查询显式限制单批行数，
			// 避免一条大 DELETE 长时间持锁拖慢探测写入。
			deleted := db.Where("id IN (?)",
				db.Model(&models.NodeProbeRecord{}).Select("id").
					Where("checked_at < ?", cutoff).Limit(cleanupBatchSize),
			).Delete(&models.NodeProbeRecord{})
			if deleted.Error != nil {
				return deleted.Error
			}
			if deleted.RowsAffected < cleanupBatchSize {
				break
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
	// 出站已删除的节点不应继续出现在状态列表里。
	// 没有目标时不做清理：此时无法区分「节点被删」与「配置读取异常」。
	if len(tags) > 0 {
		if err := db.Where("node_tag NOT IN ?", tags).Delete(&models.NodeProbeStatus{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) deliveryLoop(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		if err := m.processDeliveries(ctx); err != nil && ctx.Err() == nil {
			logger.Warn("[Probe] webhook delivery cycle failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Manager) processDeliveries(ctx context.Context) error {
	database := m.conn()
	if database == nil {
		return ErrUnavailable
	}
	now := time.Now()
	var deliveries []models.WebhookDelivery
	if err := database.WithContext(ctx).
		Where("status IN ? AND (next_retry_at IS NULL OR next_retry_at <= ?)", []string{deliveryPending, deliveryRetry}, now).
		Order("id ASC").Limit(deliveryBatchSize).Find(&deliveries).Error; err != nil {
		return err
	}
	if len(deliveries) > 0 {
		// 并行投递：某个端点挂死时，其余通知仍能按时发出。
		var wg sync.WaitGroup
		slots := make(chan struct{}, deliveryConcurrency)
		for _, delivery := range deliveries {
			delivery := delivery
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				wg.Wait()
				return ctx.Err()
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-slots }()
				if err := m.deliverOne(ctx, database, delivery); err != nil && ctx.Err() == nil {
					logger.Warn("[Probe] webhook delivery %d failed: %v", delivery.ID, err)
				}
			}()
		}
		wg.Wait()
	}
	// 无论本轮是否有待投都执行清理：投递积压时终态记录同样会堆积。
	m.pruneDeliveries(ctx, database)
	return nil
}

// pruneDeliveries 按小时节流地清理终态投递记录，防止 webhook_deliveries 无限膨胀。
// 与历史清理一致：只有删除成功才推进节流时间戳。
func (m *Manager) pruneDeliveries(ctx context.Context, database *gorm.DB) {
	m.mu.RLock()
	last := m.lastDeliveryCleanup
	m.mu.RUnlock()
	if !last.IsZero() && time.Since(last) < cleanupInterval {
		return
	}
	cutoff := time.Now().Add(-deliveryRetentionDays * 24 * time.Hour)
	if err := database.WithContext(ctx).
		Where("status IN ? AND updated_at < ?", []string{deliverySent, deliveryFailed}, cutoff).
		Delete(&models.WebhookDelivery{}).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("[Probe] webhook delivery cleanup failed: %v", err)
		}
		return
	}
	m.mu.Lock()
	m.lastDeliveryCleanup = time.Now()
	m.mu.Unlock()
}

func (m *Manager) deliverOne(ctx context.Context, database *gorm.DB, delivery models.WebhookDelivery) error {
	attempts := delivery.Attempts + 1
	statusCode, err := postWebhook(ctx, delivery)
	if err == nil && statusCode >= 200 && statusCode < 300 {
		deliveredAt := time.Now()
		return database.WithContext(ctx).Model(&models.WebhookDelivery{}).Where("id = ?", delivery.ID).Updates(map[string]any{
			"status":          deliverySent,
			"attempts":        attempts,
			"response_status": statusCode,
			"last_error":      "",
			"delivered_at":    deliveredAt,
			"next_retry_at":   nil,
			"updated_at":      deliveredAt,
		}).Error
	}

	message := ""
	if err != nil {
		message = err.Error()
	} else {
		message = "webhook returned non-2xx status"
	}
	permanent := statusCode >= 400 && statusCode < 500 && statusCode != http.StatusRequestTimeout && statusCode != http.StatusTooManyRequests
	newStatus := deliveryRetry
	if permanent || attempts >= maxDeliveryTries {
		newStatus = deliveryFailed
	}
	var nextRetry *time.Time
	if newStatus == deliveryRetry {
		delay := time.Duration(1<<minInt(attempts, 6)) * time.Second
		if delay > 5*time.Minute {
			delay = 5 * time.Minute
		}
		next := time.Now().Add(delay)
		nextRetry = &next
	}
	return database.WithContext(ctx).Model(&models.WebhookDelivery{}).Where("id = ?", delivery.ID).Updates(map[string]any{
		"status":          newStatus,
		"attempts":        attempts,
		"response_status": statusCode,
		"last_error":      truncateError(message),
		"next_retry_at":   nextRetry,
		"updated_at":      time.Now(),
	}).Error
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
