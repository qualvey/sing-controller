package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/store"
)

func (h *Handler) handleToolUUID(w http.ResponseWriter, r *http.Request) {
	id := store.NewUUID()
	logger.Debug("生成 UUID: %s", id)
	writeJSON(w, http.StatusOK, map[string]any{"uuid": id})
}

// handleToolRealityKeypair 生成 Reality X25519 密钥对（与 sing-box generate reality-keypair 一致，URL-safe base64）。
func (h *Handler) handleToolRealityKeypair(w http.ResponseWriter, r *http.Request) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		logger.Error("生成 Reality 密钥对失败: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	publicKey := privateKey.PublicKey()
	logger.Debug("生成 Reality 密钥对成功")
	writeJSON(w, http.StatusOK, map[string]any{
		"private_key": base64.RawURLEncoding.EncodeToString(privateKey.Bytes()),
		"public_key":  base64.RawURLEncoding.EncodeToString(publicKey.Bytes()),
	})
}

// handleToolPassword 生成随机密码：16 字节随机数 → 标准 base64（与 shadowsocks 示例密码格式一致，
// 如 8JCsPssfgS8tiRwiMlhARg==）。新建 shadowsocks 入站时前端默认调用并自动填充。
func (h *Handler) handleToolPassword(w http.ResponseWriter, r *http.Request) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		logger.Error("生成随机密码失败: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Debug("生成随机密码成功")
	writeJSON(w, http.StatusOK, map[string]any{"password": base64.StdEncoding.EncodeToString(key)})
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
		logger.Warn("解析 JSON 工具请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var data any
	if err := json.Unmarshal([]byte(request.JSON), &data); err != nil {
		logger.Warn("解析用户提供的 JSON 文本失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	logger.Debug("解析 JSON 文本成功")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": data})
}

// handlePortAvailable 返回可用端口：GET /api/ports/available?start=NNN
// 未指定 start 时使用 controller 配置的 min_port（默认 8000）。
func (h *Handler) handlePortAvailable(w http.ResponseWriter, r *http.Request) {
	start := h.settings.Values().MinPort
	if raw := r.URL.Query().Get("start"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 16)
		if err != nil || parsed < 1024 {
			logger.Warn("请求可用端口参数非法: start=%s", raw)
			writeError(w, http.StatusBadRequest, strconv.ErrRange)
			return
		}
		start = uint16(parsed)
	}
	port := findAvailablePort(start)
	if port == 0 {
		logger.Error("未找到可用端口: start=%d", start)
		writeError(w, http.StatusInternalServerError, strconv.ErrRange)
		return
	}
	logger.Debug("分配可用端口成功: port=%d (start=%d)", port, start)
	writeJSON(w, http.StatusOK, map[string]any{"port": port, "start": start})
}

