// Package settings 管理 sing-controller 自身的配置。
// 遵循 sing-box 模块化理念设计。
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/jsonc"
)

// ServerOptions 定义 Controller HTTP / WebUI 服务的监听配置
type ServerOptions struct {
	Listen     string `json:"listen,omitempty"`
	ListenPort uint16 `json:"listen_port,omitempty"`
	Secret     string `json:"secret,omitempty"`
}

// Addr 返回供 http.Server 使用的标准化监听地址（如 "127.0.0.1:8080" 或 ":8080"）
func (s *ServerOptions) Addr() string {
	if s == nil {
		return ":8080"
	}
	listen := strings.TrimSpace(s.Listen)
	port := s.ListenPort
	if port == 0 {
		port = 8080
	}

	// 兼容：如果 listen 已经显式包含了端口（如 127.0.0.1:8080）且 listen_port 未指定
	if strings.Contains(listen, ":") && !strings.HasPrefix(listen, "[") && s.ListenPort == 0 {
		if _, _, err := net.SplitHostPort(listen); err == nil {
			return listen
		}
	}

	if listen == "" || listen == "::" || listen == "0.0.0.0" {
		return ":" + strconv.Itoa(int(port))
	}
	return net.JoinHostPort(listen, strconv.Itoa(int(port)))
}

// LogOptions 与 sing-box 配置的 log 段同构
type LogOptions struct {
	Level   string `json:"level,omitempty"`
	Output  string `json:"output,omitempty"`  // stderr | stdout | file
	Outfile string `json:"outfile,omitempty"` // 日志落盘路径
}

// ReloadOptions sing-box 重载机制选项
type ReloadOptions struct {
	Mode      string `json:"mode,omitempty"`
	Service   string `json:"service,omitempty"`
	PidFile   string `json:"pid_file,omitempty"`
	Hook      string `json:"hook,omitempty"`
	AfterSave bool   `json:"after_save,omitempty"`
}

func (o *ReloadOptions) Validate() error {
	if o == nil {
		return nil
	}
	switch o.Mode {
	case "", "auto":
		o.Mode = "auto"
		return nil
	case "none":
		return nil
	case "systemd", "procd", "rc-service":
		return nil
	case "pidfile":
		if o.PidFile == "" {
			return errors.New("pidfile 模式需要 pid_file")
		}
		return nil
	case "hook":
		if o.Hook == "" {
			return errors.New("hook 模式需要 hook 命令")
		}
		return nil
	default:
		return errors.New("未知 reload 模式: " + o.Mode + "（auto/systemd/procd/rc-service/pidfile/hook/none）")
	}
}

// CoreOptions 管理 sing-box 内核生命周期
type CoreOptions struct {
	ConfigPath string         `json:"config_path,omitempty"`
	Managed    *bool          `json:"managed,omitempty"`
	BinaryPath string         `json:"binary_path,omitempty"`
	WorkingDir string         `json:"working_dir,omitempty"`
	Reload     *ReloadOptions `json:"reload,omitempty"`
}

// EndpointAPI 通用的上游 API 地址与密钥配置
type EndpointAPI struct {
	Address string `json:"address,omitempty"`
	Secret  string `json:"secret,omitempty"`
}

// APIOptions 定义与 sing-box 交互的各类端点
type APIOptions struct {
	Clash   *EndpointAPI `json:"clash,omitempty"`
	Service *EndpointAPI `json:"service,omitempty"`
	V2Ray   *EndpointAPI `json:"v2ray,omitempty"`
}

// RealTimeStats 实时推送统计配置
type RealTimeStats struct {
	Enabled            bool `json:"enabled"`
	IntervalMS         int  `json:"interval_ms"`
	OnlineThresholdSec int  `json:"online_threshold_sec"`
}

// StatsOptions 流量监控统计采集配置
type StatsOptions struct {
	PollInterval string        `json:"poll_interval,omitempty"`
	Pattern      string        `json:"pattern,omitempty"`
	ResetOnQuery bool          `json:"reset_on_query,omitempty"`
	RealTime     RealTimeStats `json:"real_time"`
}

// NodeHealthOptions configures scheduled outbound availability probes and webhook notifications.
type NodeHealthOptions struct {
	Enabled           bool   `json:"enabled"`
	Interval          string `json:"interval,omitempty"`
	TestURL           string `json:"test_url,omitempty"`
	TimeoutMS         int    `json:"timeout_ms,omitempty"`
	FailureThreshold  int    `json:"failure_threshold,omitempty"`
	Concurrency       int    `json:"concurrency,omitempty"`
	WebhookURL        string `json:"webhook_url,omitempty"`
	WebhookToken      string `json:"webhook_token,omitempty"`
	WebhookSignSecret string `json:"webhook_sign_secret,omitempty"`
}

