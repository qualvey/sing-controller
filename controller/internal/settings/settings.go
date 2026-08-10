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
}

type Settings struct {
	Config   string   `json:"config"`
	Listen   string   `json:"listen,omitempty"`
	MinPort  uint16   `json:"min_port,omitempty"`
	Defaults Defaults `json:"defaults,omitempty"`
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
		MinPort: 8000,
		Defaults: Defaults{
			InboundType:  "mixed",
			OutboundType: "vless",
			Listen:       "127.0.0.1",
			ListenPort:   2080,
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
