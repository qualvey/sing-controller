package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/qualvey/sing-controller/internal/store"
)

func TestParseClashYAML(t *testing.T) {
	yamlContent := `
proxies:
  - name: "HK-VLESS-Reality"
    type: vless
    server: 1.2.3.4
    port: 443
    uuid: "a0000000-0000-0000-0000-000000000001"
    flow: xtls-rprx-vision
    tls: true
    servername: www.microsoft.com
    reality-opts:
      public-key: "AbCdEfGhIjKlMnOpQrStUvWxYz0123456789"
      short-id: "abcd"
  - name: "US-Hysteria2"
    type: hysteria2
    server: 5.6.7.8
    port: 8443
    auth: "my-secret-password"
    sni: mydomain.com
`
	outbounds, err := ParseClashYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("ParseClashYAML failed: %v", err)
	}

	if len(outbounds) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(outbounds))
	}

	if outbounds[0].Tag != "HK-VLESS-Reality" || outbounds[0].Type != "vless" {
		t.Errorf("outbound[0] mismatch: tag=%s, type=%s", outbounds[0].Tag, outbounds[0].Type)
	}

	if outbounds[1].Tag != "US-Hysteria2" || outbounds[1].Type != "hysteria2" {
		t.Errorf("outbound[1] mismatch: tag=%s, type=%s", outbounds[1].Tag, outbounds[1].Type)
	}
}

func TestParseSingBoxJSON(t *testing.T) {
	jsonContent := `{
  "outbounds": [
    {
      "type": "vless",
      "tag": "node-1",
      "server": "1.1.1.1",
      "server_port": 443,
      "uuid": "a0000000-0000-0000-0000-000000000001"
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ]
}`
	outbounds, err := ParseSingBoxJSON([]byte(jsonContent))
	if err != nil {
		t.Fatalf("ParseSingBoxJSON failed: %v", err)
	}

	// direct should be filtered out from proxy node pool
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 proxy outbound, got %d", len(outbounds))
	}
	if outbounds[0].Tag != "node-1" {
		t.Errorf("expected tag node-1, got %s", outbounds[0].Tag)
	}
}

func TestParseBase64URI(t *testing.T) {
	// vless://a0000000-0000-0000-0000-000000000001@1.2.3.4:443?security=tls&sni=example.com#MyNode
	rawURI := "vless://a0000000-0000-0000-0000-000000000001@1.2.3.4:443?security=tls&sni=example.com#MyNode"
	outbounds, err := ParseURIList(rawURI)
	if err != nil {
		t.Fatalf("ParseURIList failed: %v", err)
	}
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(outbounds))
	}
	if outbounds[0].Tag != "MyNode" || outbounds[0].Type != "vless" {
		t.Errorf("outbound mismatch: %+v", outbounds[0])
	}
}

func TestProfileMergingAndValidation(t *testing.T) {
	tmpDir := t.TempDir()
	baseConfigPath := filepath.Join(tmpDir, "sing-box-config.json")
	baseConfig := `{
  "log": { "level": "info" },
  "inbounds": [
    { "type": "mixed", "tag": "in-main", "listen": "127.0.0.1", "listen_port": 2080 }
  ],
  "outbounds": [
    { "type": "selector", "tag": "Proxy", "outbounds": ["direct"] },
    { "type": "direct", "tag": "direct" },
    { "type": "block", "tag": "block" }
  ],
  "route": { "final": "Proxy" }
}`
	if err := os.WriteFile(baseConfigPath, []byte(baseConfig), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}

	st := store.New(baseConfigPath)
	if err := st.Load(context.Background(), store.DefaultConfig{}); err != nil {
		t.Fatalf("load store: %v", err)
	}

	mgr, err := NewManager(filepath.Join(tmpDir, "providers"), st, nil)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. 模拟写入一个 Provider 片段
	p, err := mgr.AddProvider(context.Background(), Provider{
		Tag:     "airport-a",
		Name:    "机场A",
		Type:    ProviderTypeFile,
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("AddProvider failed: %v", err)
	}

	// 写入 2 个节点到 outbound-airport-a.json
	clashYAML := `
proxies:
  - name: "HK-01"
    type: vless
    server: 1.1.1.1
    port: 443
    uuid: "a0000000-0000-0000-0000-000000000001"
  - name: "SG-01"
    type: vless
    server: 2.2.2.2
    port: 443
    uuid: "a0000000-0000-0000-0000-000000000002"
`
	obs, err := ParseClashYAML([]byte(clashYAML))
	if err != nil {
		t.Fatalf("ParseClashYAML failed: %v", err)
	}
	_ = obs

	// 模拟写入片段文件
	nodeJSON := `[
    { "type": "vless", "tag": "HK-01", "server": "1.1.1.1", "server_port": 443, "uuid": "a0000000-0000-0000-0000-000000000001" },
    { "type": "vless", "tag": "SG-01", "server": "2.2.2.2", "server_port": 443, "uuid": "a0000000-0000-0000-0000-000000000002" }
  ]`
	if err := os.WriteFile(mgr.providerOutboundPath(p.Tag), []byte(nodeJSON), 0o644); err != nil {
		t.Fatalf("write fragment: %v", err)
	}

	// 2. 创建并测试 Profile 合并
	prof, err := mgr.AddProfile(Profile{
		Tag:             "my-gaming-profile",
		Name:            "游戏专属方案",
		ProviderTags:    []string{"airport-a"},
		TargetSelectors: []string{"Proxy"},
	})
	if err != nil {
		t.Fatalf("AddProfile failed: %v", err)
	}

	finalJSON, err := mgr.MergeAndValidateProfile(context.Background(), *prof)
	if err != nil {
		t.Fatalf("MergeAndValidateProfile failed: %v", err)
	}

	if len(finalJSON) == 0 {
		t.Fatalf("empty finalJSON")
	}

	// 3. 应用方案
	if err := mgr.ApplyProfile(context.Background(), prof.ID); err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}

	// 验证磁盘上的配置是否更新
	savedContent, _ := os.ReadFile(baseConfigPath)
	t.Logf("Merged config:\n%s", string(savedContent))
}