func DefaultNodeHealthOptions() NodeHealthOptions {
	return NodeHealthOptions{
		Interval:         "5m",
		TestURL:          "https://www.gstatic.com/generate_204",
		TimeoutMS:        5000,
		FailureThreshold: 2,
		Concurrency:      5,
	}
}

func (o *NodeHealthOptions) FillDefaults() {
	defaults := DefaultNodeHealthOptions()
	if o.Interval == "" {
		o.Interval = defaults.Interval
	}
	if o.TestURL == "" {
		o.TestURL = defaults.TestURL
	}
	if o.TimeoutMS == 0 {
		o.TimeoutMS = defaults.TimeoutMS
	}
	if o.FailureThreshold == 0 {
		o.FailureThreshold = defaults.FailureThreshold
	}
	if o.Concurrency == 0 {
		o.Concurrency = defaults.Concurrency
	}
}

func (o *NodeHealthOptions) Normalize() {
	o.FillDefaults()
	defaults := DefaultNodeHealthOptions()
	interval, intervalErr := time.ParseDuration(o.Interval)
	if intervalErr != nil || interval < 30*time.Second {
		o.Interval = defaults.Interval
	}
	if o.TimeoutMS < 500 || o.TimeoutMS > 60000 {
		o.TimeoutMS = defaults.TimeoutMS
	}
	if o.FailureThreshold < 1 || o.FailureThreshold > 10 {
		o.FailureThreshold = defaults.FailureThreshold
	}
	if o.Concurrency < 1 || o.Concurrency > 32 {
		o.Concurrency = defaults.Concurrency
	}
}

func (o NodeHealthOptions) Validate() error {
	if _, err := time.ParseDuration(o.Interval); err != nil {
		return fmt.Errorf("node_health.interval 格式无效: %w", err)
	}
	interval, _ := time.ParseDuration(o.Interval)
	if interval < 30*time.Second {
		return errors.New("node_health.interval 不能小于 30s")
	}
	if o.TimeoutMS < 500 || o.TimeoutMS > 60000 {
		return errors.New("node_health.timeout_ms 必须在 500 到 60000 之间")
	}
	if o.FailureThreshold < 1 || o.FailureThreshold > 10 {
		return errors.New("node_health.failure_threshold 必须在 1 到 10 之间")
	}
	if o.Concurrency < 1 || o.Concurrency > 32 {
		return errors.New("node_health.concurrency 必须在 1 到 32 之间")
	}
	if o.TestURL != "" {
		u, err := url.ParseRequestURI(o.TestURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("node_health.test_url 必须是有效的 HTTP/HTTPS URL")
		}
	}
	if o.WebhookURL != "" {
		u, err := url.ParseRequestURI(o.WebhookURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("node_health.webhook_url 必须是有效的 HTTP/HTTPS URL")
		}
	}

	return nil
}

// DatabaseOptions 配置 Controller 使用的 PostgreSQL 数据库。
// DSN 留空时不会初始化数据库，节点探测也会保持未启用状态。
type DatabaseOptions struct {
	DSN string `json:"dsn,omitempty"`
}

// WebhookOptions 配置节点状态变化通知端点。
// 配置了 URL 即视为启用；Secret 会作为 X-Secret 请求头发送。
type WebhookOptions struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

// NodeProbeOptions 节点在线状态与延迟探测配置。
type NodeProbeOptions struct {
	Enabled              bool             `json:"enabled"`
	Interval             string           `json:"interval,omitempty"`
	TestURL              string           `json:"test_url,omitempty"`
	Timeout              string           `json:"timeout,omitempty"`
	HistoryRetentionDays int              `json:"history_retention_days,omitempty"`
	Webhooks             []WebhookOptions `json:"webhooks,omitempty"`
}

const (
	// DefaultNodeProbeInterval 是节点探测的默认周期。
	DefaultNodeProbeInterval  = "120s"
	defaultNodeProbeTestURL   = "https://www.gstatic.com/generate_204"
	defaultNodeProbeTimeout   = "10s"
	defaultProbeRetentionDays = 30
)

