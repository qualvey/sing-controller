package supervisor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
)

type Config struct {
	BinaryPath  string
	ConfigPath  string
	WorkingDir  string
	LogCapacity int
	AutoRestart bool
}

type Manager struct {
	mu          sync.RWMutex
	cfg         Config
	state       State
	cmd         *exec.Cmd
	waitDone    chan struct{}
	startedAt   *time.Time
	stoppedAt   *time.Time
	exitCode    int
	lastError   string
	restarts    int
	recentExits []time.Time

	platform *platformData
	logs     *RingBuffer

	subMu       sync.RWMutex
	subscribers map[chan LogLine]struct{}

	cancelFn context.CancelFunc
}

func New(cfg Config) (*Manager, error) {
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = "sing-box"
	}
	if cfg.LogCapacity <= 0 {
		cfg.LogCapacity = 2000
	}

	pData, err := initPlatform()
	if err != nil {
		logger.Warn("[Supervisor] init platform job object failed: %v", err)
	}

	return &Manager{
		cfg:         cfg,
		state:       StateStopped,
		platform:    pData,
		logs:        NewRingBuffer(cfg.LogCapacity),
		subscribers: make(map[chan LogLine]struct{}),
	}, nil
}

func (m *Manager) Status() StatusInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var uptime int64
	var pid int
	if m.state == StateRunning && m.startedAt != nil {
		uptime = int64(time.Since(*m.startedAt).Seconds())
	}
	if m.cmd != nil && m.cmd.Process != nil && m.state == StateRunning {
		pid = m.cmd.Process.Pid
	}

	return StatusInfo{
		State:      m.state,
		Pid:        pid,
		UptimeSec:  uptime,
		StartedAt:  m.startedAt,
		StoppedAt:  m.stoppedAt,
		ExitCode:   m.exitCode,
		LastError:  m.lastError,
		BinaryPath: m.cfg.BinaryPath,
		ConfigPath: m.cfg.ConfigPath,
		Restarts:   m.restarts,
	}
}

func (m *Manager) Logs(limit int) []LogLine {
	return m.logs.Slice(limit)
}

func (m *Manager) Subscribe() (chan LogLine, func()) {
	ch := make(chan LogLine, 256)
	m.subMu.Lock()
	m.subscribers[ch] = struct{}{}
	m.subMu.Unlock()

	unsubscribe := func() {
		m.subMu.Lock()
		delete(m.subscribers, ch)
		close(ch)
		m.subMu.Unlock()
	}

	return ch, unsubscribe
}

func (m *Manager) broadcast(line LogLine) {
	m.logs.Push(line)

	m.subMu.RLock()
	defer m.subMu.RUnlock()

	for ch := range m.subscribers {
		select {
		case ch <- line:
		default:
			// 消费者阻塞时跳过，避免阻塞日志流
		}
	}
}

func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateRunning || m.state == StateStarting {
		return errors.New("内核已在运行中")
	}

	return m.startLocked(ctx)
}

func (m *Manager) startLocked(ctx context.Context) error {
	m.state = StateStarting

	args := []string{"run"}
	if m.cfg.ConfigPath != "" {
		args = append(args, "-c", m.cfg.ConfigPath)
	}

	cmd := exec.Command(m.cfg.BinaryPath, args...)
	if m.cfg.WorkingDir != "" {
		cmd.Dir = m.cfg.WorkingDir
	}
	setupCmdPlatform(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.state = StateError
		m.lastError = fmt.Sprintf("stdout pipe error: %v", err)
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.state = StateError
		m.lastError = fmt.Sprintf("stderr pipe error: %v", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		m.state = StateError
		m.lastError = fmt.Sprintf("启动内核失败: %v", err)
		m.broadcast(LogLine{
			Timestamp: time.Now(),
			Source:    "system",
			Message:   fmt.Sprintf("[Supervisor] 启动内核失败 (%s): %v", m.cfg.BinaryPath, err),
		})
		return err
	}

	if m.platform != nil {
		if err := m.platform.bind(cmd); err != nil {
			logger.Warn("[Supervisor] 将进程绑定至 Job Object 失败: %v", err)
		}
	}

	now := time.Now()
	waitDone := make(chan struct{})
	m.waitDone = waitDone
	m.cmd = cmd
	m.startedAt = &now
	m.state = StateRunning
	m.lastError = ""

	m.broadcast(LogLine{
		Timestamp: now,
		Source:    "system",
		Message:   fmt.Sprintf("[Supervisor] sing-box 内核已启动 (PID: %d)", cmd.Process.Pid),
	})

	go m.streamLogs(stdout, "stdout")
	go m.streamLogs(stderr, "stderr")
	go m.waitProcess(cmd, waitDone)

	return nil
}

func (m *Manager) streamLogs(r io.Reader, source string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		m.broadcast(LogLine{
			Timestamp: time.Now(),
			Source:    source,
			Message:   text,
		})
	}
}

