package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/provider"
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

// handleToolParseNode 解析节点配置文本或分享链接（JSON 或 vless://, vmess://, ss:// 等）：
// 合法 → {ok:true, data:<解析结果>}；非法 → 400 带错误信息。
func (h *Handler) handleToolParseNode(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var request struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		logger.Warn("解析节点工具请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input := strings.TrimSpace(request.Text)
	if input == "" {
		input = strings.TrimSpace(request.Content)
	}
	if input == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("输入内容不能为空"))
		return
	}

	// 1. 如果是 JSON (以 { 开头)
	if strings.HasPrefix(input, "{") {
		var data any
		if err := json.Unmarshal([]byte(input), &data); err != nil {
			logger.Warn("解析用户提供的 JSON 节点失败: %v", err)
			writeError(w, http.StatusBadRequest, fmt.Errorf("JSON 格式错误: %w", err))
			return
		}
		logger.Debug("解析 JSON 节点成功")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": data})
		return
	}

	// 2. 尝试作为代理 URI (vless://, vmess://, ss://, trojan://, tuic://, hy2:// 等) 解析
	sbJSON, err := provider.ParseProxyURIToSingBox(input)
	if err != nil || len(sbJSON) == 0 {
		logger.Warn("解析节点链接失败: %v", err)
		if err == nil {
			err = fmt.Errorf("不支持的节点链接格式或协议")
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var data any
	if err := json.Unmarshal(sbJSON, &data); err != nil {
		logger.Warn("反序列化 sing-box 节点配置失败: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	logger.Debug("解析节点链接成功")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": data})
}

// handlePortAvailable 返回可用端口：GET /api/ports/available?start=NNN
// 未指定 start 时默认从 8000 起始探测。
func (h *Handler) handlePortAvailable(w http.ResponseWriter, r *http.Request) {
	start := uint16(8000)
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

