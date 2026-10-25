package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

// version 由构建脚本 ldflags 注入
var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	logFile := filepath.Join(exeDir, "wails.log")

	_ = os.WriteFile(logFile, []byte(fmt.Sprintf("[%s] Desktop main starting...\n", time.Now().Format(time.RFC3339))), 0644)

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("[%s] PANIC: %v\nStack:\n%s\n", time.Now().Format(time.RFC3339), r, string(debug.Stack()))
			f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
			if f != nil {
				_, _ = f.WriteString(msg)
				_ = f.Close()
			}
		}
	}()

	app := NewApp()

	f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if f != nil {
		_, _ = f.WriteString(fmt.Sprintf("[%s] Calling wails.Run...\n", time.Now().Format(time.RFC3339)))
		_ = f.Close()
	}

	err := wails.Run(&options.App{
		Title:             "sing-box Desktop",
		Width:             1200,
		Height:            800,
		MinWidth:          960,
		MinHeight:         640,
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.handler(),
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
			Theme:                windows.Dark,
		},
	})

	f, _ = os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0644)
	if f != nil {
		_, _ = f.WriteString(fmt.Sprintf("[%s] wails.Run returned! err=%v\n", time.Now().Format(time.RFC3339), err))
		_ = f.Close()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running desktop app: %v\n", err)
		os.Exit(1)
	}
}