// IntervalDuration 返回节点探测周期，非法值回退到默认 120 秒。
func (o NodeProbeOptions) IntervalDuration() time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(o.Interval))
	if err != nil || d <= 0 {
		d, _ = time.ParseDuration(DefaultNodeProbeInterval)
	}
	return d
}

// TimeoutDuration 返回单次探测超时，非法值回退到默认 10 秒。
func (o NodeProbeOptions) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(o.Timeout))
	if err != nil || d <= 0 {
		d, _ = time.ParseDuration(defaultNodeProbeTimeout)
	}
	return d
}

func (o *NodeProbeOptions) Normalize() {
	if o == nil {
		return
	}
	o.Interval = strings.TrimSpace(o.Interval)
	if o.Interval == "" {
		o.Interval = DefaultNodeProbeInterval
	}
	o.TestURL = strings.TrimSpace(o.TestURL)
	if o.TestURL == "" {
		o.TestURL = defaultNodeProbeTestURL
	}
	o.Timeout = strings.TrimSpace(o.Timeout)
	if o.Timeout == "" {
		o.Timeout = defaultNodeProbeTimeout
	}
	if o.HistoryRetentionDays <= 0 {
		o.HistoryRetentionDays = defaultProbeRetentionDays
	}
	// 重新构造切片而不是就地改写：NodeProbeOptions 常来自浅拷贝，
	// 就地修改会写到别处（甚至已发布的 Values() 快照）共享的底层数组上。
	if len(o.Webhooks) > 0 {
		normalized := make([]WebhookOptions, len(o.Webhooks))
		for i, webhook := range o.Webhooks {
			webhook.URL = strings.TrimSpace(webhook.URL)
			normalized[i] = webhook
		}
		o.Webhooks = normalized
	}
}

// Validate 校验节点探测配置。URL 在保存设置时校验，避免后台任务持续失败。
func (o NodeProbeOptions) Validate() error {
	if _, err := time.ParseDuration(o.Interval); err != nil || strings.TrimSpace(o.Interval) == "" {
		return fmt.Errorf("node_probe.interval 必须是合法的 Go duration")
	}
	if d, err := time.ParseDuration(o.Timeout); err != nil || d <= 0 {
		return fmt.Errorf("node_probe.timeout 必须是合法的正 Go duration")
	}
	if o.HistoryRetentionDays < 0 {
		return fmt.Errorf("node_probe.history_retention_days 不能为负数")
	}
	if _, err := url.ParseRequestURI(o.TestURL); err != nil {
		return fmt.Errorf("node_probe.test_url 无效: %w", err)
	}
	for _, webhook := range o.Webhooks {
		u, err := url.Parse(webhook.URL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("node_probe.webhooks.url 无效: %q", webhook.URL)
		}
	}
	return nil
}

// Settings 是 controller 的主配置根结构
type Settings struct {
	Server     ServerOptions     `json:"server"`
	Log        *LogOptions       `json:"log,omitempty"`
	Core       CoreOptions       `json:"core"`
	API        APIOptions        `json:"api"`
	Stats      StatsOptions      `json:"stats"`
	NodeHealth NodeHealthOptions `json:"node_health"`
	Database   DatabaseOptions   `json:"database"`
	NodeProbe  NodeProbeOptions  `json:"node_probe"`
}

func (s *Settings) UnmarshalJSON(data []byte) error {
	cleanData, err := jsonc.StripComments(data)
	if err != nil {
		return err
	}
	type Alias Settings
	aux := (*Alias)(s)
	if err := json.Unmarshal(cleanData, aux); err != nil {
		return err
	}

	var old rawOldSettings
	if err := json.Unmarshal(cleanData, &old); err == nil {
		if old.Config != "" && s.Core.ConfigPath == "" {
			s.Core.ConfigPath = old.Config
		}
		if old.Listen != "" && s.Server.Listen == "" && s.Server.ListenPort == 0 {
			if h, p, err := net.SplitHostPort(old.Listen); err == nil {
				s.Server.Listen = h
				if pu, err := strconv.ParseUint(p, 10, 16); err == nil {
					s.Server.ListenPort = uint16(pu)
				}
			} else {
				s.Server.Listen = old.Listen
			}
		}
		if old.V2RayAPI != "" && (s.API.V2Ray == nil || s.API.V2Ray.Address == "") {
			s.API.V2Ray = &EndpointAPI{Address: old.V2RayAPI}
		}
		if old.ClashAPI != nil && s.API.Clash == nil {
			s.API.Clash = old.ClashAPI
		}
		if old.ServiceAPI != nil && s.API.Service == nil {
			s.API.Service = old.ServiceAPI
		}
		if old.PollInterval != "" && s.Stats.PollInterval == "" {
			s.Stats.PollInterval = old.PollInterval
		}
		if old.Pattern != "" && s.Stats.Pattern == "" {
			s.Stats.Pattern = old.Pattern
		}
		if old.ResetOnQuery {
			s.Stats.ResetOnQuery = true
		}
		if old.RealTime != nil && !s.Stats.RealTime.Enabled {
			s.Stats.RealTime = *old.RealTime
		}
		if old.Reload != nil && s.Core.Reload == nil {
			s.Core.Reload = old.Reload
		}
	}
	return nil
}

