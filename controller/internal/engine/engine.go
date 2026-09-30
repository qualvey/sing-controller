package engine

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/api"
	"github.com/qualvey/sing-controller/internal/collector"
	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/nodehealth"
	"github.com/qualvey/sing-controller/internal/provider"
	"github.com/qualvey/sing-controller/internal/realtime"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/qualvey/sing-controller/internal/supervisor"
)

// FindSingBoxBinary 自动探测可用 sing-box 二进制文件路径
func FindSingBoxBinary() string {
	candidates := []string{
		"./sing-box.exe",
		"./sing-box",
		"../sing-box.exe",
		"../sing-box",
		"./core_bin/sing-box.exe",
		"../core_bin/sing-box.exe",
		"../core_bin/extracted/sing-box-1.14.0-windows-amd64/sing-box.exe",
	}

	// 1. 优先探测可执行文件自身所在目录
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append([]string{
			filepath.Join(exeDir, "sing-box.exe"),
			filepath.Join(exeDir, "sing-box"),
			filepath.Join(exeDir, "bin", "sing-box.exe"),
			filepath.Join(exeDir, "bin", "sing-box"),
			filepath.Join(exeDir, "core_bin", "sing-box.exe"),
		}, candidates...)
	}

	// 2. Linux / Unix 标准 FHS 路径
	candidates = append(candidates,
		"/usr/bin/sing-box",
		"/usr/local/bin/sing-box",
	)

	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}

	// 3. 环境变量 PATH 查找
	if path, err := exec.LookPath("sing-box"); err == nil {
		return path
	}

	return "sing-box"
}

// Config Engine 运行所需选项
type Config struct {
	SettingsPath  string
	ListenAddr    string
	Secret        string
	ManagedMode   *bool
	Version       string
	StaticHandler  http.Handler
	BinaryPath    string
	ProvidersDir  string
	StartServer   bool // 是否在 Start() 时监听并启动内置 HTTP 服务器
}

// Engine 聚合 Controller 的配置管理、监控采集、内核托管与 API 路由
type Engine struct {
	cfg        *settings.Manager
	store      *store.Store
	supervisor *supervisor.Manager
	provMgr    *provider.Manager
	rt         *realtime.Broadcaster
	handler    http.Handler
	apiHandler *api.Handler
	nodeHealth *nodehealth.Runner
	server     *http.Server

	opts       Config
	listenAddr string
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	running    bool
}

// New 创建 Engine 实例
func New(opts Config) (*Engine, error) {
	if opts.SettingsPath == "" {
		opts.SettingsPath = "controller.json"
	}
	if opts.ProvidersDir == "" {
		opts.ProvidersDir = "providers"
	}

	cfg := settings.New(opts.SettingsPath)
	if err := cfg.Load(); err != nil {
		return nil, fmt.Errorf("load controller config from %s: %w", opts.SettingsPath, err)
	}

	values := cfg.Values()

	// 模式决策
	effectiveManaged := runtime.GOOS == "windows"
	if values.Core.Managed != nil {
		effectiveManaged = *values.Core.Managed
	}
	if opts.ManagedMode != nil {
		effectiveManaged = *opts.ManagedMode
	}

	secret := opts.Secret
	if secret == "" {
		secret = values.Server.Secret
	}

	listenAddr := opts.ListenAddr
	if listenAddr == "" {
		listenAddr = values.Server.Addr()
	}

	// 初始化日志
	if values.Log != nil {
		logger.Init(values.Log.Level, values.Log.Output, values.Log.Outfile)
	} else {
		logger.Init("info", "stderr", "")
	}

	e := &Engine{
		cfg:        cfg,
		opts:       opts,
		listenAddr: listenAddr,
	}

	// 实时广播器
	e.rt = realtime.NewBroadcaster(values.Stats.RealTime.IntervalMS, values.Stats.RealTime.OnlineThresholdSec)

	// 配置存储
	cfgStore := store.New(values.Core.ConfigPath)
	e.store = cfgStore

	// 二进制定位
	binPath := opts.BinaryPath
	if binPath == "" {
		binPath = values.Core.BinaryPath
	}
	if binPath == "" {
		binPath = FindSingBoxBinary()
	}
	store.BinaryPath = binPath

	// Supervisor
	if effectiveManaged {
		workDir := values.Core.WorkingDir
		if workDir == "" {
			workDir = filepath.Dir(cfgStore.Path())
		}
		if runtime.GOOS == "linux" && values.Core.WorkingDir == "" {
			if fi, err := os.Stat("/usr/share/sing-box"); err == nil && fi.IsDir() {
				workDir = "/usr/share/sing-box"
			} else if err := os.MkdirAll("/var/lib/sing-box", 0750); err == nil {
				workDir = "/var/lib/sing-box"
			} else if fi, err := os.Stat("/etc/sing-box"); err == nil && fi.IsDir() {
				workDir = "/etc/sing-box"
			}
		}

		sup, err := supervisor.New(supervisor.Config{
			BinaryPath:  binPath,
			ConfigPath:  cfgStore.Path(),
			WorkingDir:  workDir,
			LogCapacity: 2000,
			AutoRestart: true,
		})
		if err != nil {
			slog.Warn("supervisor init failed", "error", err)
		} else {
			e.supervisor = sup
		}
	}

	// Provider & Profile 管理器
	provMgr, err := provider.NewManager(opts.ProvidersDir, cfgStore, &engineSupervisorAdapter{engine: e})
	if err != nil {
		slog.Warn("provider manager init failed", "error", err)
	} else {
		e.provMgr = provMgr
	}

	// API Handler
	nodeHealthRunner := nodehealth.New(cfg, cfgStore)
	apiH := api.NewHandler(api.HandlerOptions{
		Store:       cfgStore,
		Settings:    cfg,
		Supervisor:  e.supervisor,
		ProviderMgr: e.provMgr,
		Secret:      secret,
		Version:     opts.Version,
		Static:      opts.StaticHandler,
		NodeHealth:  nodeHealthRunner,
	})
	e.apiHandler = apiH
	e.nodeHealth = nodeHealthRunner
	e.handler = apiH

	return e, nil
}

