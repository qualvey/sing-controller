//go:build !linux

package api

import "errors"

// killByPid 仅 Linux 支持（SIGHUP 语义）；其他平台返回明确错误。
func killByPid(pid int) error {
	return errors.New("pidfile 重载模式仅支持 Linux")
}
