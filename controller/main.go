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

	"github.com/qualvey/sing-controller/internal/api"
	"github.com/qualvey/sing-controller/internal/collector"
	"github.com/qualvey/sing-controller/internal/realtime"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing-box/log"
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

	ctx := context.Background()

	// controller 自身配置
	cfg := settings.New(settingsPath)
	if err := cfg.Load(); err != nil {
		slog.Error("load controller config", "path", settingsPath, "error", err)
		return
	}

	values := cfg.Values()

	rt := realtime.NewBroadcaster(cfg.RealTime.IntervalMS, cfg.RealTime.OnlineThresholdSec)
	rt.Start()
	defer rt.Stop()
	go collector.StartCollector(cfg, rt)
	// 日志级别（复用 sing-box 的 level 枚举/解析）
	setLogLevel(values.Log.Level)

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
		return
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
		"log_level", values.Log.Level,
	)
	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		slog.Error("http server", "error", err)
	}
}

// setLogLevel 将 sing-box 的日志级别映射到 slog 并设置全局级别。
func setLogLevel(levelText string) {
	level, err := log.ParseLevel(levelText)
	if err != nil {
		slog.Warn("unknown log level, fallback to info", "level", levelText)
		level = log.LevelInfo
	}
	var slogLevel slog.Level
	switch level {
	case log.LevelTrace, log.LevelDebug:
		slogLevel = slog.LevelDebug
	case log.LevelInfo:
		slogLevel = slog.LevelInfo
	case log.LevelWarn:
		slogLevel = slog.LevelWarn
	case log.LevelError, log.LevelFatal, log.LevelPanic:
		slogLevel = slog.LevelError
	}
	slog.SetLogLoggerLevel(slogLevel)
}
