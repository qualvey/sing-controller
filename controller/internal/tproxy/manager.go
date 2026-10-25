// Package tproxy manages the optional host-level TProxy setup hook.
package tproxy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/qualvey/sing-controller/internal/logger"
)

const DefaultScriptPath = "/usr/local/bin/tproxy.sh"

// Config controls the external TProxy lifecycle script.
// The script is deliberately opt-in: most installations do not use TProxy.
type Config struct {
	Enabled bool
	Script  string
}

// Manager invokes the configured script once for each controller lifecycle.
type Manager struct {
	mu         sync.Mutex
	config     Config
	started    bool
	stopCalled bool
}

func New(config Config) *Manager {
	if strings.TrimSpace(config.Script) == "" {
		config.Script = DefaultScriptPath
	}
	return &Manager{config: config}
}

// Start enables host-level TProxy rules. It is a no-op when the feature is
// disabled, so existing non-TProxy installations are unaffected.
func (m *Manager) Start(ctx context.Context) error {
	if m == nil || !m.config.Enabled {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return nil
	}

	if err := m.run(ctx, "start"); err != nil {
		// A script can fail after partially installing routes. Best-effort
		// cleanup prevents leaving the host in a half-configured state.
		if cleanupErr := m.run(context.Background(), "stop"); cleanupErr != nil {
			return fmt.Errorf("启动 TProxy 失败: %w；回滚失败: %v", err, cleanupErr)
		}
		return fmt.Errorf("启动 TProxy 失败: %w", err)
	}
	m.started = true
	m.stopCalled = false
	return nil
}

// Stop disables host-level TProxy rules. It intentionally does not require
// started=true: the controller may have been restarted while the host rules
// from a previous instance are still present.
func (m *Manager) Stop(ctx context.Context) error {
	if m == nil || !m.config.Enabled {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopCalled {
		return nil
	}

	if err := m.run(ctx, "stop"); err != nil {
		return fmt.Errorf("停止 TProxy 失败: %w", err)
	}
	m.started = false
	m.stopCalled = true
	return nil
}

func (m *Manager) run(ctx context.Context, action string) error {
	cmd := exec.CommandContext(ctx, m.config.Script, action)
	output, err := cmd.CombinedOutput()
	if text := strings.TrimSpace(string(output)); text != "" {
		logger.Info("[TProxy] %s: %s", action, text)
	}
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return fmt.Errorf("执行 %s %q: %w", action, m.config.Script, err)
		}
		return fmt.Errorf("执行 %s %q: %w (输出: %s)", action, m.config.Script, err, text)
	}
	return nil
}
