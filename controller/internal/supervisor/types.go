package supervisor

import (
	"errors"
	"time"
)

var errReloadNotSupported = errors.New("platform does not support SIGHUP reload")

type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateStopping State = "stopping"
	StateCrashed  State = "crashed"
	StateError    State = "error"
)

type LogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"` // "stdout" | "stderr" | "system"
	Message   string    `json:"message"`
}

type StatusInfo struct {
	State           State      `json:"state"`
	Pid             int        `json:"pid"`
	UptimeSec       int64      `json:"uptime_sec"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	StoppedAt       *time.Time `json:"stopped_at,omitempty"`
	ExitCode        int        `json:"exit_code"`
	LastError       string     `json:"last_error,omitempty"`
	BinaryPath      string     `json:"binary_path"`
	ConfigPath      string     `json:"config_path"`
	Restarts        int        `json:"restarts"`
	TunAccessDenied bool       `json:"tun_access_denied"`
}
