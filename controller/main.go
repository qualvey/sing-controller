// sing-box-controller：sing-box 配置管理服务。
// 职责：读取/校验/生成 sing-box 主配置文件，通过 RESTful API 提供给 webui。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/qualvey/sing-controller/internal/engine"
)

// version 由 goreleaser ldflags 注入（-X main.version={{ .Version }}），本地构建为 dev
var version = "dev"

func main() {
	var (
		listenAddr   string
		settingsPath string
		secret       string
		showVersion  bool
		managedMode  bool
		restartDelay time.Duration
	)

	defaultManaged := runtime.GOOS == "windows"
	flag.StringVar(&listenAddr, "listen", "", "HTTP listen address (override config.json listen)")
	flag.StringVar(&settingsPath, "config", "controller.json", "controller config file path")
	flag.StringVar(&secret, "secret", "", "optional API secret (X-Secret header)")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&managedMode, "managed", defaultManaged, "manage sing-box child process via supervisor (default true on Windows, false on Linux/Unix)")
	flag.DurationVar(&restartDelay, "elevated-restart-delay", 0, "internal delay before starting after UAC elevation")
	flag.Parse()
	if restartDelay > 0 {
		time.Sleep(restartDelay)
	}

	// 支持两种形式：-version flag 与 version 子命令（sing-box 风格）
	if showVersion || (flag.NArg() > 0 && flag.Arg(0) == "version") {
		fmt.Printf("sing-controller %s\n", version)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var managedFlag *bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "managed" {
			managedFlag = &managedMode
		}
	})

	// 嵌入 webui（web/dist 已构建时）；构建失败仅影响页面，API 照常
	var staticHandler http.Handler
	if builtHandler, err := webHandler(); err == nil {
		staticHandler = builtHandler
	} else {
		slog.Warn("webui static handler init failed, API-only mode", "error", err)
	}

	eng, err := engine.New(engine.Config{
		SettingsPath:  settingsPath,
		ListenAddr:    listenAddr,
		Secret:        secret,
		ManagedMode:   managedFlag,
		Version:       version,
		StaticHandler:  staticHandler,
		ProvidersDir:  "providers",
		StartServer:   true,
	})
	if err != nil {
		slog.Error("init controller engine failed", "error", err)
		os.Exit(1)
	}

	if err := eng.Start(ctx); err != nil {
		slog.Error("start controller engine failed", "error", err)
		os.Exit(1)
	}

	// 等待中断信号
	<-ctx.Done()
	slog.Info("shutting down server...")
	if err := eng.Stop(); err != nil {
		slog.Error("stop controller engine error", "error", err)
	}
}
