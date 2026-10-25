package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/provider"
)

func (h *Handler) handleListProviders(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeJSON(w, http.StatusOK, map[string]any{"providers": []any{}})
		return
	}
	list := h.providerMgr.ListProviders()
	writeJSON(w, http.StatusOK, map[string]any{"providers": list})
}

func (h *Handler) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusNotFound, errors.New("provider manager not available"))
		return
	}
	id := r.PathValue("id")
	p, err := h.providerMgr.GetProvider(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) handleCreateProvider(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	var p provider.Provider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("解析请求失败: %w", err))
		return
	}
	created, err := h.providerMgr.AddProvider(r.Context(), p)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	id := r.PathValue("id")
	var p provider.Provider
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("解析请求失败: %w", err))
		return
	}
	p.ID = id
	if err := h.providerMgr.UpdateProvider(r.Context(), p); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handler) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	id := r.PathValue("id")
	if err := h.providerMgr.DeleteProvider(id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handler) handleRefreshProvider(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	id := r.PathValue("id")
	logger.Info("[API] 收到手动刷新订阅请求: %s", id)
	outbounds, err := h.providerMgr.RefreshProvider(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("刷新订阅失败: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"node_count": len(outbounds),
	})
}

func (h *Handler) handleGetProviderNodes(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	tag := r.PathValue("tag")
	outbounds, err := h.providerMgr.GetProviderOutbounds(tag)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("未找到该订阅节点片段: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tag":       tag,
		"count":     len(outbounds),
		"outbounds": outbounds,
	})
}

// Profiles API

func (h *Handler) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeJSON(w, http.StatusOK, map[string]any{"profiles": []any{}})
		return
	}
	list := h.providerMgr.ListProfiles()
	writeJSON(w, http.StatusOK, map[string]any{"profiles": list})
}

func (h *Handler) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusNotFound, errors.New("provider manager not available"))
		return
	}
	id := r.PathValue("id")
	p, err := h.providerMgr.GetProfile(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	var p provider.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("解析请求失败: %w", err))
		return
	}
	created, err := h.providerMgr.AddProfile(p)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	id := r.PathValue("id")
	var p provider.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("解析请求失败: %w", err))
		return
	}
	p.ID = id
	if err := h.providerMgr.UpdateProfile(p); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handler) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	id := r.PathValue("id")
	if err := h.providerMgr.DeleteProfile(id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handler) handleApplyProfile(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusBadRequest, errors.New("provider manager not available"))
		return
	}
	id := r.PathValue("id")
	logger.Info("[API] 收到应用配置组方案请求: %s", id)
	if err := h.providerMgr.ApplyProfile(r.Context(), id); err != nil {
		logger.Warn("[API] 应用配置组方案失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 双服务模式（Linux/OpenWrt）：若未托管内核，保存方案后通过系统机制（systemd/procd）重载 sing-box
	if h.supervisor == nil {
		if reloaded, err := h.reloadIfEnabled(); err != nil {
			logger.Warn("[API] 应用方案后系统重载失败: %v", err)
		} else if reloaded {
			logger.Info("[API] 应用方案后系统服务已触发重载")
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "方案已应用并生效",
	})
}