// Start 启动 Engine 各后台工作协程与托管内核
func (e *Engine) Start(parentCtx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running {
		return nil
	}

	ctx, cancel := context.WithCancel(parentCtx)
	e.ctx = ctx
	e.cancel = cancel
	e.running = true

	// 1. 加载默认配置
	defaults := store.DefaultConfig{
		InboundType: "mixed",
		Listen:      "127.0.0.1",
		ListenPort:  2080,
	}
	if err := e.store.Load(ctx, defaults); err != nil {
		slog.Error("load sing-box config", "path", e.cfg.Values().Core.ConfigPath, "error", err)
	}

	// 2. 启动实时状态广播与后台采集
	e.rt.Start()
	e.wg.Add(1)
	go collector.StartCollector(ctx, &e.wg, e.cfg, e.rt)
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.nodeHealth.Run(ctx)
	}()

	// 3. Supervisor 托管状态检查与启动
	if e.supervisor != nil {
		if e.store.Exists() {
			if err := e.supervisor.Start(ctx); err != nil {
				slog.Warn("supervisor auto-start core failed", "error", err)
			}
		} else {
			slog.Info("sing-box config not found, supervisor standing by for initial profile setup", "path", e.store.Path())
		}
	}

	// 4. 冷启动方案自愈恢复
	if e.provMgr != nil && !e.store.Exists() {
		if activeID := e.provMgr.ActiveProfileID(); activeID != "" {
			if err := e.provMgr.ApplyProfile(ctx, activeID); err != nil {
				slog.Debug("auto apply profile on startup", "error", err)
			} else {
				slog.Info("auto-applied active profile on startup", "profile", activeID)
			}
		}
	}

	// 5. 若配置了直接启动 HTTP 服务器
	if e.opts.StartServer {
		e.server = &http.Server{
			Addr:    e.listenAddr,
			Handler: e.handler,
		}
		slog.Info("sing-controller engine HTTP server starting", "addr", e.listenAddr)
		go func() {
			if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("http server failed", "error", err)
			}
		}()
	}

	return nil
}

// Stop 安全停止 Engine
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return nil
	}

	slog.Info("shutting down engine...")
	if e.cancel != nil {
		e.cancel()
	}

	// 停止 HTTP 服务
	if e.server != nil {
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer sCancel()
		if err := e.server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server forced to shutdown", "error", err)
		}
	}

	// 关闭 Supervisor
	if e.supervisor != nil {
		slog.Info("stopping supervised core process...")
		e.supervisor.Close()
	}

	// 等待后台 Collector 退出并停止实时广播
	slog.Info("waiting for background tasks to exit...")
	e.wg.Wait()
	if e.rt != nil {
		e.rt.Stop()
	}

	e.running = false
	slog.Info("engine stopped cleanly")
	return nil
}

// Handler 返回 API http.Handler
func (e *Engine) Handler() http.Handler {
	return e.handler
}

// Store 返回配置存储
func (e *Engine) Store() *store.Store {
	return e.store
}

// Supervisor 返回内核进程管理器
func (e *Engine) Supervisor() *supervisor.Manager {
	return e.supervisor
}

// ProviderMgr 返回订阅与方案管理器
func (e *Engine) ProviderMgr() *provider.Manager {
	return e.provMgr
}

// Settings 返回配置管理
func (e *Engine) Settings() *settings.Manager {
	return e.cfg
}

// Broadcaster 返回实时广播器
func (e *Engine) Broadcaster() *realtime.Broadcaster {
	return e.rt
}

// ListenAddr 返回监听地址
func (e *Engine) ListenAddr() string {
	return e.listenAddr
}

type engineSupervisorAdapter struct {
	engine *Engine
}

func (a *engineSupervisorAdapter) Restart(ctx context.Context) error {
	if a.engine.supervisor != nil {
		return a.engine.supervisor.Restart(ctx)
	}
	if a.engine.apiHandler != nil {
		return a.engine.apiHandler.RestartCore(ctx)
	}
	return nil
}
