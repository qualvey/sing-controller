package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
)

func newTestHandlerWithInbounds(t *testing.T, inboundsJSON string) http.Handler {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{"log":{"level":"info"},"inbounds":[` + inboundsJSON + `],"outbounds":[{"type":"direct","tag":"direct"}]}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	st := store.New(cfgPath)
	if err := st.Load(t.Context(), store.DefaultConfig{}); err != nil {
		t.Fatalf("store load: %v", err)
	}
	set := settings.New(filepath.Join(dir, "settings.json"))
	_ = set.Load()
	return NewHandler(HandlerOptions{Store: st, Settings: set, Version: "test"})
}

const tuicInboundJSON = `{"type":"trojan","tag":"trojan-in","listen":"127.0.0.1","listen_port":2080,"users":[]}`
const vlessInboundJSON = `{"type":"vless","tag":"vless-in","listen":"127.0.0.1","listen_port":2081,"users":[]}`
const mixedInboundJSON = `{"type":"mixed","tag":"mixed-in","listen":"127.0.0.1","listen_port":2082}`

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestUserCreateBindsToInbounds(t *testing.T) {
	h := newTestHandlerWithInbounds(t, tuicInboundJSON+","+vlessInboundJSON+","+mixedInboundJSON)

	// 创建用户绑定 tuic-in + vless-in（mixed 不支持 users，应被忽略）
	rec := doRequest(t, h, http.MethodPost, "/api/users",
		`{"name":"alice","uuid":"11111111-1111-1111-1111-111111111111","password":"pwd1","flow":"xtls-rprx-vision","bound_inbounds":["trojan-in","vless-in","mixed-in"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rec.Code, rec.Body.String())
	}

	// tuic-in 应注入 {name,uuid,password}（无 flow）
	rec2 := doRequest(t, h, http.MethodGet, "/api/inbounds/trojan-in", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("get tuic-in status = %d", rec2.Code)
	}
	body := rec2.Body.String()
	if !contains(body, `"name":"alice"`) || !contains(body, `"password":"pwd1"`) {
		t.Errorf("trojan-in users wrong: %s", body)
	}
	if contains(body, "11111111") || contains(body, "xtls-rprx-vision") {
		t.Errorf("trojan 不应包含 uuid/flow 字段: %s", body)
	}

	// vless-in 应注入 {name,uuid,flow}（无 password）
	rec3 := doRequest(t, h, http.MethodGet, "/api/inbounds/vless-in", "")
	if rec3.Code != http.StatusOK {
		t.Fatalf("get vless-in status = %d", rec3.Code)
	}
	body3 := rec3.Body.String()
	if !contains(body3, `"flow":"xtls-rprx-vision"`) {
		t.Errorf("vless-in 应含 flow: %s", body3)
	}
	if contains(body3, "pwd1") {
		t.Errorf("vless 不应包含 password: %s", body3)
	}

	// mixed-in 不应有 users
	rec4 := doRequest(t, h, http.MethodGet, "/api/inbounds/mixed-in", "")
	if contains(rec4.Body.String(), `"users"`) {
		t.Errorf("mixed-in 不应有 users: %s", rec4.Body.String())
	}
}

func TestUserUnbindRemovesFromInbound(t *testing.T) {
	h := newTestHandlerWithInbounds(t, tuicInboundJSON)
	rec := doRequest(t, h, http.MethodPost, "/api/users",
		`{"name":"bob","uuid":"22222222-2222-2222-2222-222222222222","bound_inbounds":["trojan-in"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	// 解绑
	rec2 := doRequest(t, h, http.MethodPut, "/api/users/bob",
		`{"name":"bob","uuid":"22222222-2222-2222-2222-222222222222","bound_inbounds":[]}`)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}
	rec3 := doRequest(t, h, http.MethodGet, "/api/inbounds/trojan-in", "")
	if contains(rec3.Body.String(), "bob") {
		t.Errorf("解绑后 tuic-in 仍含 bob: %s", rec3.Body.String())
	}
}

func TestUserDeleteRemovesFromAllInbounds(t *testing.T) {
	h := newTestHandlerWithInbounds(t, tuicInboundJSON+","+vlessInboundJSON)
	doRequest(t, h, http.MethodPost, "/api/users",
		`{"name":"carol","uuid":"33333333-3333-3333-3333-333333333333","bound_inbounds":["trojan-in","vless-in"]}`)
	rec := doRequest(t, h, http.MethodDelete, "/api/users/carol", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	for _, tag := range []string{"trojan-in", "vless-in"} {
		rec2 := doRequest(t, h, http.MethodGet, "/api/inbounds/"+tag, "")
		if contains(rec2.Body.String(), "carol") {
			t.Errorf("%s 删除用户后仍含 carol: %s", tag, rec2.Body.String())
		}
	}
}

func TestUserDuplicateRejected(t *testing.T) {
	h := newTestHandlerWithInbounds(t, tuicInboundJSON)
	doRequest(t, h, http.MethodPost, "/api/users", `{"name":"dup","uuid":"44444444-4444-4444-4444-444444444444"}`)
	rec := doRequest(t, h, http.MethodPost, "/api/users", `{"name":"dup","uuid":"55555555-5555-5555-5555-555555555555"}`)
	if rec.Code == http.StatusOK {
		t.Errorf("重复用户应被拒绝: %s", rec.Body.String())
	}
}
