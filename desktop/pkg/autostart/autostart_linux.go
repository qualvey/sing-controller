//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopEntryTemplate = `[Desktop Entry]
Type=Application
Version=1.0
Name=sing-box-desktop
Comment=sing-box Desktop Client
Exec=%s
Terminal=false
Categories=Network;
`

type LinuxManager struct{}

func New() Manager {
	return &LinuxManager{}
}

func (m *LinuxManager) entryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "autostart", "sing-box-desktop.desktop"), nil
}

func (m *LinuxManager) Enable(execPath string, args ...string) error {
	if execPath == "" {
		p, err := os.Executable()
		if err != nil {
			return err
		}
		execPath = p
	}
	p, err := m.entryPath()
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	content := fmt.Sprintf(desktopEntryTemplate, execPath)
	return os.WriteFile(p, []byte(content), 0644)
}

func (m *LinuxManager) Disable() error {
	p, err := m.entryPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m *LinuxManager) IsEnabled() (bool, error) {
	p, err := m.entryPath()
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
