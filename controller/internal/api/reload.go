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

	"github.com/qualvey/sing-controller/internal/logger"
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
	// 若 Supervisor 正在托管子进程，优先通过 Supervisor 触发 SIGHUP 原生原地热重载（Windows 下自动降级为重启）
	if h.supervisor != nil && h.supervisor.Status().BinaryPath != "" {
		if err := h.supervisor.Reload(context.Background()); err != nil {
			return true, err
		}
		h.waitCoreReady(1500 * time.Millisecond)
		return true, nil
	}

	values := h.settings.Values()
	reload := values.Core.Reload
	if reload == nil || reload.Mode == "none" {
		return false, nil
	}
	if err := h.reloadWithMode(reload, context.Background()); err != nil {
		return true, err
	}
	h.waitCoreReady(1500 * time.Millisecond)
	return true, nil
}

// waitCoreReady 在重载发信后快速轮询等待内核恢复响应，避免客户端瞬时打到重启过程中的断开端口
func (h *Handler) waitCoreReady(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(60 * time.Millisecond)
		status := h.probeCoreHealth(100 * time.Millisecond)
		if status.Alive {
			return true
		}
	}
	return false
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

// reloadIfRequested 保存成功后调用：
// 1. 若请求显式指定 ?reload=false，跳过重载；
// 2. 若请求显式指定 ?reload=true 或 settings 开启了 after_save，执行重载；
// 3. 针对 /api/outbounds 节点类运维接口，默认总是触发内核重载（除非 ?reload=false），确保节点增删改与内核运行时实时一致。
func (h *Handler) reloadIfRequested(r *http.Request) (bool, error) {
	values := h.settings.Values()
	if values.Core.Reload == nil || values.Core.Reload.Mode == "none" {
		return false, nil
	}
	if r != nil && r.URL.Query().Get("reload") == "false" {
		return false, nil
	}
	force := r != nil && r.URL.Query().Get("reload") == "true"
	isOutboundOp := r != nil && strings.HasPrefix(r.URL.Path, "/api/outbounds")
	if !force && !isOutboundOp && !values.Core.Reload.AfterSave {
		return false, nil
	}
	executed, err := h.reloadNow()
	return executed, err
}

// reloadIfEnabled 保存成功后调用：启用 after_save 时执行重载，返回 (是否执行, 错误)。
func (h *Handler) reloadIfEnabled() (bool, error) {
	return h.reloadIfRequested(nil)
}

// detectServiceActionCommand 自动探测可用的 init 管理机制执行指定动作（reload / restart / start / stop / status）
func detectServiceActionCommand(service string, action string) (*exec.Cmd, string, error) {
	// systemd：/run/systemd/system 是 systemd 作为 PID 1 的标准标志（比 systemctl 二进制更可靠）
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		if _, err := exec.LookPath("systemctl"); err == nil {
			return exec.Command("systemctl", action, service), "systemd", nil
		}
	}
	// openrc（Alpine 默认 init）
	if path, err := exec.LookPath("rc-service"); err == nil {
		return exec.Command(path, service, action), "openrc", nil
	}
	// OpenWrt / procd
	if _, err := os.Stat("/etc/openwrt_release"); err == nil {
		initScript := "/etc/init.d/" + service
		if _, err := os.Stat(initScript); err == nil {
			return exec.Command(initScript, action), "openwrt", nil
		}
		if _, err := exec.LookPath("service"); err == nil {
			return exec.Command("service", service, action), "openwrt", nil
		}
		return nil, "", errors.New("OpenWrt 下未找到 /etc/init.d/" + service + "（请先安装 sing-box init 脚本）")
	}
	// 通用 SysV service
	if _, err := os.Stat("/etc/init.d/" + service); err == nil {
		if path, err := exec.LookPath("service"); err == nil {
			return exec.Command(path, service, action), "sysv", nil
		}
	}
	return nil, "", errors.New("未检测到 systemd / openrc(rc-service) / OpenWrt(procd) 等服务管理机制（请安装对应 init 脚本，或在 Settings 配置 pidfile/hook）")
}

