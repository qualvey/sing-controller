package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/qualvey/sing-controller/internal/probe"
)

func (h *Handler) handleListNodeProbes(w http.ResponseWriter, r *http.Request) {
	if h.probe == nil {
		writeError(w, http.StatusServiceUnavailable, probe.ErrUnavailable)
		return
	}
	statuses, err := h.probe.ListStatuses(r.Context())
	if err != nil {
		writeNodeProbeError(w, err)
		return
	}
	tag := strings.TrimSpace(r.URL.Query().Get("tag"))
	if tag != "" {
		filtered := make([]probe.StatusView, 0, 1)
		for _, status := range statuses {
			if status.NodeTag == tag {
				filtered = append(filtered, status)
			}
		}
		statuses = filtered
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": statuses})
}

func (h *Handler) handleGetNodeProbe(w http.ResponseWriter, r *http.Request) {
	if h.probe == nil {
		writeError(w, http.StatusServiceUnavailable, probe.ErrUnavailable)
		return
	}
	tag := strings.TrimSpace(r.PathValue("tag"))
	statuses, err := h.probe.ListStatuses(r.Context())
	if err != nil {
		writeNodeProbeError(w, err)
		return
	}
	for _, status := range statuses {
		if status.NodeTag == tag {
			writeJSON(w, http.StatusOK, status)
			return
		}
	}
	writeError(w, http.StatusNotFound, errors.New("outbound node not found: "+tag))
}

func (h *Handler) handleTriggerNodeProbes(w http.ResponseWriter, r *http.Request) {
	h.triggerNodeProbe(w, r, "")
}

func (h *Handler) handleTriggerNodeProbe(w http.ResponseWriter, r *http.Request) {
	h.triggerNodeProbe(w, r, strings.TrimSpace(r.PathValue("tag")))
}

func (h *Handler) triggerNodeProbe(w http.ResponseWriter, r *http.Request, tag string) {
	if h.probe == nil {
		writeError(w, http.StatusServiceUnavailable, probe.ErrUnavailable)
		return
	}
	var (
		results []probe.Result
		err     error
	)
	if tag == "" {
		results, err = h.probe.Trigger(r.Context(), probe.TriggerManual)
	} else {
		results, err = h.probe.TriggerOne(r.Context(), tag, probe.TriggerManual)
	}
	if err != nil && len(results) == 0 {
		writeNodeProbeError(w, err)
		return
	}
	status := http.StatusAccepted
	if err != nil {
		status = http.StatusBadGateway
	}
	response := map[string]any{"results": results, "count": len(results)}
	if err != nil {
		response["error"] = err.Error()
	}
	writeJSON(w, status, response)
}

func (h *Handler) handleNodeProbeHistory(w http.ResponseWriter, r *http.Request) {
	if h.probe == nil {
		writeError(w, http.StatusServiceUnavailable, probe.ErrUnavailable)
		return
	}
	tag := strings.TrimSpace(r.PathValue("tag"))
	if tag == "" {
		writeError(w, http.StatusBadRequest, errors.New("node tag is required"))
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be a positive integer"))
			return
		}
		limit = parsed
	}
	history, err := h.probe.History(r.Context(), tag, limit)
	if err != nil {
		writeNodeProbeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"node_tag": tag, "history": history})
}

func writeNodeProbeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, probe.ErrUnavailable):
		status = http.StatusServiceUnavailable
	case errors.Is(err, probe.ErrNoTargets), errors.Is(err, probe.ErrNodeNotFound):
		status = http.StatusNotFound
	}
	writeError(w, status, err)
}
