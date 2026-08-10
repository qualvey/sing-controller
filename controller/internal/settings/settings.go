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

type Settings struct {
	Config   string      `json:"config"`
	Listen   string      `json:"listen,omitempty"`
	Log      *LogOptions `json:"log,omitempty"`
	MinPort  uint16      `json:"min_port,omitempty"`
	Defaults Defaults    `json:"defaults,omitempty"`
}

type Manager struct {
	mu     sync.Mutex
	path   string
	values Settings
}

func New(path string) *Manager { return &Manager{path: path} }

func (m *Manager) Path() string { return m.path }

func defaultSettings() Settings {
	return Settings{
		Config:  "./sing-box-config.json",
		Listen:  "127.0.0.1:8080",
		Log:     &LogOptions{Level: "info"},
		MinPort: 8000,
		Defaults: Defaults{
			InboundType:  "mixed",
			OutboundType: "vless",
			Listen:       "127.0.0.1",
			ListenPort:   2080,
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
