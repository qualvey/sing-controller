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

// handleToolRealityKeypair 生成 Reality X25519 密钥对（与 sing-box generate reality-keypair 一致，URL-safe base64）。
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

// handleToolParseJSON 解析任意 JSON 文本（前端"粘贴 JSON 解析字段"用）：
// 合法 → {ok:true, data:<解析结果>}；非法 → 400 带错误信息。
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

// handlePortAvailable 返回可用端口：GET /api/ports/available?start=NNN
// 未指定 start 时使用 controller 配置的 min_port（默认 8000）。
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
