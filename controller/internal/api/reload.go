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
// 触发方式由 settings.reload 配置：
//   - auto（默认）：自动探测 systemd → openrc(rc-service) → OpenWrt/procd → SysV service
//   - systemd：systemctl reload <service>
//   - pidfile：kill -HUP
//   - hook：自定义命令

// reloadNow 按配置执行一次重载；mode 为 none 时返回 (false, nil)。
func (h *Handler) reloadNow() (bool, error) {
	values := h.settings.Values()
	reload := values.Reload
	if reload == nil || reload.Mode == "none" {
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
	case "auto":
		service := reload.Service
		if service == "" {
			service = "sing-box"
		}
		cmd, detected, err := detectReloadCommand(service)
		if err != nil {
			slog.Warn("sing-box reload failed", "mode", "auto", "error", err)
			return fmt.Errorf("自动检测重载方式失败: %w", err)
		}
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = runErr.Error()
			}
			slog.Warn("sing-box reload failed", "mode", "auto", "detected", detected, "service", service, "error", detail)
			return fmt.Errorf("%s reload %s 失败: %s", detected, service, detail)
		}
		slog.Info("sing-box reloaded", "mode", "auto", "detected", detected, "service", service)
		return nil
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
	if values.Reload == nil || values.Reload.Mode == "none" || !values.Reload.AfterSave {
		return false, nil
	}
	executed, err := h.reloadNow()
	return executed, err
}

// detectReloadCommand 自动探测可用的 init 重载机制（按优先级）：
//  1. systemd：/run/systemd/system 存在（systemd 为 PID 1）→ systemctl reload <service>
//  2. openrc（Alpine 等）：rc-service 在 PATH → rc-service <service> reload
//  3. OpenWrt / procd：/etc/openwrt_release 标记 → /etc/init.d/<service> reload（脚本不存在时退回 service 包装）
//  4. 通用 SysV：/etc/init.d/<service> 存在且 service 命令可用 → service <service> reload
//
// 返回 (命令, 探测到的机制名, 错误)。
func detectReloadCommand(service string) (*exec.Cmd, string, error) {
	// systemd：/run/systemd/system 是 systemd 作为 PID 1 的标准标志（比 systemctl 二进制更可靠）
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		if _, err := exec.LookPath("systemctl"); err == nil {
			return exec.Command("systemctl", "reload", service), "systemd", nil
		}
	}
	// openrc（Alpine 默认 init）
	if path, err := exec.LookPath("rc-service"); err == nil {
		return exec.Command(path, service, "reload"), "openrc", nil
	}
	// OpenWrt / procd
	if _, err := os.Stat("/etc/openwrt_release"); err == nil {
		initScript := "/etc/init.d/" + service
		if _, err := os.Stat(initScript); err == nil {
			return exec.Command(initScript, "reload"), "openwrt", nil
		}
		if _, err := exec.LookPath("service"); err == nil {
			return exec.Command("service", service, "reload"), "openwrt", nil
		}
		return nil, "", errors.New("OpenWrt 下未找到 /etc/init.d/" + service + "（请先安装 sing-box init 脚本）")
	}
	// 通用 SysV service
	if _, err := os.Stat("/etc/init.d/" + service); err == nil {
		if path, err := exec.LookPath("service"); err == nil {
			return exec.Command(path, service, "reload"), "sysv", nil
		}
	}
	return nil, "", errors.New("未检测到 systemd / openrc(rc-service) / OpenWrt(procd) 等重载机制（请安装对应 init 脚本，或在 Settings 配置 pidfile/hook）")
}

func (h *Handler) handleReload(w http.ResponseWriter, r *http.Request) {
	executed, err := h.reloadNow()
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if !executed {
		writeError(w, http.StatusBadRequest, errors.New("重载已禁用（Settings → reload.mode 为 none）"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reloaded": true})
}
