//go:build windows

package sysproxy

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const (
	internetSettingsKeyPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37
)

var (
	modWininet             = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOptionW = modWininet.NewProc("InternetSetOptionW")
)

func internetSetOption(option uintptr) error {
	r1, _, err := procInternetSetOptionW.Call(0, option, 0, 0)
	if r1 == 0 {
		return err
	}
	return nil
}

func notifySettingsChanged() {
	_ = internetSetOption(internetOptionSettingsChanged)
	_ = internetSetOption(internetOptionRefresh)
}

type WindowsManager struct{}

// New 创建 Windows 平台的系统代理管理器
func New() Manager {
	return &WindowsManager{}
}

func (m *WindowsManager) Enable(host string, port int, bypass []string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open internet settings registry key: %w", err)
	}
	defer k.Close()

	server := FormatProxyServer(host, port)
	bypassStr := FormatBypass(bypass)

	// 设置 ProxyEnable = 1
	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}

	// 设置 ProxyServer = "127.0.0.1:port"
	if err := k.SetStringValue("ProxyServer", server); err != nil {
		return fmt.Errorf("set ProxyServer: %w", err)
	}

	// 设置 ProxyOverride = "<local>;127.*;..."
	if err := k.SetStringValue("ProxyOverride", bypassStr); err != nil {
		return fmt.Errorf("set ProxyOverride: %w", err)
	}

	// 广播配置变更通知系统组件刷新
	notifySettingsChanged()
	return nil
}

func (m *WindowsManager) Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open internet settings registry key: %w", err)
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("set ProxyEnable: %w", err)
	}

	notifySettingsChanged()
	return nil
}

func (m *WindowsManager) IsEnabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("open internet settings registry key: %w", err)
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil {
		return false, fmt.Errorf("get ProxyEnable: %w", err)
	}

	return val == 1, nil
}

func (m *WindowsManager) GetServer() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("open internet settings registry key: %w", err)
	}
	defer k.Close()

	val, _, err := k.GetStringValue("ProxyServer")
	if err != nil {
		return "", fmt.Errorf("get ProxyServer: %w", err)
	}

	return val, nil
}
