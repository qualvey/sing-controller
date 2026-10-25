package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHandleToolParseNode(t *testing.T) {
	h := newTestHandler(t)

	t.Run("parse vless reality link", func(t *testing.T) {
		link := "vless://a0000000-0000-0000-0000-000000000001@1.2.3.4:443?security=reality&sni=example.com&pbk=wBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB&sid=12345678&fp=chrome&flow=xtls-rprx-vision#MyVlessNode"
		body, _ := json.Marshal(map[string]string{"text": link})
		rec := doRequest(t, h, http.MethodPost, "/api/tools/parse-node", string(body))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Ok   bool           `json:"ok"`
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if !resp.Ok {
			t.Fatalf("expected ok=true")
		}
		if resp.Data["type"] != "vless" {
			t.Errorf("expected type vless, got %v", resp.Data["type"])
		}
		if resp.Data["tag"] != "MyVlessNode" {
			t.Errorf("expected tag MyVlessNode, got %v", resp.Data["tag"])
		}
		if resp.Data["server"] != "1.2.3.4" {
			t.Errorf("expected server 1.2.3.4, got %v", resp.Data["server"])
		}
		if resp.Data["uuid"] != "a0000000-0000-0000-0000-000000000001" {
			t.Errorf("expected uuid, got %v", resp.Data["uuid"])
		}
	})

	t.Run("parse json outbound", func(t *testing.T) {
		jsonText := `{"type":"vless","tag":"vless-json","server":"8.8.8.8","server_port":443,"uuid":"11111111-2222-3333-4444-555555555555"}`
		body, _ := json.Marshal(map[string]string{"text": jsonText})
		rec := doRequest(t, h, http.MethodPost, "/api/tools/parse-node", string(body))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Ok   bool           `json:"ok"`
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Data["tag"] != "vless-json" || resp.Data["type"] != "vless" {
			t.Errorf("unexpected data: %v", resp.Data)
		}
	})

	t.Run("parse invalid link returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"text": "invalid://not-a-supported-link"})
		rec := doRequest(t, h, http.MethodPost, "/api/tools/parse-node", string(body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("parse empty returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"text": "   "})
		rec := doRequest(t, h, http.MethodPost, "/api/tools/parse-node", string(body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