type Manager struct {
	mu     sync.Mutex
	path   string
	values Settings
}

func New(path string) *Manager { return &Manager{path: path} }

func (m *Manager) Path() string { return m.path }

func (m *Manager) PollIntervalDuration() time.Duration {
	d, err := time.ParseDuration(m.values.Stats.PollInterval)
	if err != nil || d <= 0 {
		return 10 * time.Second
	}
	return d
}

func defaultSettings() Settings {
	singBoxCfg := "/etc/sing-box/config.json"
	managed := runtime.GOOS == "windows"
	if runtime.GOOS == "windows" {
		singBoxCfg = "sing-box.json"
	}

	return Settings{
		Server: ServerOptions{
			Listen:     "::",
			ListenPort: 8080,
		},
		Log: &LogOptions{
			Level:  "info",
			Output: "stderr",
		},
		Core: CoreOptions{
			ConfigPath: singBoxCfg,
			Managed:    &managed,
			Reload:     &ReloadOptions{Mode: "auto"},
		},
		API: APIOptions{
			Clash:   &EndpointAPI{},
			Service: &EndpointAPI{},
			V2Ray:   &EndpointAPI{},
		},
		Stats: StatsOptions{
			PollInterval: "10s",
			RealTime: RealTimeStats{
				Enabled:            false,
				IntervalMS:         1000,
				OnlineThresholdSec: 60,
			},
		},
		NodeHealth: DefaultNodeHealthOptions(),
		NodeProbe: NodeProbeOptions{
			Enabled:              true,
			Interval:             DefaultNodeProbeInterval,
			TestURL:              defaultNodeProbeTestURL,
			Timeout:              defaultNodeProbeTimeout,
			HistoryRetentionDays: defaultProbeRetentionDays,
		},
	}
}

// rawOldSettings 仅用于平滑迁移旧版扁平化配置
type rawOldSettings struct {
	Config       string         `json:"config"`
	Listen       string         `json:"listen"`
	MinPort      uint16         `json:"min_port"`
	V2RayAPI     string         `json:"v2ray_api"`
	PollInterval string         `json:"poll_interval"`
	Pattern      string         `json:"pattern"`
	ResetOnQuery bool           `json:"reset_on_query"`
	ClashAPI     *EndpointAPI   `json:"clash_api"`
	ServiceAPI   *EndpointAPI   `json:"service_api"`
	Reload       *ReloadOptions `json:"reload"`
	RealTime     *RealTimeStats `json:"real_time"`
}

