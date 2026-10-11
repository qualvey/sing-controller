package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
)

func (h *Handler) handleCoreStatus(w http.ResponseWriter, r *http.Request) {
	if h.supervisor == nil {
		probe := h.probeCoreHealth(200 * time.Millisecond)
		state := "stopped"
		if probe.Alive {
			state = "running"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"managed": false,
			"state":   state,
			"alive":   probe.Alive,
			"version": probe.Version,
			"type":    probe.Type,
			"error":   probe.Error,
		})
		return
	}
	status := h.supervisor.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"managed":     true,
		"state":       status.State,
		"pid":         status.Pid,
		"uptime_sec":  status.UptimeSec,
		"started_at":  status.StartedAt,
		"stopped_at":  status.StoppedAt,
		"exit_code":   status.ExitCode,
		"last_error":  status.LastError,
		"fatal_error": status.FatalError,
		"binary_path": status.BinaryPath,
		"config_path": status.ConfigPath,
		"restarts":    status.Restarts,
	})
}

func (h *Handler) handleCoreStart(w http.ResponseWriter, r *http.Request) {
	if h.supervisor != nil {
		logger.Info("[API] 收到启动内核请求 (Supervisor 托管模式)")
		if err := h.supervisor.Start(r.Context()); err != nil {
			logger.Warn("[API] 启动内核失败: %v", err)
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "内核已启动",
		})
		return
	}

	logger.Info("[API] 收到启动内核请求 (系统服务模式)")
	if err := h.controlService(r.Context(), "start"); err != nil {
		logger.Warn("[API] 启动系统服务失败: %v", err)
		writeError(w, http.StatusBadRequest, fmt.Errorf("内核守护管理器未启用，且启动系统服务失败: %w", err))
		return
	}

	h.waitCoreReady(2 * time.Second)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "内核服务已启动",
	})
}

func (h *Handler) handleCoreStop(w http.ResponseWriter, r *http.Request) {
	if h.supervisor != nil {
		logger.Info("[API] 收到停止内核请求 (Supervisor 托管模式)")
		if err := h.supervisor.Stop(r.Context()); err != nil {
			logger.Warn("[API] 停止内核失败: %v", err)
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "内核已停止",
		})
		return
	}

	logger.Info("[API] 收到停止内核请求 (系统服务模式)")
	if err := h.controlService(r.Context(), "stop"); err != nil {
		logger.Warn("[API] 停止系统服务失败: %v", err)
		writeError(w, http.StatusBadRequest, fmt.Errorf("内核守护管理器未启用，且停止系统服务失败: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "内核服务已停止",
	})
}

func (h *Handler) handleCoreRestart(w http.ResponseWriter, r *http.Request) {
	if h.supervisor != nil {
		logger.Info("[API] 收到重启内核请求 (Supervisor 托管模式)")
		if err := h.supervisor.Restart(r.Context()); err != nil {
			logger.Warn("[API] 重启内核失败: %v", err)
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "内核已重启",
		})
		return
	}

	logger.Info("[API] 收到重启内核请求 (系统服务模式)")
	if err := h.controlService(r.Context(), "restart"); err != nil {
		// 若系统服务不可用，尝试触发热重载 (reloadNow) 兜底
		if _, reloadErr := h.reloadNow(); reloadErr == nil {
			logger.Info("[API] 系统服务重启不可用，已通过热重载 sing-box 配置生效")
			writeJSON(w, http.StatusOK, map[string]any{
				"success": true,
				"message": "内核配置已热重载 (非守护进程模式)",
			})
			return
		}
		logger.Warn("[API] 重启内核失败: %v", err)
		writeError(w, http.StatusBadRequest, fmt.Errorf("内核守护管理器未启用，且系统服务重启失败: %w", err))
		return
	}

	h.waitCoreReady(2 * time.Second)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "内核服务已重启",
	})
}

func (h *Handler) handleCoreLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if h.supervisor == nil {
		// 尝试从 journalctl 读取 systemd 服务日志
		if _, err := os.Stat("/run/systemd/system"); err == nil {
			if _, err := exec.LookPath("journalctl"); err == nil {
				service := "sing-box"
				if h.settings.Values().Core.Reload != nil && h.settings.Values().Core.Reload.Service != "" {
					service = h.settings.Values().Core.Reload.Service
				}
				out, err := exec.Command("journalctl", "-u", service, "-n", strconv.Itoa(limit), "--no-pager", "-o", "cat").CombinedOutput()
				if err == nil {
					rawLines := strings.Split(strings.TrimSpace(string(out)), "\n")
					lines := make([]any, 0, len(rawLines))
					for _, line := range rawLines {
						if strings.TrimSpace(line) != "" {
							lines = append(lines, map[string]any{"raw": line, "message": line})
						}
					}
					writeJSON(w, http.StatusOK, map[string]any{"logs": lines, "count": len(lines)})
					return
				}
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"logs": []any{}})
		return
	}
	logs := h.supervisor.Logs(limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"logs":  logs,
		"count": len(logs),
	})
}

func (h *Handler) handleCoreLogsStream(w http.ResponseWriter, r *http.Request) {
	if h.supervisor == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("内核守护管理器未启用"))
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("不支持流式响应"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	logChan, unsubscribe := h.supervisor.Subscribe()
	defer unsubscribe()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-logChan:
			if !ok {
				return
			}
			data, err := json.Marshal(line)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
