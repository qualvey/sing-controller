//go:build windows

package supervisor

import (
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

type platformData struct {
	job windows.Handle
}

func initPlatform() (*platformData, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create job object: %w", err)
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}

	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("set job object info: %w", err)
	}

	return &platformData{job: job}, nil
}

func (p *platformData) bind(cmd *exec.Cmd) error {
	if p == nil || p.job == 0 || cmd == nil || cmd.Process == nil {
		return nil
	}
	handle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("open process: %w", err)
	}
	defer windows.CloseHandle(handle)

	if err := windows.AssignProcessToJobObject(p.job, handle); err != nil {
		return fmt.Errorf("assign process to job object: %w", err)
	}
	return nil
}

func (p *platformData) close() {
	if p != nil && p.job != 0 {
		_ = windows.CloseHandle(p.job)
		p.job = 0
	}
}

func setupCmdPlatform(cmd *exec.Cmd) {
	// Windows下不需额外设pgid
}
