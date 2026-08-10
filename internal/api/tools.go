package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/sagernet/sing-box-webui/internal/store"
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
