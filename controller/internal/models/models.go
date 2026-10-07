package models

import (
	"time"

	"gorm.io/datatypes"
)

// DefaultCycleDays 用户未单独配置周期天数时的默认值
const DefaultCycleDays = 30

// User 对齐现有数据库 users 表 + 周期字段（cycle_start/cycle_days 为新增列）
type User struct {
	ID           uint64     `gorm:"primaryKey" json:"id"`
	Email        string     `gorm:"uniqueIndex;size:128" json:"email"`
	UUID         string     `gorm:"size:64" json:"uuid"`
	Password     string     `gorm:"size:128" json:"password"`
	Flow         string     `gorm:"size:64;default:xtls-rprx-vision" json:"flow"`
	Enable       bool       `gorm:"default:true" json:"enable"`
	TrafficLimit int64      `gorm:"default:0" json:"traffic_limit"` // 字节；0=不限额
	ExpireAt     *time.Time `json:"expire_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	// 周期字段（新系统新增，兼容迁移）
	CycleStart *time.Time `gorm:"index" json:"cycle_start"` // 周期锚点；NULL 时回退 CreatedAt
	CycleDays  int        `gorm:"default:30" json:"cycle_days"`
}

// InboundNode 对齐现有 inbound_nodes 表
type InboundNode struct {
	ID                  uint64    `gorm:"primaryKey" json:"id"`
	Tag                 string    `gorm:"uniqueIndex;size:128" json:"tag"`
	Type                string    `gorm:"size:32" json:"type"`
	Listen              string    `gorm:"size:64;default:::" json:"listen"`
	ListenPort          int64     `gorm:"not null" json:"listen_port"`
	Enable              bool      `gorm:"default:true" json:"enable"`
	ServerName          string    `gorm:"size:255" json:"server_name"`
	HandshakeServer     string    `gorm:"size:255" json:"handshake_server"`
	HandshakePort       int64     `gorm:"default:443" json:"handshake_port"`
	PrivateKey          string    `gorm:"size:255" json:"private_key"`
	ShortID             string    `gorm:"size:255" json:"short_id"`
	CongestionControl   string    `gorm:"size:32;default:bbr" json:"congestion_control"`
	AuthTimeout         string    `gorm:"size:32;default:3s" json:"auth_timeout"`
	ZeroRttHandshake    bool      `gorm:"default:false" json:"zero_rtt_handshake"`
	CertificateProvider string    `gorm:"size:128" json:"certificate_provider"`
	ALPN                string    `gorm:"size:64;default:h3" json:"alpn"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// TrafficLog 对齐现有 traffic_logs 表（增量）
type TrafficLog struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	Category      string    `gorm:"size:32;index:idx_log_cat_target_time,priority:1" json:"category"`
	TargetName    string    `gorm:"size:255;index:idx_log_cat_target_time,priority:2" json:"target_name"`
	UplinkDelta   int64     `gorm:"default:0" json:"uplink_delta"`
	DownlinkDelta int64     `gorm:"default:0" json:"downlink_delta"`
	Timestamp     time.Time `gorm:"index:idx_log_cat_target_time,priority:3" json:"timestamp"`
}

