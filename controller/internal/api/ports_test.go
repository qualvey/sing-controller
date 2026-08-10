package api

import (
	"net"
	"strconv"
	"testing"
)

func TestFindAvailablePort(t *testing.T) {
	t.Run("normal range", func(t *testing.T) {
		port := findAvailablePort(30000)
		if port == 0 {
			t.Fatal("no port found")
		}
		if port < 30000 {
			t.Errorf("port %d < start 30000", port)
		}
	})
	t.Run("below 1024 clamped", func(t *testing.T) {
		port := findAvailablePort(80)
		if port == 0 {
			t.Fatal("no port found")
		}
		if port < 1024 {
			t.Errorf("port %d < 1024 (privileged port returned)", port)
		}
	})
	t.Run("found port is bindable", func(t *testing.T) {
		port := findAvailablePort(40000)
		if port == 0 {
			t.Skip("no port available")
		}
		// 返回值是刚释放的端口，立即重新 bind 应成功（轻微竞态可接受）
		if !tcpBindable(t, int(port)) {
			t.Errorf("port %d not bindable", port)
		}
	})
}

func tcpBindable(t *testing.T, port int) bool {
	t.Helper()
	l, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}
