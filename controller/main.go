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
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/qualvey/sing-controller/internal/api"
	"github.com/qualvey/sing-controller/internal/collector"
	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/realtime"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
)

// version 由 goreleaser ldflags 注入（-X main.version={{ .Version }}），本地构建为 dev
var version = "dev"

func main() {
	var (
		listenAddr   string
		settingsPath string
		secret       string
		showVersion  bool
	)

	var wg sync.WaitGroup
	flag.StringVar(&listenAddr, "listen", "", "HTTP listen address (override config.json listen)")
	flag.StringVar(&settingsPath, "config", "config.json", "controller config file path")
	flag.StringVar(&secret, "secret", "", "optional API secret (X-Secret header)")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
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

	// 初始化并设置日志级别（支持 trace, debug, info, warn, error 等）
	if values.Log != nil {
		logger.Init(values.Log.Level)
	} else {
		logger.Init("info")
	}

	rt := realtime.NewBroadcaster(values.RealTime.IntervalMS, values.RealTime.OnlineThresholdSec)
	rt.Start()
	wg.Add(1)
	go collector.StartCollector(ctx, &wg, cfg, rt)

	// 监听地址：命令行 -listen 优先，否则取 config.json 的 listen
	if listenAddr == "" {
		listenAddr = values.Listen
	}

	// sing-box 主配置存储（路径来自 settings.config）
	cfgStore := store.New(values.Config)
	defaults := store.DefaultConfig{
		InboundType: values.Defaults.InboundType,
		Listen:      values.Defaults.Listen,
		ListenPort:  values.Defaults.ListenPort,
	}
	if err := cfgStore.Load(ctx, defaults); err != nil {
		slog.Error("load sing-box config", "path", values.Config, "error", err)
		os.Exit(1)

	}

	// 嵌入 webui（web/dist 已构建时）；构建失败仅影响页面，API 照常
	var staticHandler http.Handler
	if builtHandler, err := webHandler(); err == nil {
		staticHandler = builtHandler
	} else {
		slog.Warn("webui static handler init failed, API-only mode", "error", err)
	}
	handler := api.NewHandler(api.HandlerOptions{
		Store:    cfgStore,
		Settings: cfg,
		Secret:   secret,
		Version:  version,
		Static:   staticHandler,
	})
	slog.Info("sing-box-controller started",
		"listen", listenAddr,
		"controller_config", settingsPath,
		"sing_box_config", cfgStore.Path(),
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
	// 5. 等待后台 Collector 执行完手头最后一次写入并安全退出
	slog.Info("waiting for background tasks to exit...")
	wg.Wait()
	rt.Stop()
	slog.Info("controller stopped cleanly")
}
