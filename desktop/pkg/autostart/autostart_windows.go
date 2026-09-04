//go:build windows

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName    = "sing-box-desktop"
)

type WindowsManager struct{}

func New() Manager {
	return &WindowsManager{}
}

func (m *WindowsManager) Enable(execPath string, args ...string) error {
	if execPath == "" {
		p, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}
		execPath = p
	}

	execPath = filepath.Clean(execPath)
	cmdValue := fmt.Sprintf("\"%s\"", execPath)
	if len(args) > 0 {
		cmdValue += " " + strings.Join(args, " ")
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open run registry key: %w", err)
	}
	defer k.Close()

	if err := k.SetStringValue(appName, cmdValue); err != nil {
		return fmt.Errorf("set run value: %w", err)
	}

	return nil
}

func (m *WindowsManager) Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open run registry key: %w", err)
	}
	defer k.Close()

	_ = k.DeleteValue(appName)
	return nil
}

func (m *WindowsManager) IsEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("open run registry key: %w", err)
	}
	defer k.Close()

	_, _, err = k.GetStringValue(appName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