// Load 读取 controller 配置，自动兼容旧版配置并完成缺省值补全。
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	content, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		if m.path == "controller.json" {
			if oldContent, oldErr := os.ReadFile("config.json"); oldErr == nil {
				content = oldContent
				err = nil
			}
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		m.values = defaultSettings()
		return m.saveLocked()
	}
	if err != nil {
		return err
	}
	content, err = jsonc.StripComments(content)
	if err != nil {
		return err
	}

	values := defaultSettings()
	// 1. 尝试反序列化新版模块化配置
	if err := json.Unmarshal(content, &values); err != nil {
		return err
	}

	// 2. 检查并兼容旧版扁平字段
	var old rawOldSettings
	_ = json.Unmarshal(content, &old)

	if old.Config != "" && (values.Core.ConfigPath == "" || values.Core.ConfigPath == defaultSettings().Core.ConfigPath) {
		values.Core.ConfigPath = old.Config
	}
	if old.Listen != "" && values.Server.Listen == "::" && values.Server.ListenPort == 8080 {
		// 旧格式如果带有端口，解析它
		if h, p, err := net.SplitHostPort(old.Listen); err == nil {
			values.Server.Listen = h
			if pu, err := strconv.ParseUint(p, 10, 16); err == nil {
				values.Server.ListenPort = uint16(pu)
			}
		} else {
			values.Server.Listen = old.Listen
		}
	}
	if old.V2RayAPI != "" && values.API.V2Ray.Address == "" {
		values.API.V2Ray.Address = old.V2RayAPI
	}
	if old.ClashAPI != nil && values.API.Clash.Address == "" {
		values.API.Clash = old.ClashAPI
	}
	if old.ServiceAPI != nil && values.API.Service.Address == "" {
		values.API.Service = old.ServiceAPI
	}
	if old.PollInterval != "" && (values.Stats.PollInterval == "" || values.Stats.PollInterval == "10s") {
		values.Stats.PollInterval = old.PollInterval
	}
	if old.Pattern != "" && values.Stats.Pattern == "" {
		values.Stats.Pattern = old.Pattern
	}
	if old.ResetOnQuery {
		values.Stats.ResetOnQuery = true
	}
	if old.RealTime != nil && !values.Stats.RealTime.Enabled {
		values.Stats.RealTime = *old.RealTime
	}
	if old.Reload != nil && values.Core.Reload == nil {
		values.Core.Reload = old.Reload
	}

	// 3. 校验与默认值强化
	if values.Core.Reload == nil {
		values.Core.Reload = &ReloadOptions{Mode: "auto"}
	} else if values.Core.Reload.Mode == "" {
		values.Core.Reload.Mode = "auto"
	}
	if values.Core.Managed == nil {
		defaultManaged := runtime.GOOS == "windows"
		values.Core.Managed = &defaultManaged
	}
	values.NodeHealth.Normalize()
	values.NodeProbe.Normalize()
	if err := values.NodeProbe.Validate(); err != nil {
		return err
	}

	// 4. 自动推断 service API
	if values.API.Service == nil || values.API.Service.Address == "" {
		if detected := m.autoDetectServiceAPI(values.Core.ConfigPath); detected != nil {
			values.API.Service = detected
		}
	}

	m.values = values
	return nil
}

// Values 返回当前配置副本。
// Webhooks 是切片，必须深拷贝：否则调用方拿到的是与内部配置
// 共享的底层数组，锁外的写入会与其他读路径形成数据竞争。
func (m *Manager) Values() Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	values := m.values
	values.NodeProbe.Webhooks = cloneWebhooks(values.NodeProbe.Webhooks)
	return values
}

func cloneWebhooks(source []WebhookOptions) []WebhookOptions {
	if len(source) == 0 {
		return nil
	}
	return append(make([]WebhookOptions, 0, len(source)), source...)
}

// Update 更新配置并原子写盘。
// 先在副本上变更与校验，只有全部通过才提交到内存与磁盘，
// 避免校验失败或写盘失败后内存里残留半成品配置。
func (m *Manager) Update(mutate func(*Settings) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	previous := m.values
	next := previous
	next.NodeProbe.Webhooks = cloneWebhooks(previous.NodeProbe.Webhooks)
	if err := mutate(&next); err != nil {
		return err
	}
	next.NodeProbe.Normalize()
	if err := next.NodeProbe.Validate(); err != nil {
		return err
	}
	m.values = next
	if err := m.saveLocked(); err != nil {
		m.values = previous
		return err
	}
	return nil
}

func (m *Manager) saveLocked() error {
	content, err := json.MarshalIndent(m.values, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}

type singBoxServiceConfig struct {
	Services []struct {
		Type       string `json:"type"`
		Listen     string `json:"listen"`
		ListenPort uint16 `json:"listen_port"`
		Secret     string `json:"secret"`
	} `json:"services"`
}

func (m *Manager) autoDetectServiceAPI(singBoxConfigPath string) *EndpointAPI {
	data, err := os.ReadFile(singBoxConfigPath)
	if err != nil {
		return nil
	}

	cleanData, err := jsonc.StripComments(data)
	if err != nil {
		return nil
	}
	var sbCfg singBoxServiceConfig
	if err := json.Unmarshal(cleanData, &sbCfg); err != nil {
		return nil
	}

	for _, s := range sbCfg.Services {
		if s.Type == "api" && s.ListenPort > 0 {
			listenHost := s.Listen
			if listenHost == "" || listenHost == "::" || listenHost == "0.0.0.0" {
				listenHost = "127.0.0.1"
			}
			return &EndpointAPI{
				Address: fmt.Sprintf("http://%s:%d", listenHost, s.ListenPort),
				Secret:  s.Secret,
			}
		}
	}
	return nil
}
