package autostart

// Manager 开机自启统一接口
type Manager interface {
	Enable(execPath string, args ...string) error
	Disable() error
	IsEnabled() (bool, error)
}
