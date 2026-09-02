package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/qualvey/sing-controller/internal/logger"
)

func (h *Handler) handleCoreStatus(w http.ResponseWriter, r *http.Request) {
	if h.supervisor == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"managed": false,
			"state":   "unmanaged",
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
		"binary_path": status.BinaryPath,
		"config_path": status.ConfigPath,
		"restarts":    status.Restarts,
	})
}

func (h *Handler) handleCoreStart(w http.ResponseWriter, r *http.Request) {
	if h.supervisor == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("内核守护管理器未启用"))
		return
	}
	logger.Info("[API] 收到启动内核请求")
	if err := h.supervisor.Start(r.Context()); err != nil {
		logger.Warn("[API] 启动内核失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "内核已启动",
	})
}

func (h *Handler) handleCoreStop(w http.ResponseWriter, r *http.Request) {
	if h.supervisor == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("内核守护管理器未启用"))
		return
	}
	logger.Info("[API] 收到停止内核请求")
	if err := h.supervisor.Stop(r.Context()); err != nil {
		logger.Warn("[API] 停止内核失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "内核已停止",
	})
}

func (h *Handler) handleCoreRestart(w http.ResponseWriter, r *http.Request) {
	if h.supervisor == nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("内核守护管理器未启用"))
		return
	}
	logger.Info("[API] 收到重启内核请求")
	if err := h.supervisor.Restart(r.Context()); err != nil {
		logger.Warn("[API] 重启内核失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "内核已重启",
	})
}

func (h *Handler) handleCoreLogs(w http.ResponseWriter, r *http.Request) {
	if h.supervisor == nil {
		writeJSON(w, http.StatusOK, map[string]any{"logs": []any{}})
		return
	}
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
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
