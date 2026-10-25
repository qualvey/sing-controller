//go:build !windows

package supervisor

import (
	"os"
	"syscall"
)

func signalReload(proc *os.Process) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(syscall.SIGHUP)
}

func signalInterruptGroup(proc *os.Process) error {
	if proc == nil {
		return nil
	}
	// 优先尝试向负 PID 进程组发送 SIGINT，确保整个进程树优雅退出
	if err := syscall.Kill(-proc.Pid, syscall.SIGINT); err == nil {
		return nil
	}
	return proc.Signal(os.Interrupt)
}
