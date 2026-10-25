//go:build darwin

package sysproxy

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type DarwinManager struct{}

func New() Manager {
	return &DarwinManager{}
}

// getActiveNetworkServices 获取 macOS 上的活动网络服务（如 Wi-Fi、Ethernet）
func (m *DarwinManager) getActiveNetworkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	var services []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.Contains(l, "*") { // 包含星号表示被禁用的服务
			continue
		}
		services = append(services, l)
	}
	return services, nil
}

func (m *DarwinManager) Enable(host string, port int, bypass []string) error {
	if host == "" {
		host = "127.0.0.1"
	}
	services, err := m.getActiveNetworkServices()
	if err != nil {
		return err
	}

	portStr := strconv.Itoa(port)
	for _, s := range services {
		// HTTP 代理
		_ = exec.Command("networksetup", "-setwebproxy", s, host, portStr).Run()
		_ = exec.Command("networksetup", "-setwebproxystate", s, "on").Run()
		// HTTPS 代理
		_ = exec.Command("networksetup", "-setsecurewebproxy", s, host, portStr).Run()
		_ = exec.Command("networksetup", "-setsecurewebproxystate", s, "on").Run()
		// SOCKS 代理
		_ = exec.Command("networksetup", "-setsocksfirewallproxy", s, host, portStr).Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxystate", s, "on").Run()

		// 绕过名单
		if len(bypass) > 0 {
			args := append([]string{"-setproxybypassdomains", s}, bypass...)
			_ = exec.Command("networksetup", args...).Run()
		}
	}
	return nil
}

func (m *DarwinManager) Disable() error {
	services, err := m.getActiveNetworkServices()
	if err != nil {
		return err
	}
	for _, s := range services {
		_ = exec.Command("networksetup", "-setwebproxystate", s, "off").Run()
		_ = exec.Command("networksetup", "-setsecurewebproxystate", s, "off").Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxystate", s, "off").Run()
	}
	return nil
}

func (m *DarwinManager) IsEnabled() (bool, error) {
	services, err := m.getActiveNetworkServices()
	if err != nil || len(services) == 0 {
		return false, err
	}
	// 查询首个网络服务状态
	out, err := exec.Command("networksetup", "-getwebproxy", services[0]).Output()
	if err != nil {
		return false, err
	}
	return bytes.Contains(out, []byte("Enabled: Yes")), nil
}

func (m *DarwinManager) GetServer() (string, error) {
	services, err := m.getActiveNetworkServices()
	if err != nil || len(services) == 0 {
		return "", err
	}
	out, err := exec.Command("networksetup", "-getwebproxy", services[0]).Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	var host, port string
	for _, l := range lines {
		if strings.HasPrefix(l, "Server: ") {
			host = strings.TrimPrefix(l, "Server: ")
		}
		if strings.HasPrefix(l, "Port: ") {
			port = strings.TrimPrefix(l, "Port: ")
		}
	}
	if host != "" && port != "" {
		return fmt.Sprintf("%s:%s", host, port), nil
	}
	return "", nil
}
