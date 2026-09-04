//go:build linux

package supervisor

import (
	"os/exec"
	"syscall"
)

type platformData struct{}

func initPlatform() (*platformData, error) {
	return &platformData{}, nil
}

func (p *platformData) bind(cmd *exec.Cmd) error {
	return nil
}

func (p *platformData) close() {}

func setupCmdPlatform(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// 1. 设置独立的进程组
	cmd.SysProcAttr.Setpgid = true
	// 2. 关键防护：当父进程（sing-controller）意外终止或被强杀时，
	//    Linux 内核会自动向子进程（sing-box）投递 SIGKILL，彻底杜绝孤儿进程占用端口与网卡
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
