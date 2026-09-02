//go:build !windows

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
	cmd.SysProcAttr.Setpgid = true
}
