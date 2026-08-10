package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/qualvey/sing-controller/internal/store"
)

func (h *Handler) handleToolUUID(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"uuid": store.NewUUID()})
}

// handleToolRealityKeypair ç”Ÿæˆ Reality X25519 å¯†é’¥å¯¹ï¼ˆä¸Ž sing-box generate reality-keypair ä¸€è‡´ï¼ŒURL-safe base64ï¼‰ã€‚
func (h *Handler) handleToolRealityKeypair(w http.ResponseWriter, r *http.Request) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	publicKey := privateKey.PublicKey()
	writeJSON(w, http.StatusOK, map[string]any{
		"private_key": base64.RawURLEncoding.EncodeToString(privateKey.Bytes()),
		"public_key":  base64.RawURLEncoding.EncodeToString(publicKey.Bytes()),
	})
}

// handleToolParseJSON è§£æžä»»æ„ JSON æ–‡æœ¬ï¼ˆå‰ç«¯"ç²˜è´´ JSON è§£æžå­—æ®µ"ç”¨ï¼‰ï¼š
// åˆæ³• â†’ {ok:true, data:<è§£æžç»“æžœ>}ï¼›éžæ³• â†’ 400 å¸¦é”™è¯¯ä¿¡æ¯ã€‚
func (h *Handler) handleToolParseJSON(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var request struct {
		JSON string `json:"json"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var data any
	if err := json.Unmarshal([]byte(request.JSON), &data); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": data})
}

// handlePortAvailable è¿”å›žå¯ç”¨ç«¯å£ï¼šGET /api/ports/available?start=NNN
// æœªæŒ‡å®š start æ—¶ä½¿ç”¨ controller é…ç½®çš„ min_portï¼ˆé»˜è®¤ 8000ï¼‰ã€‚
func (h *Handler) handlePortAvailable(w http.ResponseWriter, r *http.Request) {
	start := h.settings.Values().MinPort
	if raw := r.URL.Query().Get("start"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 16)
		if err != nil || parsed < 1024 {
			writeError(w, http.StatusBadRequest, strconv.ErrRange)
			return
		}
		start = uint16(parsed)
	}
	port := findAvailablePort(start)
	if port == 0 {
		writeError(w, http.StatusInternalServerError, strconv.ErrRange)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"port": port, "start": start})
}
