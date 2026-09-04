package sysproxy

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// DefaultBypassList 默认直连/绕过域名与 IP 网段
	DefaultBypassList = []string{
		"<local>",
		"localhost",
		"127.*",
		"10.*",
		"172.16.*",
		"172.17.*",
		"172.18.*",
		"172.19.*",
		"172.20.*",
		"172.21.*",
		"172.22.*",
		"172.23.*",
		"172.24.*",
		"172.25.*",
		"172.26.*",
		"172.27.*",
		"172.28.*",
		"172.29.*",
		"172.30.*",
		"172.31.*",
		"192.168.*",
	}

	ErrNotImplemented = errors.New("system proxy not implemented on this platform")
)

// Manager 系统代理管理器统一接口
type Manager interface {
	Enable(host string, port int, bypass []string) error
	Disable() error
	IsEnabled() (bool, error)
	GetServer() (string, error)
}

// FormatBypass 将 bypass 切片合并为系统格式（以分号分隔）
func FormatBypass(bypass []string) string {
	if len(bypass) == 0 {
		bypass = DefaultBypassList
	}
	return strings.Join(bypass, ";")
}

// FormatProxyServer 格式化代理服务器地址
func FormatProxyServer(host string, port int) string {
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", host, port)
}
