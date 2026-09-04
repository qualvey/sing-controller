//go:build windows

package supervisor

import (
	"os"
)

func signalReload(proc *os.Process) error {
	return errReloadNotSupported
}

func signalInterruptGroup(proc *os.Process) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(os.Interrupt)
}