func (m *Manager) waitProcess(cmd *exec.Cmd, waitDone chan struct{}) {
	err := cmd.Wait()
	close(waitDone)

	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果当前 cmd 已被替换，忽略旧进程的退出事件
	if m.cmd != cmd {
		return
	}

	now := time.Now()
	m.stoppedAt = &now
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	m.exitCode = exitCode

	// 判断是否为正常主动停止
	if m.state == StateStopping || m.state == StateStopped {
		m.state = StateStopped
		m.broadcast(LogLine{
			Timestamp: now,
			Source:    "system",
			Message:   fmt.Sprintf("[Supervisor] sing-box 内核已正常停止 (ExitCode: %d)", exitCode),
		})
		return
	}

	// 异常退出 / Crash 诊断
	m.state = StateCrashed
	m.lastError = fmt.Sprintf("内核异常退出 (ExitCode: %d)", exitCode)
	m.broadcast(LogLine{
		Timestamp: now,
		Source:    "system",
		Message:   fmt.Sprintf("[Supervisor] ⚠️ 警告: sing-box 内核异常退出! (ExitCode: %d)", exitCode),
	})

	// 生成 Crash Dump 文件
	m.generateCrashDump(exitCode)

	// 连续崩溃防抖检测（10秒内连续退出超过 3 次）
	m.recentExits = append(m.recentExits, now)
	m.cleanRecentExits(now)

	if len(m.recentExits) >= 3 {
		m.state = StateError
		m.lastError = "内核在短时间内频繁崩溃，已暂停自动拉起以防止系统雪崩"
		m.broadcast(LogLine{
			Timestamp: now,
			Source:    "system",
			Message:   "[Supervisor] 🛑 内核在 10 秒内连续崩溃 3 次，熔断暂停自动重启",
		})
		return
	}

	// 若开启了自动重启
	if m.cfg.AutoRestart {
		m.restarts++
		go func() {
			time.Sleep(1 * time.Second)
			m.mu.Lock()
			defer m.mu.Unlock()
			if m.state == StateCrashed {
				logger.Info("[Supervisor] 尝试自动重新拉起内核 (第 %d 次)", m.restarts)
				_ = m.startLocked(context.Background())
			}
		}()
	}
}

func (m *Manager) cleanRecentExits(now time.Time) {
	threshold := now.Add(-10 * time.Second)
	var kept []time.Time
	for _, t := range m.recentExits {
		if t.After(threshold) {
			kept = append(kept, t)
		}
	}
	m.recentExits = kept
}

func (m *Manager) generateCrashDump(exitCode int) {
	recentLogs := m.logs.Slice(100)
	if len(recentLogs) == 0 {
		return
	}

	crashDir := "logs"
	_ = os.MkdirAll(crashDir, 0755)

	dumpFileName := filepath.Join(crashDir, fmt.Sprintf("crash-%s.log", time.Now().Format("20060102-150405")))
	var b strings.Builder
	b.WriteString(fmt.Sprintf("=== SING-BOX CRASH DUMP ===\nTimestamp: %s\nExitCode: %d\nBinary: %s\nConfig: %s\n\n--- RECENT LOGS ---\n",
		time.Now().Format(time.RFC3339), exitCode, m.cfg.BinaryPath, m.cfg.ConfigPath))

	for _, l := range recentLogs {
		b.WriteString(fmt.Sprintf("[%s] [%s] %s\n", l.Timestamp.Format("15:04:05.000"), l.Source, l.Message))
	}

	_ = os.WriteFile(dumpFileName, []byte(b.String()), 0644)
	logger.Warn("[Supervisor] 已生成崩溃诊断日志: %s", dumpFileName)
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if m.state == StateStopped || m.cmd == nil || m.cmd.Process == nil {
		m.state = StateStopped
		m.mu.Unlock()
		return nil
	}

	m.state = StateStopping
	proc := m.cmd.Process
	waitDone := m.waitDone
	m.broadcast(LogLine{
		Timestamp: time.Now(),
		Source:    "system",
		Message:   "[Supervisor] 正在停止 sing-box 内核...",
	})
	m.mu.Unlock()

	// 优雅终止：发送中断信号
	_ = proc.Signal(os.Interrupt)

	// 等待 waitProcess 退出并关闭 waitDone，超时强制杀死
	if waitDone != nil {
		select {
		case <-waitDone:
		case <-time.After(3 * time.Second):
			_ = proc.Kill()
			select {
			case <-waitDone:
			case <-time.After(1 * time.Second):
			}
		case <-ctx.Done():
			_ = proc.Kill()
			return ctx.Err()
		}
	} else {
		_ = proc.Kill()
	}

	m.mu.Lock()
	m.state = StateStopped
	m.mu.Unlock()
	return nil
}

func (m *Manager) Restart(ctx context.Context) error {
	_ = m.Stop(ctx)
	time.Sleep(200 * time.Millisecond)
	return m.Start(ctx)
}

func (m *Manager) Close() {
	_ = m.Stop(context.Background())
	if m.platform != nil {
		m.platform.close()
	}
}
