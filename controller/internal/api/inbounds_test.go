package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestToolPassword 密码工具：16 字节随机 → 标准 base64（与 shadowsocks 示例密码格式一致）。
func TestToolPassword(t *testing.T) {
	h := newTestHandler(t)
	rec := doRequest(t, h, http.MethodPost, "/api/tools/password", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Password == "" {
		t.Fatal("password is empty")
	}
	raw, err := base64.StdEncoding.DecodeString(resp.Password)
	if err != nil {
		t.Fatalf("password not base64: %v", err)
	}
	if len(raw) != 16 {
		t.Fatalf("password decodes to %d bytes, want 16", len(raw))
	}
	// 两次生成不应相同（随机性冒烟）
	rec2 := doRequest(t, h, http.MethodPost, "/api/tools/password", "")
	var resp2 struct {
		Password string `json:"password"`
	}
	_ = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	if resp2.Password == resp.Password {
		t.Error("two generated passwords are identical")
	}
}

// TestShadowsocksInboundCRUD shadowsocks 入站控制：创建/回读/更新/删除 + 非法配置拒绝。
// 默认值（method chacha20-ietf-poly1305 / listen :: / 端口 23010 / 密码自动生成）由前端表单
// 负责填充，后端保持通用 CRUD + 全量校验管线（box.New 干跑）。
func TestShadowsocksInboundCRUD(t *testing.T) {
	h := newTestHandler(t)

	// 创建（等价于用户给出的示例配置）
	body := `{"type":"shadowsocks","tag":"ss-in","listen":"::","listen_port":23010,"method":"chacha20-ietf-poly1305","password":"8JCsPssfgS8tiRwiMlhARg=="}`
	rec := doRequest(t, h, http.MethodPost, "/api/inbounds", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", rec.Code, rec.Body.String())
	}

	// 回读：字段完整保留
	rec2 := doRequest(t, h, http.MethodGet, "/api/inbounds/ss-in", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", rec2.Code, rec2.Body.String())
	}
	for _, want := range []string{`"type":"shadowsocks"`, `"listen":"::"`, `"listen_port":23010`, `"method":"chacha20-ietf-poly1305"`, "8JCsPssfgS8tiRwiMlhARg=="} {
		if !strings.Contains(rec2.Body.String(), want) {
			t.Errorf("get response missing %s:\n%s", want, rec2.Body.String())
		}
	}

	// 列表也返回
	recList := doRequest(t, h, http.MethodGet, "/api/inbounds", "")
	if !strings.Contains(recList.Body.String(), `"ss-in"`) {
		t.Errorf("list missing ss-in:\n%s", recList.Body.String())
	}

	// 更新：换 method/password
	update := `{"type":"shadowsocks","tag":"ss-in","listen":"0.0.0.0","listen_port":23011,"method":"aes-256-gcm","password":"newpass"}`
	rec3 := doRequest(t, h, http.MethodPut, "/api/inbounds/ss-in", update)
	if rec3.Code != http.StatusOK {
		t.Fatalf("update status = %d, body=%s", rec3.Code, rec3.Body.String())
	}
	rec4 := doRequest(t, h, http.MethodGet, "/api/inbounds/ss-in", "")
	if !strings.Contains(rec4.Body.String(), `"method":"aes-256-gcm"`) || !strings.Contains(rec4.Body.String(), `"newpass"`) {
		t.Errorf("update not persisted:\n%s", rec4.Body.String())
	}

	// 非法 method 拒绝
	bad := `{"type":"shadowsocks","tag":"ss-bad","method":"aes-256-ctr","password":"x"}`
	rec5 := doRequest(t, h, http.MethodPost, "/api/inbounds", bad)
	if rec5.Code != http.StatusBadRequest {
		t.Errorf("invalid method accepted: status=%d body=%s", rec5.Code, rec5.Body.String())
	}

	// 非 none method 缺 password 拒绝
	noPass := `{"type":"shadowsocks","tag":"ss-nopass","method":"aes-128-gcm"}`
	rec6 := doRequest(t, h, http.MethodPost, "/api/inbounds", noPass)
	if rec6.Code != http.StatusBadRequest {
		t.Errorf("missing password accepted: status=%d body=%s", rec6.Code, rec6.Body.String())
	}

	// none method 可无 password
	none := `{"type":"shadowsocks","tag":"ss-none","method":"none"}`
	rec7 := doRequest(t, h, http.MethodPost, "/api/inbounds", none)
	if rec7.Code != http.StatusOK {
		t.Errorf("none method rejected: status=%d body=%s", rec7.Code, rec7.Body.String())
	}

	// 多用户：走用户池绑定投影（用户池为唯一真相，users[] 由 syncUsersToInbounds 注入）
	user := `{"name":"u1","password":"p1","bound_inbounds":["ss-in"]}`
	rec8 := doRequest(t, h, http.MethodPost, "/api/users", user)
	if rec8.Code != http.StatusOK {
		t.Fatalf("create user status = %d, body=%s", rec8.Code, rec8.Body.String())
	}
	rec8b := doRequest(t, h, http.MethodGet, "/api/inbounds/ss-in", "")
	if rec8b.Code != http.StatusOK {
		t.Fatalf("get after bind status = %d", rec8b.Code)
	}
	if !strings.Contains(rec8b.Body.String(), `"users"`) || !strings.Contains(rec8b.Body.String(), `"u1"`) || !strings.Contains(rec8b.Body.String(), `"p1"`) {
		t.Errorf("pool user not injected into shadowsocks inbound:\n%s", rec8b.Body.String())
	}

	// 删除
	rec9 := doRequest(t, h, http.MethodDelete, "/api/inbounds/ss-in", "")
	if rec9.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body=%s", rec9.Code, rec9.Body.String())
	}
	rec10 := doRequest(t, h, http.MethodGet, "/api/inbounds/ss-in", "")
	if rec10.Code != http.StatusNotFound {
		t.Errorf("deleted inbound still exists: status=%d", rec10.Code)
	}
}
