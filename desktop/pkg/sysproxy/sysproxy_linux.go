//go:build linux

package sysproxy

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type LinuxManager struct{}

func New() Manager {
	return &LinuxManager{}
}

func (m *LinuxManager) Enable(host string, port int, bypass []string) error {
	if host == "" {
		host = "127.0.0.1"
	}
	portStr := strconv.Itoa(port)

	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "'manual'").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", fmt.Sprintf("'%s'", host)).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", fmt.Sprintf("'%s'", host)).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", fmt.Sprintf("'%s'", host)).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", portStr).Run()

	if len(bypass) > 0 {
		var formatted []string
		for _, b := range bypass {
			formatted = append(formatted, fmt.Sprintf("'%s'", b))
		}
		listStr := fmt.Sprintf("[%s]", strings.Join(formatted, ", "))
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "ignore-hosts", listStr).Run()
	}

	return nil
}

func (m *LinuxManager) Disable() error {
	return exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "'none'").Run()
}

func (m *LinuxManager) IsEnabled() (bool, error) {
	out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode").Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), "manual"), nil
}

func (m *LinuxManager) GetServer() (string, error) {
	hostOut, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.http", "host").Output()
	if err != nil {
		return "", err
	}
	portOut, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.http", "port").Output()
	if err != nil {
		return "", err
	}
	host := strings.Trim(strings.TrimSpace(string(hostOut)), "'")
	port := strings.TrimSpace(string(portOut))
	if host != "" && port != "" {
		return fmt.Sprintf("%s:%s", host, port), nil
	}
	return "", nil
}
