package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReloadValidateModes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		err  bool
	}{
		{"empty defaults to auto", "", "auto", false},
		{"auto", "auto", "auto", false},
		{"none", "none", "none", false},
		{"systemd", "systemd", "systemd", false},
		{"procd", "procd", "procd", false},
		{"rc-service", "rc-service", "rc-service", false},
		{"pidfile with path", "pidfile", "pidfile", false},
		{"hook with command", "hook", "hook", false},
		{"unknown rejected", "docker", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := &ReloadOptions{Mode: tc.in}
			if tc.name == "pidfile with path" {
				o.PidFile = "/run/sing-box.pid"
			}
			if tc.name == "hook with command" {
				o.Hook = "systemctl reload sing-box"
			}
			err := o.Validate()
			if tc.err {
				if err == nil {
					t.Fatalf("Validate(%q) expected error, got nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate(%q) unexpected error: %v", tc.in, err)
			}
			if o.Mode != tc.want {
				t.Errorf("Validate(%q) mode = %q, want %q", tc.in, o.Mode, tc.want)
			}
		})
	}
}

// TestLoadModularAndLegacyCompatibility 测试新模块化配置解析与旧配置的平滑迁移兼容性
func TestLoadModularAndLegacyCompatibility(t *testing.T) {
	dir := t.TempDir()

	// 1. 测试旧版扁平配置迁移
	legacyJSON := `{
		"config": "./sing-box-config.json",
		"listen": "127.0.0.1:9999",
		"v2ray_api": "127.0.0.1:10085",
		"poll_interval": "5s",
		"reload": {"mode": "auto", "after_save": true}
	}`
	legacyPath := filepath.Join(dir, "legacy.json")
	if err := os.WriteFile(legacyPath, []byte(legacyJSON), 0644); err != nil {
		t.Fatal(err)
	}
	m := New(legacyPath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load legacy: %v", err)
	}
	v := m.Values()
	if v.Core.ConfigPath != "./sing-box-config.json" {
		t.Errorf("Core.ConfigPath = %s, want ./sing-box-config.json", v.Core.ConfigPath)
	}
	if v.Server.ListenPort != 9999 {
		t.Errorf("Server.ListenPort = %d, want 9999", v.Server.ListenPort)
	}
	if v.API.V2Ray.Address != "127.0.0.1:10085" {
		t.Errorf("API.V2Ray.Address = %s, want 127.0.0.1:10085", v.API.V2Ray.Address)
	}
	if v.Stats.PollInterval != "5s" {
		t.Errorf("Stats.PollInterval = %s, want 5s", v.Stats.PollInterval)
	}

	// 2. 测试全新模块化配置
	modularJSON := `{
		"server": {
			"listen": "::",
			"listen_port": 8888,
			"secret": "topsecret"
		},
		"log": {
			"level": "debug",
			"output": "file",
			"outfile": "/tmp/test.log"
		},
		"core": {
			"config_path": "/etc/sing-box/config.json",
			"managed": false,
			"reload": {
				"mode": "systemd",
				"service": "sing-box-prod"
			}
		},
		"api": {
			"clash": { "address": "http://127.0.0.1:9090", "secret": "sec" }
		},
		"stats": {
			"poll_interval": "15s"
		}
	}`
	modPath := filepath.Join(dir, "modular.json")
	if err := os.WriteFile(modPath, []byte(modularJSON), 0644); err != nil {
		t.Fatal(err)
	}
	m2 := New(modPath)
	if err := m2.Load(); err != nil {
		t.Fatalf("Load modular: %v", err)
	}
	v2 := m2.Values()
	if v2.Server.ListenPort != 8888 || v2.Server.Secret != "topsecret" {
		t.Errorf("Server mismatch: %+v", v2.Server)
	}
	if v2.Server.Addr() != ":8888" {
		t.Errorf("Server.Addr() = %s, want :8888", v2.Server.Addr())
	}
	if v2.Log.Level != "debug" || v2.Log.Outfile != "/tmp/test.log" {
		t.Errorf("Log mismatch: %+v", v2.Log)
	}
	if *v2.Core.Managed != false || v2.Core.Reload.Mode != "systemd" {
		t.Errorf("Core mismatch: %+v", v2.Core)
	}
	if v2.API.Clash.Address != "http://127.0.0.1:9090" {
		t.Errorf("Clash API mismatch: %+v", v2.API.Clash)
	}
	if v2.NodeHealth.Interval != "5m" || v2.NodeHealth.FailureThreshold != 2 || v2.NodeHealth.Enabled {
		t.Errorf("node health defaults mismatch: %+v", v2.NodeHealth)
	}
}

func TestLoadJSONCAndDetectCommentedSingBoxConfig(t *testing.T) {
	dir := t.TempDir()
	corePath := filepath.Join(dir, "sing-box.json")
	coreConfig := `{
		// Service API used by the local controller
		"services": [{
			"type": "api",
			"listen": "127.0.0.1",
			"listen_port": 9901
		}],
		"outbounds": [{"type": "direct", "tag": "direct"}]
	}`
	if err := os.WriteFile(corePath, []byte(coreConfig), 0644); err != nil {
		t.Fatal(err)
	}

	configPathJSON, err := json.Marshal(corePath)
	if err != nil {
		t.Fatal(err)
	}
	settingsJSON := `{
		// Comments are allowed in controller config too.
		"core": {"config_path": ` + string(configPathJSON) + `},
		"api": {"clash": {"address": "http://127.0.0.1:9090"}}
	}`
	settingsPath := filepath.Join(dir, "controller.json")
	if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0644); err != nil {
		t.Fatal(err)
	}

	manager := New(settingsPath)
	if err := manager.Load(); err != nil {
		t.Fatalf("Load() JSONC settings: %v", err)
	}
	values := manager.Values()
	if values.Core.ConfigPath != corePath {
		t.Fatalf("Core.ConfigPath = %q, want %q", values.Core.ConfigPath, corePath)
	}
	if values.API.Service.Address != "http://127.0.0.1:9901" {
		t.Fatalf("auto-detected service API = %+v, want port 9901", values.API.Service)
	}
}

func TestNodeHealthOptionsValidate(t *testing.T) {
	options := DefaultNodeHealthOptions()
	if err := options.Validate(); err != nil {
		t.Fatalf("valid defaults rejected: %v", err)
	}

	cases := []struct {
		name   string
		update func(*NodeHealthOptions)
	}{
		{"short interval", func(o *NodeHealthOptions) { o.Interval = "1s" }},
		{"invalid test url", func(o *NodeHealthOptions) { o.TestURL = "file:///tmp/probe" }},
		{"invalid webhook url", func(o *NodeHealthOptions) { o.WebhookURL = "not-a-url" }},
		{"invalid concurrency", func(o *NodeHealthOptions) { o.Concurrency = 100 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := DefaultNodeHealthOptions()
			tc.update(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatal("Validate() accepted invalid options")
			}
		})
	}
}
