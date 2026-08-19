// Package settings 管理 sing-box-controller 自身的配置。
// controller 配置格式（config.json）：
//
//	{
//	  "config": "./sing-box-config.json",   // sing-box 主配置文件路径
//	  "min_port": 8000,                     // 自动分配端口的起点（默认 8000）
//	  "defaults": {                         // 新建 inbound/outbound 的默认值
//	    "inbound_type": "mixed",
//	    "outbound_type": "vless",
//	    "listen": "127.0.0.1",
//	    "listen_port": 2080
//	  }
//	}
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type Defaults struct {
	InboundType  string `json:"inbound_type,omitempty"`
	OutboundType string `json:"outbound_type,omitempty"`
	Listen       string `json:"listen,omitempty"`
	ListenPort   uint16 `json:"listen_port,omitempty"`
	// 新建 outbound 时自动并入指定 selector（默认开；指针区分"缺失"与"显式 false"）
	AttachToSelector *bool  `json:"attach_to_selector,omitempty"`
	ProxySelector    string `json:"proxy_selector,omitempty"`
}

// LogOptions 与 sing-box 配置的 log 段同构（level 枚举复用 sing-box 的）。
type LogOptions struct {
	Level string `json:"level,omitempty"`
}

// ReloadOptions 配置保存后/手动触发 sing-box 重载的方式。
// sing-box 官方重载机制只有 SIGHUP（cmd_run.go 收到 SIGHUP 重载配置）：
//   - auto（默认）：自动探测可用的 init 机制（systemd → openrc/rc-service → OpenWrt/procd → SysV service）
//   - systemd：systemctl reload <service>（推荐，默认服务名 sing-box）
//   - pidfile：读 pid 文件后 kill -HUP
//   - hook：自定义 shell 命令（最灵活）
//   - none：不启用
//
// clash_api 无 reload 端点（已查源码确认），不走该方案。
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
	case "systemd":
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
		return errors.New("未知 reload 模式: " + o.Mode + "（auto/systemd/pidfile/hook/none）")
	}
}

// ClashAPIOptions controller ?? clash API ????（可选）
// ??:?? sing-box ? experimental.clash_api ??（external_controller/secret）
// 显式配置??（?? sing-box ?? clash_api ?指向外部核心）
type ClashAPIOptions struct {
	Address string `json:"address,omitempty"` // ??: http://127.0.0.1:9090
	Secret  string `json:"secret,omitempty"`
}

// ServiceAPIOptions controller ?? service API（gRPC-Web）????（可选）
// ??:?? sing-box ? services[type=api] ??（listen/listen_port/secret）
type ServiceAPIOptions struct {
	Address string `json:"address,omitempty"` // ??: http://127.0.0.1:9900
	Secret  string `json:"secret,omitempty"`
}

type Settings struct {
	Config       string             `json:"config"`
	Listen       string             `json:"listen,omitempty"`
	Log          *LogOptions        `json:"log,omitempty"`
	MinPort      uint16             `json:"min_port,omitempty"`
	Defaults     Defaults           `json:"defaults,omitempty"`
	Reload       *ReloadOptions     `json:"reload,omitempty"`
	ClashAPI     *ClashAPIOptions   `json:"clash_api,omitempty"`
	ServiceAPI   *ServiceAPIOptions `json:"service_api,omitempty"`
	V2rayAPI     string             `json:"v2ray_api,omitempty"`
	PollInterval string             `json:"poll_interval"`
	Pattern      string             `json:"pattern"`
	ResetOnQuery bool               `json:"reset_on_query"`
}

type Manager struct {
	mu     sync.Mutex
	path   string
	values Settings
}

func New(path string) *Manager { return &Manager{path: path} }

func (m *Manager) Path() string { return m.path }

func (m *Manager) PollIntervalDuration() time.Duration {
	d, err := time.ParseDuration(m.values.PollInterval)
	if err != nil || d <= 0 {
		return 10 * time.Second
	}
	return d
}
func defaultSettings() Settings {
	return Settings{
		Config:  "./sing-box-config.json",
		Listen:  "127.0.0.1:8080",
		Log:     &LogOptions{Level: "info"},
		MinPort: 8000,
		Reload:  &ReloadOptions{Mode: "auto"},
		Defaults: Defaults{
			InboundType:   "mixed",
			OutboundType:  "vless",
			Listen:        "127.0.0.1",
			ListenPort:    2080,
			ProxySelector: "Proxy",
		},
	}
}

// Load 读取 controller 配置；不存在时生成默认配置。
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	content, err := os.ReadFile(m.path)
	if errors.Is(err, os.ErrNotExist) {
		m.values = defaultSettings()
		return m.saveLocked()
	}
	if err != nil {
		return err
	}
	var values Settings
	if err := json.Unmarshal(content, &values); err != nil {
		return err
	}
	if values.Config == "" {
		values.Config = "./sing-box-config.json"
	}
	if values.Listen == "" {
		values.Listen = "127.0.0.1:8080"
	}
	if values.Log == nil {
		values.Log = &LogOptions{Level: "info"}
	}
	// 旧配置迁移：attach_to_selector 字段缺失时默认开启
	if values.Defaults.AttachToSelector == nil {
		attached := true
		values.Defaults.AttachToSelector = &attached
	}
	// 重载默认自动适配：未配置/空模式 → auto（systemd → rc-service → OpenWrt service）；显式 none 保留
	if values.Reload == nil {
		values.Reload = &ReloadOptions{Mode: "auto"}
	} else if values.Reload.Mode == "" {
		values.Reload.Mode = "auto"
	}
	if values.MinPort == 0 {
		values.MinPort = 8000
	}
	m.values = values
	return nil
}

// Values 返回当前配置副本。
func (m *Manager) Values() Settings {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.values
}

// Update 更新配置并原子写盘。
func (m *Manager) Update(mutate func(*Settings) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mutate(&m.values); err != nil {
		return err
	}
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	content, err := json.MarshalIndent(m.values, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.path)
}
