//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.qualvey.sing-box-desktop</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>ProcessType</key>
	<string>Interactive</string>
</dict>
</plist>
`

type DarwinManager struct{}

func New() Manager {
	return &DarwinManager{}
}

func (m *DarwinManager) plistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.qualvey.sing-box-desktop.plist"), nil
}

func (m *DarwinManager) Enable(execPath string, args ...string) error {
	if execPath == "" {
		p, err := os.Executable()
		if err != nil {
			return err
		}
		execPath = p
	}
	p, err := m.plistPath()
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	content := fmt.Sprintf(plistTemplate, execPath)
	return os.WriteFile(p, []byte(content), 0644)
}

func (m *DarwinManager) Disable() error {
	p, err := m.plistPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *DarwinManager) IsEnabled() (bool, error) {
	p, err := m.plistPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
