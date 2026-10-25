package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/qualvey/sing-controller/desktop/pkg/autostart"
	"github.com/qualvey/sing-controller/desktop/pkg/notification"
	"github.com/qualvey/sing-controller/desktop/pkg/sysproxy"
	"github.com/qualvey/sing-controller/internal/engine"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App Wails 桌面端主结构体，绑定前端并管理控制器生命周期
type App struct {
	ctx           context.Context
	engine        *engine.Engine
	sysProxy      sysproxy.Manager
	autoStart     autostart.Manager
	tray          *TrayManager

	proxyPort     int
	sysProxyOn    bool
	mu            sync.Mutex
}

// NewApp 构造 App 实例并在启动时同步拉起 Engine
func NewApp() *App {
	a := &App{
		sysProxy:  sysproxy.New(),
		autoStart: autostart.New(),
		proxyPort: 2080,
	}

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "controller.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "controller.json"
	}

	managed := true
	eng, err := engine.New(engine.Config{
		SettingsPath: configPath,
		ManagedMode:  &managed,
		Version:      version,
		ProvidersDir: filepath.Join(exeDir, "providers"),
		StartServer:  true, // 桌面模式下监听本地端口，供外部 API、gRPC 与 WebSocket 调用
	})
	if err != nil {
		slog.Error("failed to create controller engine", "error", err)
	} else {
		a.engine = eng
		if err := eng.Start(context.Background()); err != nil {
			slog.Error("failed to start controller engine", "error", err)
		}
	}

	return a
}

// startup 由 Wails 在前端就绪时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 1. 初始化检测系统代理状态
	if a.sysProxy != nil {
		enabled, err := a.sysProxy.IsEnabled()
		if err == nil {
			a.sysProxyOn = enabled
		}
	}

	// 2. 启动系统托盘
	a.tray = NewTrayManager(a)
	go a.tray.Run()
}

// shutdown 退出应用时清理资源
func (a *App) shutdown(ctx context.Context) {
	slog.Info("desktop app shutting down...")

	// 1. 若当前开启了系统代理，自动清理恢复直连，防止断网
	if a.sysProxyOn && a.sysProxy != nil {
		_ = a.sysProxy.Disable()
		slog.Info("system proxy automatically cleaned up on exit")
	}

	// 2. 安全关闭 Engine (停内核与所有采集协程)
	if a.engine != nil {
		_ = a.engine.Stop()
	}
}

// handler 返回处理 Wails AssetServer 请求回退与 API 桥接的 http.Handler
func (a *App) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. API 路由与健康检查：透传给 Controller 内部 Handler
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" || r.URL.Path == "/healthz" {
			if a.engine != nil && a.engine.Handler() != nil {
				a.engine.Handler().ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
			return
		}

		// 2. SPA 前端路由回退：当非 API 请求且静态文件未命中时，回退至 index.html
		if indexData, err := assets.ReadFile("frontend/dist/index.html"); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(indexData)
			return
		}

		http.NotFound(w, r)
	})
}

// ======================= 暴露给前端 JS 的 Native 接口 =======================

// DesktopInfo 桌面运行元信息
type DesktopInfo struct {
	IsDesktop bool   `json:"isDesktop"`
	Version   string `json:"version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Port      int    `json:"port"`
}

func (a *App) GetDesktopInfo() DesktopInfo {
	port := 8080
	if a.engine != nil && a.engine.Settings() != nil {
		if p := a.engine.Settings().Values().Server.ListenPort; p > 0 {
			port = int(p)
		}
	}
	return DesktopInfo{
		IsDesktop: true,
		Version:   version,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Port:      port,
	}
}

// GetSystemProxyStatus 获取当前系统代理状态
func (a *App) GetSystemProxyStatus() (bool, error) {
	if a.sysProxy == nil {
		return false, sysproxy.ErrNotImplemented
	}
	enabled, err := a.sysProxy.IsEnabled()
	if err != nil {
		return false, err
	}
	a.mu.Lock()
	a.sysProxyOn = enabled
	a.mu.Unlock()
	return enabled, nil
}

// SetSystemProxy 开启或关闭系统代理
func (a *App) SetSystemProxy(enabled bool) error {
	if a.sysProxy == nil {
		return sysproxy.ErrNotImplemented
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	port := a.proxyPort
	// 尝试从 Store 中获取 mixed/http 入站端口
	if a.engine != nil && a.engine.Store() != nil {
		if data, err := os.ReadFile(a.engine.Store().Path()); err == nil {
			var cfg struct {
				Inbounds []struct {
					Type       string `json:"type"`
					ListenPort int    `json:"listen_port"`
				} `json:"inbounds"`
			}
			if err := json.Unmarshal(data, &cfg); err == nil {
				for _, in := range cfg.Inbounds {
					if in.Type == "mixed" || in.Type == "http" {
						if in.ListenPort > 0 {
							port = in.ListenPort
							break
						}
					}
				}
			}
		}
	}

	if enabled {
		if err := a.sysProxy.Enable("127.0.0.1", port, nil); err != nil {
			return err
		}
		a.sysProxyOn = true
		_ = notification.Send(a.ctx, "系统代理", fmt.Sprintf("系统代理已开启 (127.0.0.1:%d)", port))
	} else {
		if err := a.sysProxy.Disable(); err != nil {
			return err
		}
		a.sysProxyOn = false
		_ = notification.Send(a.ctx, "系统代理", "系统代理已关闭")
	}

	if a.tray != nil {
		a.tray.UpdateSystemProxyState(a.sysProxyOn)
	}

	// 通知前端状态变更
	wailsRuntime.EventsEmit(a.ctx, "desktop:systemProxyChanged", a.sysProxyOn)
	return nil
}

// GetAutoStartStatus 获取开机自启状态
func (a *App) GetAutoStartStatus() (bool, error) {
	if a.autoStart == nil {
		return false, nil
	}
	return a.autoStart.IsEnabled()
}

// SetAutoStart 设置开机自启
func (a *App) SetAutoStart(enabled bool) error {
	if a.autoStart == nil {
		return nil
	}
	if enabled {
		p, err := os.Executable()
		if err != nil {
			return err
		}
		return a.autoStart.Enable(p)
	}
	return a.autoStart.Disable()
}

// SendNotification 发送原生系统通知
func (a *App) SendNotification(title, message string) error {
	return notification.Send(a.ctx, title, message)
}

// RestartCore 重启 sing-box 内核
func (a *App) RestartCore() error {
	if a.engine == nil || a.engine.Supervisor() == nil {
		return fmt.Errorf("supervisor not running")
	}
	return a.engine.Supervisor().Restart(a.ctx)
}

// ShowWindow 显示主窗口
func (a *App) ShowWindow() {
	if a.ctx != nil {
		wailsRuntime.WindowShow(a.ctx)
		wailsRuntime.WindowUnminimise(a.ctx)
	}
}

// HideWindow 隐藏主窗口至托盘
func (a *App) HideWindow() {
	if a.ctx != nil {
		wailsRuntime.WindowHide(a.ctx)
	}
}

// Quit 退出桌面应用
func (a *App) Quit() {
	if a.ctx != nil {
		wailsRuntime.Quit(a.ctx)
	}
}
