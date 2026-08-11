package settings

import (
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

// TestLoadDefaultsReloadAuto 旧配置无 reload 段 / 空 mode → 迁移为 auto（自动适配）；显式 none 保留。
func TestLoadDefaultsReloadAuto(t *testing.T) {
	dir := t.TempDir()

	// 无 reload 段 → auto
	noReload := `{"config":"./sing-box-config.json","min_port":8000}`
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(noReload), 0o644); err != nil {
		t.Fatal(err)
	}
	m := New(filepath.Join(dir, "a.json"))
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.values.Reload == nil || m.values.Reload.Mode != "auto" {
		t.Errorf("no reload section: mode = %+v, want auto", m.values.Reload)
	}

	// 空 mode → auto
	emptyMode := `{"config":"./sing-box-config.json","reload":{"after_save":true}}`
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(emptyMode), 0o644); err != nil {
		t.Fatal(err)
	}
	m2 := New(filepath.Join(dir, "b.json"))
	if err := m2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m2.values.Reload.Mode != "auto" {
		t.Errorf("empty mode: mode = %q, want auto", m2.values.Reload.Mode)
	}
	if !m2.values.Reload.AfterSave {
		t.Error("after_save lost during load")
	}

	// 显式 none → 保留
	explicitNone := `{"config":"./sing-box-config.json","reload":{"mode":"none"}}`
	if err := os.WriteFile(filepath.Join(dir, "c.json"), []byte(explicitNone), 0o644); err != nil {
		t.Fatal(err)
	}
	m3 := New(filepath.Join(dir, "c.json"))
	if err := m3.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m3.values.Reload.Mode != "none" {
		t.Errorf("explicit none: mode = %q, want none", m3.values.Reload.Mode)
	}

	// 新生成默认配置 → auto
	m4 := New(filepath.Join(dir, "d.json"))
	if err := m4.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m4.values.Reload == nil || m4.values.Reload.Mode != "auto" {
		t.Errorf("default settings: reload = %+v, want auto", m4.values.Reload)
	}
}
