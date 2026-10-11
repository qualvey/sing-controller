package provider

import (
	"time"
)

// ProviderType 定义节点源类型
type ProviderType string

const (
	ProviderTypeRemote ProviderType = "remote" // 远程订阅 URL
	ProviderTypeFile   ProviderType = "file"   // 本地独立文件
	ProviderTypeInline ProviderType = "inline" // 直接内联录入
)

// SubscriptionFormat 订阅格式
type SubscriptionFormat string

const (
	FormatAuto    SubscriptionFormat = "auto"     // 自动检测
	FormatSingBox SubscriptionFormat = "sing-box" // sing-box 原生 JSON (outbounds 数组或完整配置)
	FormatClash   SubscriptionFormat = "clash"    // Clash YAML (proxies 数组)
	FormatBase64  SubscriptionFormat = "base64"   // Base64 编码的节点 URI 列表
)

// Provider 表示一个独立的节点源/订阅
type Provider struct {
	ID                string             `json:"id"`
	Tag               string             `json:"tag"`                 // 唯一标识 (用于生成 outbound-<tag>.json)
	Name              string             `json:"name"`                // 可读名称
	Type              ProviderType       `json:"type"`                // 节点源类型
	URL               string             `json:"url,omitempty"`       // 远程订阅地址
	Format            SubscriptionFormat `json:"format"`              // 订阅格式
	UseFullConfig     bool               `json:"use_full_config"`     // 使用订阅提供的完整 sing-box 配置，而不是仅提取节点
	UseProviderRules  bool               `json:"use_provider_rules"`  // 使用订阅完整配置中的 route/dns 规则
	UpdateIntervalMin int                `json:"update_interval_min"` // 自动更新间隔（分钟，0 表示不自动更新）
	LastUpdated       *time.Time         `json:"last_updated,omitempty"`
	LastETag          string             `json:"last_etag,omitempty"`
	NodeCount         int                `json:"node_count"`
	Enabled           bool               `json:"enabled"`
	LastError         string             `json:"last_error,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// Profile 表示一个配置组/场景方案
type Profile struct {
	ID              string    `json:"id"`
	Tag             string    `json:"tag"`                        // 唯一标识 (如 default, gaming, travel)
	Name            string    `json:"name"`                       // 方案名称
	Description     string    `json:"description,omitempty"`      // 方案描述
	ProviderTags    []string  `json:"provider_tags"`              // 启用的 Provider Tag 列表
	TargetSelectors []string  `json:"target_selectors,omitempty"` // 节点自动挂载的目标 selector/urltest tag（默认 ["Proxy", "Auto"]）
	AutoReload      bool      `json:"auto_reload"`                // 订阅刷新后是否自动热重载内核
	IsActive        bool      `json:"is_active"`                  // 是否为当前正在运行的活动方案
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