// detectReloadCommand 自动探测可用的 init 重载机制（按优先级）：
//  1. systemd：/run/systemd/system 存在（systemd 为 PID 1）→ systemctl reload <service>
//  2. openrc（Alpine 等）：rc-service 在 PATH → rc-service <service> reload
//  3. OpenWrt / procd：/etc/openwrt_release 标记 → /etc/init.d/<service> reload（脚本不存在时退回 service 包装）
//  4. 通用 SysV：/etc/init.d/<service> 存在且 service 命令可用 → service <service> reload
//
// 返回 (命令, 探测到的机制名, 错误)。
func detectReloadCommand(service string) (*exec.Cmd, string, error) {
	return detectServiceActionCommand(service, "reload")
}

// controlService 按 settings 配置或自动探测执行指定动作 (restart / start / stop)。
func (h *Handler) controlService(ctx context.Context, action string) error {
	values := h.settings.Values()
	service := "sing-box"
	if values.Core.Reload != nil && values.Core.Reload.Service != "" {
		service = values.Core.Reload.Service
	}

	mode := "auto"
	if values.Core.Reload != nil && values.Core.Reload.Mode != "" && values.Core.Reload.Mode != "none" {
		mode = values.Core.Reload.Mode
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	switch mode {
	case "auto":
		cmd, detected, err := detectServiceActionCommand(service, action)
		if err != nil {
			logger.Warn("服务控制探测失败: %v", err)
			return err
		}
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = runErr.Error()
			}
			logger.Warn("sing-box %s 失败: detected=%s service=%s error=%s", action, detected, service, detail)
			return fmt.Errorf("%s %s %s 失败: %s", detected, action, service, detail)
		}
		logger.Info("sing-box %s 成功 (mode=auto, detected=%s, service=%s)", action, detected, service)
		return nil

	case "systemd":
		cmd := exec.CommandContext(timeoutCtx, "systemctl", action, service)
		out, err := cmd.CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = err.Error()
			}
			return fmt.Errorf("systemctl %s %s 失败: %s", action, service, detail)
		}
		logger.Info("sing-box %s 成功 (systemd, service=%s)", action, service)
		return nil

	case "hook":
		if values.Core.Reload.Hook == "" {
			return errors.New("hook 模式未配置命令")
		}
		cmd := exec.CommandContext(timeoutCtx, "sh", "-c", values.Core.Reload.Hook)
		out, err := cmd.CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = err.Error()
			}
			return fmt.Errorf("hook 执行失败: %s", detail)
		}
		logger.Info("sing-box %s 成功 (hook, output=%s)", action, strings.TrimSpace(string(out)))
		return nil

	default:
		cmd, detected, err := detectServiceActionCommand(service, action)
		if err != nil {
			return fmt.Errorf("未配置服务控制模式且自动探测失败: %w", err)
		}
		out, runErr := cmd.CombinedOutput()
		if runErr != nil {
			detail := strings.TrimSpace(string(out))
			if detail == "" {
				detail = runErr.Error()
			}
			return fmt.Errorf("%s %s %s 失败: %s", detected, action, service, detail)
		}
		logger.Info("sing-box %s 成功 (detected=%s, service=%s)", action, detected, service)
		return nil
	}
}

// RestartCore 统一重启或热重载内核（支持 Supervisor 托管模式与系统服务 / 热重载降级模式）
func (h *Handler) RestartCore(ctx context.Context) error {
	if h.supervisor != nil {
		return h.supervisor.Restart(ctx)
	}
	if err := h.controlService(ctx, "restart"); err == nil {
		h.waitCoreReady(2 * time.Second)
		return nil
	}
	if _, reloadErr := h.reloadNow(); reloadErr == nil {
		return nil
	}
	return errors.New("内核守护管理器未启用，且未检测到可用的系统服务 (systemd/openrc/init.d)")
}

func (h *Handler) handleReload(w http.ResponseWriter, r *http.Request) {
	logger.Info("收到手动重载 sing-box 请求")
	executed, err := h.reloadNow()
	if err != nil {
		logger.Error("手动重载 sing-box 失败: %v", err)
		writeError(w, http.StatusBadGateway, err)
		return
	}
	if !executed {
		logger.Warn("手动重载已被禁用 (reload.mode 为 none)")
		writeError(w, http.StatusBadRequest, errors.New("重载已禁用（Settings → reload.mode 为 none）"))
		return
	}
	logger.Info("手动重载 sing-box 成功")
	writeJSON(w, http.StatusOK, map[string]any{"reloaded": true})
}

