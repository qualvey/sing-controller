package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/qualvey/sing-controller/internal/nodehealth"
	"github.com/qualvey/sing-controller/internal/settings"
)

type nodeHealthConfigRequest struct {
	settings.NodeHealthOptions
	ClearWebhookToken      bool `json:"clear_webhook_token,omitempty"`
	ClearWebhookSignSecret bool `json:"clear_webhook_sign_secret,omitempty"`
}

func (h *Handler) handleNodeHealth(w http.ResponseWriter, _ *http.Request) {
	if h.nodeHealth == nil {
		writeJSON(w, http.StatusOK, nodehealth.Snapshot{Enabled: false, Statuses: map[string]nodehealth.Status{}})
		return
	}
	writeJSON(w, http.StatusOK, h.nodeHealth.Snapshot())
}

func (h *Handler) handleTriggerNodeHealthScan(w http.ResponseWriter, _ *http.Request) {
	if h.nodeHealth == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("node health monitor is unavailable"))
		return
	}
	if !h.nodeHealth.TriggerScan() {
		writeError(w, http.StatusConflict, errors.New("a node health scan is already queued or running"))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true})
}

func (h *Handler) handleGetNodeHealthConfig(w http.ResponseWriter, _ *http.Request) {
	if h.nodeHealth == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("node health monitor is unavailable"))
		return
	}
	options := h.nodeHealth.Config()
	tokenSet := options.WebhookToken != ""
	signSecretSet := options.WebhookSignSecret != ""
	options.WebhookToken = ""
	options.WebhookSignSecret = ""
	writeJSON(w, http.StatusOK, map[string]any{
		"config":                         options,
		"webhook_token_configured":       tokenSet,
		"webhook_sign_secret_configured": signSecretSet,
	})
}

func (h *Handler) handlePutNodeHealthConfig(w http.ResponseWriter, r *http.Request) {
	if h.nodeHealth == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("node health monitor is unavailable"))
		return
	}
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var request nodeHealthConfigRequest
	if err := json.Unmarshal(body, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !request.ClearWebhookToken && request.WebhookToken == "" {
		request.WebhookToken = h.nodeHealth.Config().WebhookToken
	}
	if request.ClearWebhookToken {
		request.WebhookToken = ""
	}
	if !request.ClearWebhookSignSecret && request.WebhookSignSecret == "" {
		request.WebhookSignSecret = h.nodeHealth.Config().WebhookSignSecret
	}
	if request.ClearWebhookSignSecret {
		request.WebhookSignSecret = ""
	}
	if err := h.nodeHealth.UpdateConfig(request.NodeHealthOptions); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": true})
}
