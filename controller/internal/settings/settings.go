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
	"fmt"
	"os"
	"runtime"
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

// ClashAPIOptions controller 连接 clash API 的配置（可选）
type ClashAPIOptions struct {
	Address string `json:"address,omitempty"` // 例如: http://127.0.0.1:9090
	Secret  string `json:"secret,omitempty"`
}

// ServiceAPIOptions controller 连接 service API（gRPC-Web）的配置（可选）
type ServiceAPIOptions struct {
	Address string `json:"address,omitempty"` // 例如: http://127.0.0.1:9900
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
	RealTime     struct {
		Enabled            bool `json:"enabled"`
		IntervalMS         int  `json:"interval_ms"`
		OnlineThresholdSec int  `json:"online_threshold_sec"`
	} `json:"real_time"`
}

type Manager struct {
	mu     sync.Mutex
	path   string
	values Settings
}

// singBoxServiceConfig 用于提取 sing-box 主配置中 services 数组的最小结构
type singBoxServiceConfig struct {
	Services []struct {
		Type       string `json:"type"`
		Listen     string `json:"listen"`
		ListenPort uint16 `json:"listen_port"`
		Secret     string `json:"secret"`
	} `json:"services"`
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
	singBoxCfg := "/etc/sing-box/config.json"
	if runtime.GOOS == "windows" {
		singBoxCfg = "sing-box.json"
	}
	return Settings{
		Config:  singBoxCfg,
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
		// 平滑回退检查：如果默认的 controller.json 不存在但有旧的 config.json，尝试读取旧文件
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
	// 场景 2：文件存在，先用 defaultSettings() 填底
	values := defaultSettings()
	// Unmarshal 只会覆盖 JSON 文件里明确写了的字段，缺省字段直接继承默认值
	if err := json.Unmarshal(content, &values); err != nil {
		return err
	}
	if values.ServiceAPI == nil || values.ServiceAPI.Address == "" {
		if detected := m.autoDetectServiceAPI(values.Config); detected != nil {
			values.ServiceAPI = detected
		}
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

func (m *Manager) autoDetectServiceAPI(singBoxConfigPath string) *ServiceAPIOptions {
	data, err := os.ReadFile(singBoxConfigPath)
	if err != nil {
		return nil
	}

	var sbCfg singBoxServiceConfig
	if err := json.Unmarshal(data, &sbCfg); err != nil {
		return nil
	}

	for _, s := range sbCfg.Services {
		if s.Type == "api" && s.ListenPort > 0 {
			listenHost := s.Listen
			// 处理通配地址，controller 在本机访问需转为 loopback
			if listenHost == "" || listenHost == "::" || listenHost == "0.0.0.0" {
				listenHost = "127.0.0.1"
			}
			return &ServiceAPIOptions{
				Address: fmt.Sprintf("http://%s:%d", listenHost, s.ListenPort),
				Secret:  s.Secret,
			}
		}
	}
	return nil
}
