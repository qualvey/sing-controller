package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/qualvey/sing-controller/internal/settings"
)

// sing-box 重载：向运行中的 sing-box 发送 SIGHUP（官方唯一可靠机制，cmd_run.go:197）。
// 触发方式由 settings.reload 配置：systemd（systemctl reload）/ pidfile（kill -HUP）/ hook（自定义命令）。

// reloadNow 按配置执行一次重载；未配置返回 (false, nil)。
func (h *Handler) reloadNow() (bool, error) {
	values := h.settings.Values()
	reload := values.Reload
	if reload == nil || reload.Mode == "" || reload.Mode == "none" {
		return false, nil
	}
	if err := h.reloadWithMode(reload, context.Background()); err != nil {
		return true, err
	}
	return true, nil
}

// reloadWithMode 按指定模式执行重载。
func (h *Handler) reloadWithMode(reload *settings.ReloadOptions, parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	switch reload.Mode {
	case "systemd":
		service := reload.Service
		if service == "" {
			service = "sing-box"
		}
		cmd := exec.CommandContext(ctx, "systemctl", "reload", service)
		out, err := cmd.CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = err.Error()
			}
			slog.Warn("sing-box reload failed", "mode", "systemd", "service", service, "error", detail)
			return fmt.Errorf("systemctl reload %s 失败: %s", service, detail)
		}
		slog.Info("sing-box reloaded", "mode", "systemd", "service", service)
		return nil
	case "pidfile":
		if reload.PidFile == "" {
			slog.Warn("sing-box reload failed", "mode", "pidfile", "error", "pid_file 未配置")
			return errors.New("pidfile 模式未配置 pid_file")
		}
		content, err := os.ReadFile(reload.PidFile)
		if err != nil {
			slog.Warn("sing-box reload failed", "mode", "pidfile", "pid_file", reload.PidFile, "error", err)
			return fmt.Errorf("读取 pid 文件失败: %w", err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err != nil {
			slog.Warn("sing-box reload failed", "mode", "pidfile", "pid_file", reload.PidFile, "error", fmt.Sprintf("pid 内容无效 %q", strings.TrimSpace(string(content))))
			return fmt.Errorf("pid 文件内容无效: %q", strings.TrimSpace(string(content)))
		}
		if err := killByPid(pid); err != nil {
			slog.Warn("sing-box reload failed", "mode", "pidfile", "pid", pid, "error", err)
			return fmt.Errorf("发送 SIGHUP 到 %d 失败: %w", pid, err)
		}
		slog.Info("sing-box reloaded", "mode", "pidfile", "pid", pid)
		return nil
	case "hook":
		if reload.Hook == "" {
			slog.Warn("sing-box reload failed", "mode", "hook", "error", "hook 命令未配置")
			return errors.New("hook 模式未配置命令")
		}
		cmd := exec.CommandContext(ctx, "sh", "-c", reload.Hook)
		out, err := cmd.CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = err.Error()
			}
			slog.Warn("sing-box reload failed", "mode", "hook", "error", detail)
			return fmt.Errorf("hook 执行失败: %s", detail)
		}
		slog.Info("sing-box reloaded", "mode", "hook", "output", strings.TrimSpace(string(out)))
		return nil
	default:
		slog.Warn("sing-box reload failed", "mode", reload.Mode, "error", "未知模式")
		return fmt.Errorf("未知 reload 模式: %s", reload.Mode)
	}
}

// reloadIfEnabled 保存成功后调用：启用 after_save 时执行重载，返回 (是否执行, 错误)。
func (h *Handler) reloadIfEnabled() (bool, error) {
	values := h.settings.Values()
	if values.Reload == nil || values.Reload.Mode == "" || values.Reload.Mode == "none" || !values.Reload.AfterSave {
		return false, nil
	}
	executed, err := h.reloadNow()
	return executed, err
}

func (h *Handler) handleReload(w http.ResponseWriter, r *http.Request) {
	executed, err := h.reloadNow()
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if !executed {
		writeError(w, http.StatusBadRequest, errors.New("未配置重载方式（Settings → reload.mode：systemd/pidfile/hook）"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reloaded": true})
}
