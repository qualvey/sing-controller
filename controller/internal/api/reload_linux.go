//go:build linux

package api

import "syscall"

// killByPid 向进程发送 SIGHUP（sing-box 收到后重载配置）。
func killByPid(pid int) error {
	return syscall.Kill(pid, syscall.SIGHUP)
}
