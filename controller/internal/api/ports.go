package api

import (
	"net"
	"strconv"
)

// findAvailablePort 从 start 起向上查找第一个可 bind 的 TCP 端口（探测后立即释放）。
// start 需 >= 1024；最多探测 1024 个，找不到返回 0。
func findAvailablePort(start uint16) uint16 {
	if start < 1024 {
		start = 1024
	}
	for port := start; port < start+1024; port++ {
		listener, err := net.Listen("tcp", ":"+strconv.Itoa(int(port)))
		if err != nil {
			continue
		}
		_ = listener.Close()
		return port
	}
	return 0
}
