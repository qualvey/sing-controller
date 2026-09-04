//go:build windows

package supervisor

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

type platformData struct {
	job windows.Handle
}

func initPlatform() (*platformData, error) {
	// 在 Windows 桌面端（WebView2 环境）下，避免使用严格 Job Object，防止由于嵌套 Job 冲突引发进程被系统强制终止
	return nil, nil
}

func (p *platformData) bind(cmd *exec.Cmd) error {
	return nil
}

func (p *platformData) close() {
	if p != nil && p.job != 0 {
		_ = windows.CloseHandle(p.job)
		p.job = 0
	}
}

func setupCmdPlatform(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
