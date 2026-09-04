package main

import (
	_ "embed"
	"log/slog"
	"runtime"
	"sync"

	"github.com/getlantern/systray"
)

//go:embed build/appicon.png
var trayIconPng []byte

//go:embed build/windows/icon.ico
var trayIconIco []byte

// TrayManager 托盘管理器
type TrayManager struct {
	app       *App
	mSysProxy *systray.MenuItem
	mStatus   *systray.MenuItem
	mu        sync.Mutex
}

// NewTrayManager 构造托盘管理器
func NewTrayManager(app *App) *TrayManager {
	return &TrayManager{app: app}
}

// Run 启动托盘消息循环
func (t *TrayManager) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *TrayManager) onReady() {
	if runtime.GOOS == "windows" {
		systray.SetIcon(trayIconIco)
	} else {
		systray.SetIcon(trayIconPng)
	}
	systray.SetTitle("sing-box")
	systray.SetTooltip("sing-box Desktop Client")

	// 1. 状态展示
	t.mStatus = systray.AddMenuItem("sing-box: 运行中", "核心当前运行状态")
	t.mStatus.Disable()

	systray.AddSeparator()

	// 2. 系统代理开关
	t.mSysProxy = systray.AddMenuItemCheckbox("系统代理 (System Proxy)", "快速开启或关闭系统代理", t.app.sysProxyOn)

	systray.AddSeparator()

	// 3. 打开主界面
	mShow := systray.AddMenuItem("打开控制面板", "显示主窗口")

	// 4. 重启内核
	mRestart := systray.AddMenuItem("重启内核 (Restart Core)", "重新启动 sing-box 内核进程")

	systray.AddSeparator()

	// 5. 退出
	mQuit := systray.AddMenuItem("退出 (Quit)", "完全退出桌面端应用")

	// 事件监听
	go func() {
		for {
			select {
			case _, ok := <-t.mSysProxy.ClickedCh:
				if !ok {
					return
				}
				t.mu.Lock()
				target := !t.app.sysProxyOn
				t.mu.Unlock()
				if err := t.app.SetSystemProxy(target); err != nil {
					slog.Error("tray toggle system proxy failed", "error", err)
				}

			case _, ok := <-mShow.ClickedCh:
				if !ok {
					return
				}
				t.app.ShowWindow()

			case _, ok := <-mRestart.ClickedCh:
				if !ok {
					return
				}
				if err := t.app.RestartCore(); err != nil {
					slog.Error("tray restart core failed", "error", err)
				}

			case _, ok := <-mQuit.ClickedCh:
				if !ok {
					return
				}
				systray.Quit()
				t.app.Quit()
				return
			}
		}
	}()
}

func (t *TrayManager) onExit() {
	// 清理托盘资源
}

// UpdateSystemProxyState 更新托盘勾选状态
func (t *TrayManager) UpdateSystemProxyState(enabled bool) {
	if t.mSysProxy == nil {
		return
	}
	if enabled {
		t.mSysProxy.Check()
	} else {
		t.mSysProxy.Uncheck()
	}
}