// TrafficTotal 对齐现有 traffic_totals 表（累计）
type TrafficTotal struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	Category      string    `gorm:"size:32;uniqueIndex:idx_cat_target,priority:1" json:"category"`
	TargetName    string    `gorm:"size:255;uniqueIndex:idx_cat_target,priority:2" json:"target_name"`
	UplinkBytes   int64     `gorm:"default:0" json:"uplink_bytes"`
	DownlinkBytes int64     `gorm:"default:0" json:"downlink_bytes"`
	TotalBytes    int64     `gorm:"default:0" json:"total_bytes"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UserInboundBinding 对齐现有 user_inbound_bindings 表
type UserInboundBinding struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	UserID    uint64 `gorm:"uniqueIndex:idx_user_inbound,priority:1" json:"user_id"`
	InboundID uint64 `gorm:"uniqueIndex:idx_user_inbound,priority:2" json:"inbound_id"`
}

type OutboundNode struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Tag       string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"tag"`
	Type      string         `gorm:"type:varchar(32);not null;index" json:"type"`
	Config    datatypes.JSON `gorm:"type:json" json:"config" `
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// NodeProbeStatus 保存每个出站节点最近一次探测结果。
type NodeProbeStatus struct {
	ID                  uint64     `gorm:"primaryKey" json:"id"`
	NodeTag             string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"node_tag"`
	NodeType            string     `gorm:"type:varchar(64);not null" json:"node_type"`
	Status              string     `gorm:"type:varchar(16);not null;index" json:"status"`
	LatencyMS           int64      `gorm:"not null;default:0" json:"latency_ms"`
	LastError           string     `gorm:"type:text" json:"last_error,omitempty"`
	LastCheckedAt       time.Time  `gorm:"not null;index" json:"last_checked_at"`
	LastSuccessAt       *time.Time `json:"last_success_at,omitempty"`
	ConsecutiveFailures int        `gorm:"not null;default:0" json:"consecutive_failures"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (NodeProbeStatus) TableName() string { return "node_probe_statuses" }

// NodeProbeRecord 保存节点探测历史，供趋势查询和故障审计使用。
type NodeProbeRecord struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	NodeTag   string    `gorm:"type:varchar(128);not null;index:idx_node_probe_records_tag_time,priority:1" json:"node_tag"`
	NodeType  string    `gorm:"type:varchar(64);not null" json:"node_type"`
	Status    string    `gorm:"type:varchar(16);not null;index" json:"status"`
	LatencyMS int64     `gorm:"not null;default:0" json:"latency_ms"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	Trigger   string    `gorm:"type:varchar(16);not null;default:scheduler" json:"trigger"`
	CheckedAt time.Time `gorm:"not null;index:idx_node_probe_records_tag_time,priority:2" json:"checked_at"`
}

func (NodeProbeRecord) TableName() string { return "node_probe_records" }

// WebhookDelivery 保存 Webhook 投递状态，支持失败重试和审计。
type WebhookDelivery struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	EventID        string         `gorm:"type:varchar(64);uniqueIndex;not null" json:"event_id"`
	EventType      string         `gorm:"type:varchar(64);not null" json:"event_type"`
	NodeTag        string         `gorm:"type:varchar(128);not null;index" json:"node_tag"`
	URL            string         `gorm:"type:varchar(2048);not null" json:"url"`
	Secret         string         `gorm:"type:varchar(256)" json:"-"`
	Payload        datatypes.JSON `gorm:"type:json;not null" json:"payload"`
	Status         string         `gorm:"type:varchar(16);not null;index" json:"status"`
	Attempts       int            `gorm:"not null;default:0" json:"attempts"`
	ResponseStatus int            `gorm:"not null;default:0" json:"response_status,omitempty"`
	LastError      string         `gorm:"type:text" json:"last_error,omitempty"`
	NextRetryAt    *time.Time     `gorm:"index" json:"next_retry_at,omitempty"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (WebhookDelivery) TableName() string { return "webhook_deliveries" }

// CycleAnchor 返回周期锚点（未设置时回退创建时间）
func (u User) CycleAnchor() time.Time {
	if u.CycleStart != nil && !u.CycleStart.IsZero() {
		return *u.CycleStart
	}
	return u.CreatedAt
}

// CycleDuration 返回周期时长
func (u User) CycleDuration() time.Duration {
	days := u.CycleDays
	if days <= 0 {
		days = DefaultCycleDays
	}
	return time.Duration(days) * 24 * time.Hour
}

// CurrentCycleWindow 返回 now 所处的当前周期窗口 [start, end)
func (u User) CurrentCycleWindow(now time.Time) (time.Time, time.Time) {
	anchor := u.CycleAnchor()
	span := u.CycleDuration()
	if span <= 0 {
		span = time.Duration(DefaultCycleDays) * 24 * time.Hour
	}
	n := int64(now.Sub(anchor) / span)
	if n < 0 {
		n = 0
	}
	start := anchor.Add(time.Duration(n) * span)
	return start, start.Add(span)
}

// IsOverLimit 是否超出流量限制
func (u User) IsOverLimit(used int64) bool {
	return u.TrafficLimit > 0 && used > u.TrafficLimit
}

// Expired 是否已到期
func (u User) Expired(now time.Time) bool {
	return u.ExpireAt != nil && now.After(*u.ExpireAt)
}
