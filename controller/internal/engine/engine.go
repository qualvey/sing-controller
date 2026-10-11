package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/api"
	"github.com/qualvey/sing-controller/internal/collector"
	"github.com/qualvey/sing-controller/internal/db"
	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/nodehealth"
	"github.com/qualvey/sing-controller/internal/probe"
	"github.com/qualvey/sing-controller/internal/provider"
	"github.com/qualvey/sing-controller/internal/realtime"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/qualvey/sing-controller/internal/supervisor"
	"gorm.io/gorm"
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
	StaticHandler http.Handler
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
	probeMgr   *probe.Manager
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

	// 数据库是可选依赖：配置 DSN 后启用节点状态持久化和 Webhook 投递。
	// 环境变量便于容器部署，避免把凭据直接写入配置文件。
	database := initDatabase(values.Database.DSN)

	// 实时广播器
	e.rt = realtime.NewBroadcaster(values.Stats.RealTime.IntervalMS, values.Stats.RealTime.OnlineThresholdSec)

	// 配置存储
	cfgStore := store.New(values.Core.ConfigPath)
	e.store = cfgStore
	e.probeMgr = probe.NewManager(cfg, cfgStore, database)

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
		Probe:       e.probeMgr,
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

// initDatabase 按 DSN 建立数据库连接；DSN 为空或连接失败时返回 nil，
// 表示「数据库不可用」，此时节点探测与流量统计整体停用。
// SING_CONTROLLER_DATABASE_DSN 作为兜底，便于容器部署时注入凭据。
func initDatabase(dsn string) *gorm.DB {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("SING_CONTROLLER_DATABASE_DSN"))
	}
	if dsn == "" {
		return nil
	}
	if err := db.InitDB(dsn); err != nil {
		slog.Warn("database init failed; node probe persistence disabled", "error", err)
		return nil
	}
	database, err := db.Get()
	if err != nil {
		slog.Warn("database unavailable after init", "error", err)
		return nil
	}
	slog.Info("database initialized; node probe persistence enabled")
	return database
}

// bindDatabase 让探测管理器拿到当前数据库连接。
// 已连接时不重复初始化，避免无谓地重连与迁移；
// 未连接时按「配置 DSN → 环境变量」的顺序重连，两处都没有才判定为不可用。
func (e *Engine) bindDatabase() {
	if e.probeMgr == nil {
		return
	}
	if database, err := db.Get(); err == nil {
		e.probeMgr.SetDatabase(database)
		return
	}
	e.probeMgr.SetDatabase(initDatabase(e.cfg.Values().Database.DSN))
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
	if e.provMgr != nil {
		if err := e.provMgr.PrepareActiveProfile(ctx); err != nil {
			slog.Warn("prepare active profile failed", "error", err)
		}
	}

	// 2. 启动实时状态广播、节点探测与后台采集
	e.rt.Start()
	// Stop 会关闭并解绑数据库连接；这里按需重新建立，
	// 否则重启周期内探测与统计会一直对着已关闭的连接池。
	e.bindDatabase()
	if e.probeMgr != nil {
		if err := e.probeMgr.Start(ctx); err != nil && !errors.Is(err, probe.ErrUnavailable) {
			slog.Warn("node probe manager start failed", "error", err)
		}
	}
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

	// 4. 若配置了直接启动 HTTP 服务器
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
	// 先停止节点探测，确保不会在 Supervisor 关闭 sing-box 后继续等待 gRPC。
	if e.probeMgr != nil {
		e.probeMgr.Stop()
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
	if err := db.Close(); err != nil {
		slog.Warn("close database failed", "error", err)
	}
	// 解绑后探测子系统对 API 报 503，而不是继续使用已关闭的连接池。
	if e.probeMgr != nil {
		e.probeMgr.Invalidate()
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
