package notification

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Send 发送原生系统通知
func Send(ctx context.Context, title, message string) error {
	// 优先触发 Wails 前端事件通知
	if ctx != nil {
		wailsRuntime.EventsEmit(ctx, "desktop:notification", map[string]string{
			"title":   title,
			"message": message,
		})
	}

	switch runtime.GOOS {
	case "windows":
		// Windows PowerShell 气泡/Toast 提示
		psScript := fmt.Sprintf(`
[void] [System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms")
$objNotifyIcon = New-Object System.Windows.Forms.NotifyIcon
$objNotifyIcon.Icon = [System.Drawing.SystemIcons]::Information
$objNotifyIcon.BalloonTipIcon = "Info"
$objNotifyIcon.BalloonTipText = "%s"
$objNotifyIcon.BalloonTipTitle = "%s"
$objNotifyIcon.Visible = $True
$objNotifyIcon.ShowBalloonTip(4000)
Start-Sleep -Seconds 1
$objNotifyIcon.Dispose()
`, strings.ReplaceAll(message, `"`, `\"`), strings.ReplaceAll(title, `"`, `\"`))
		_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript).Start()

	case "darwin":
		script := fmt.Sprintf(`display notification "%s" with title "%s"`,
			strings.ReplaceAll(message, `"`, `\"`),
			strings.ReplaceAll(title, `"`, `\"`))
		_ = exec.Command("osascript", "-e", script).Start()

	case "linux":
		_ = exec.Command("notify-send", title, message).Start()
	}

	return nil
}
