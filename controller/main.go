// sing-box-controller：sing-box 配置管理服务（不运行 sing-box 实例）。
// 职责：读取/校验/生成 sing-box 主配置文件，通过 RESTful API 提供给 webui。
// 自身配置 config.json：{"config": "<主配置路径>", "listen": "127.0.0.1:8080", "log": {"level": "info"}, "min_port": 8000, "defaults": {...}}
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/qualvey/sing-controller/internal/api"
	"github.com/qualvey/sing-controller/internal/collector"
	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/provider"
	"github.com/qualvey/sing-controller/internal/realtime"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/qualvey/sing-controller/internal/supervisor"
)

// version 由 goreleaser ldflags 注入（-X main.version={{ .Version }}），本地构建为 dev
var version = "dev"

func findSingBoxBinary() string {
	candidates := []string{
		"./sing-box.exe",
		"./sing-box",
		"../sing-box.exe",
		"../sing-box",
	}

	// 1. 优先探测 controller 自身二进制所在目录（绿色便携包/独立解压包最佳实践）
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append([]string{
			filepath.Join(exeDir, "sing-box.exe"),
			filepath.Join(exeDir, "sing-box"),
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

func main() {
	var (
		listenAddr   string
		settingsPath string
		secret       string
		showVersion  bool
		managedMode  bool
	)

	defaultManaged := runtime.GOOS == "windows"
	var wg sync.WaitGroup
	flag.StringVar(&listenAddr, "listen", "", "HTTP listen address (override config.json listen)")
	flag.StringVar(&settingsPath, "config", "controller.json", "controller config file path")
	flag.StringVar(&secret, "secret", "", "optional API secret (X-Secret header)")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&managedMode, "managed", defaultManaged, "manage sing-box child process via supervisor (default true on Windows, false on Linux/Unix)")
	flag.Parse()

	// 支持两种形式：-version flag 与 version 子命令（sing-box 风格）
	if showVersion || (flag.NArg() > 0 && flag.Arg(0) == "version") {
		fmt.Printf("sing-controller %s\n", version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// controller 自身配置
	cfg := settings.New(settingsPath)
	if err := cfg.Load(); err != nil {
		slog.Error("load controller config", "path", settingsPath, "error", err)
		os.Exit(1)
	}

	values := cfg.Values()

	// 命令行 Flag 作为最高优先级覆盖项
	effectiveManaged := defaultManaged
	if values.Core.Managed != nil {
		effectiveManaged = *values.Core.Managed
	}
	// 如果用户在命令行显式传了 -managed，以命令行参数为准
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "managed" {
			effectiveManaged = managedMode
		}
		if f.Name == "secret" {
			secret = f.Value.String()
		}
	})

	if secret == "" {
		secret = values.Server.Secret
	}

	// 监听地址：命令行 -listen 优先，否则使用 Server.Addr()
	if listenAddr == "" {
		listenAddr = values.Server.Addr()
	}

	// 初始化并设置日志级别与落盘
	if values.Log != nil {
		logger.Init(values.Log.Level, values.Log.Output, values.Log.Outfile)
	} else {
		logger.Init("info", "stderr", "")
	}

	rt := realtime.NewBroadcaster(values.Stats.RealTime.IntervalMS, values.Stats.RealTime.OnlineThresholdSec)
	rt.Start()
	wg.Add(1)
	go collector.StartCollector(ctx, &wg, cfg, rt)

	// sing-box 主配置存储（路径来自 Core.ConfigPath）
	cfgStore := store.New(values.Core.ConfigPath)
	defaults := store.DefaultConfig{
		InboundType: "mixed",
		Listen:      "127.0.0.1",
		ListenPort:  2080,
	}
	if err := cfgStore.Load(ctx, defaults); err != nil {
		slog.Error("load sing-box config", "path", values.Core.ConfigPath, "error", err)
		os.Exit(1)
	}

	// 根据运行模式初始化 Supervisor
	var sup *supervisor.Manager
	var binaryPath, workDir string

	binaryPath = values.Core.BinaryPath
	if binaryPath == "" {
		binaryPath = findSingBoxBinary()
	}
	// 注入到 store 包供外挂 check 校验使用
	store.BinaryPath = binaryPath

	if effectiveManaged {
		workDir = values.Core.WorkingDir
		if workDir == "" {
			workDir = filepath.Dir(cfgStore.Path())
		}
		if runtime.GOOS == "linux" && values.Core.WorkingDir == "" {
			// Linux / OpenWrt 下优先使用标准状态目录或软路由专用 share 目录（存放 cache.db 等持久化状态）
			if fi, err := os.Stat("/usr/share/sing-box"); err == nil && fi.IsDir() {
				workDir = "/usr/share/sing-box"
			} else if err := os.MkdirAll("/var/lib/sing-box", 0750); err == nil {
				workDir = "/var/lib/sing-box"
			} else if fi, err := os.Stat("/etc/sing-box"); err == nil && fi.IsDir() {
				workDir = "/etc/sing-box"
			}
		}
		var err error
		sup, err = supervisor.New(supervisor.Config{
			BinaryPath:  binaryPath,
			ConfigPath:  cfgStore.Path(),
			WorkingDir:  workDir,
			LogCapacity: 2000,
			AutoRestart: true,
		})
		if err != nil {
			slog.Warn("supervisor init failed", "error", err)
		} else if cfgStore.Exists() {
			// 启动时若配置文件已存在，自动拉起 sing-box 内核
			if err := sup.Start(ctx); err != nil {
				slog.Warn("supervisor auto-start core failed", "error", err)
			}
		} else {
			slog.Info("sing-box config not found, supervisor standing by for initial profile setup", "path", cfgStore.Path())
		}
	} else {
		slog.Info("running in dual-service mode: sing-box core is managed by system service (systemd/procd/openrc)")
	}

	// 初始化 Provider & Profile 管理器
	provMgr, err := provider.NewManager("providers", cfgStore, sup)
	if err != nil {
		slog.Warn("provider manager init failed", "error", err)
	} else if !cfgStore.Exists() {
		// 冷启动自愈：若本地已存有订阅与活动方案，自动合并落盘并拉起内核
		if activeID := provMgr.ActiveProfileID(); activeID != "" {
			if err := provMgr.ApplyProfile(ctx, activeID); err != nil {
				slog.Debug("auto apply profile on startup", "error", err)
			} else {
				slog.Info("auto-applied active profile on startup", "profile", activeID)
			}
		}
	}

	// 嵌入 webui（web/dist 已构建时）；构建失败仅影响页面，API 照常
	var staticHandler http.Handler
	if builtHandler, err := webHandler(); err == nil {
		staticHandler = builtHandler
	} else {
		slog.Warn("webui static handler init failed, API-only mode", "error", err)
	}
	handler := api.NewHandler(api.HandlerOptions{
		Store:       cfgStore,
		Settings:    cfg,
		Supervisor:  sup,
		ProviderMgr: provMgr,
		Secret:      secret,
		Version:     version,
		Static:      staticHandler,
	})
	slog.Info("sing-box-controller started",
		"listen", listenAddr,
		"controller_config", settingsPath,
		"sing_box_config", cfgStore.Path(),
		"mode", map[bool]string{true: "managed-supervisor", false: "dual-service"}[effectiveManaged],
		"log_level", logger.GetLevel().String(),
	)
	srv := &http.Server{Addr: listenAddr, Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	<-ctx.Done()
	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	} else {
		slog.Info("server exited cleanly")
	}

	// 关闭 Supervisor 托管的子进程与 Job Object
	if sup != nil {
		slog.Info("stopping supervised core process...")
		sup.Close()
	}

	// 等待后台 Collector 执行完手头最后一次写入并安全退出
	slog.Info("waiting for background tasks to exit...")
	wg.Wait()
	rt.Stop()
	slog.Info("controller stopped cleanly")
}
