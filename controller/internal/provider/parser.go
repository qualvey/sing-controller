package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
	"gopkg.in/yaml.v3"
)

// AutoParse 根据数据特征或指定格式自动解析出 sing-box 的 Outbound 节点列表
func AutoParse(data []byte, format SubscriptionFormat) ([]option.Outbound, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("订阅内容为空")
	}

	switch format {
	case FormatSingBox:
		return ParseSingBoxJSON(trimmed)
	case FormatClash:
		return ParseClashYAML(trimmed)
	case FormatBase64:
		return ParseBase64URI(trimmed)
	}

	// 自动探测格式：
	// 1. JSON 格式（以 { 或 [ 开头）
	if trimmed[0] == '{' || trimmed[0] == '[' {
		if outbounds, err := ParseSingBoxJSON(trimmed); err == nil && len(outbounds) > 0 {
			return outbounds, nil
		}
	}

	// 2. YAML 格式（包含 proxies: 或 server:）
	if bytes.Contains(trimmed, []byte("proxies:")) || bytes.Contains(trimmed, []byte("Proxy:")) {
		if outbounds, err := ParseClashYAML(trimmed); err == nil && len(outbounds) > 0 {
			return outbounds, nil
		}
	}

	// 3. Base64 格式
	if outbounds, err := ParseBase64URI(trimmed); err == nil && len(outbounds) > 0 {
		return outbounds, nil
	}

	// 4. 明文 URI 列表（以 ss://, vless:// 等开头）
	if outbounds, err := ParseURIList(string(trimmed)); err == nil && len(outbounds) > 0 {
		return outbounds, nil
	}

	// 最后兜底尝试 YAML
	if outbounds, err := ParseClashYAML(trimmed); err == nil && len(outbounds) > 0 {
		return outbounds, nil
	}

	return nil, fmt.Errorf("无法识别该订阅格式，请检查订阅链接或内容")
}

// ParseSingBoxJSON 解析 sing-box 格式配置中的 outbounds
func ParseSingBoxJSON(data []byte) ([]option.Outbound, error) {
	ctx := include.Context(context.Background())

	// 尝试解析为包含 outbounds 字段的对象
	type docType struct {
		Outbounds []option.Outbound `json:"outbounds"`
	}
	if doc, err := sjson.UnmarshalExtendedContext[docType](ctx, data); err == nil && len(doc.Outbounds) > 0 {
		return filterProxyOutbounds(doc.Outbounds), nil
	}

	// 尝试直接解析为 []option.Outbound 数组
	if outbounds, err := sjson.UnmarshalExtendedContext[[]option.Outbound](ctx, data); err == nil && len(outbounds) > 0 {
		return filterProxyOutbounds(outbounds), nil
	}

	// 兜底尝试全量 Options 解析
	if opts, err := sjson.UnmarshalExtendedContext[option.Options](ctx, data); err == nil && len(opts.Outbounds) > 0 {
		return filterProxyOutbounds(opts.Outbounds), nil
	}

	return nil, fmt.Errorf("未找到有效的 sing-box outbounds 节点")
}

// filterProxyOutbounds 过滤掉 direct/block/dns 等非代理节点，保留实际转发节点
func filterProxyOutbounds(list []option.Outbound) []option.Outbound {
	res := make([]option.Outbound, 0, len(list))
	for _, ob := range list {
		switch ob.Type {
		case "direct", "block", "dns", "selector", "urltest":
			continue
		default:
			res = append(res, ob)
		}
	}
	return res
}

// ClashProxyRaw 用于接收 Clash YAML 中 proxies 节点段的通用字段
type ClashProxyRaw map[string]any

// ParseClashYAML 解析 Clash YAML 订阅并提取转换节点
func ParseClashYAML(data []byte) ([]option.Outbound, error) {
	var doc struct {
		Proxies []ClashProxyRaw `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析 Clash YAML 失败: %w", err)
	}

	if len(doc.Proxies) == 0 {
		return nil, fmt.Errorf("Clash 配置中未找到 proxies 节点列表")
	}

	ctx := include.Context(context.Background())
	outbounds := make([]option.Outbound, 0, len(doc.Proxies))

	for _, proxy := range doc.Proxies {
		sbJSON, err := convertClashProxyToSingBoxJSON(proxy)
		if err != nil || len(sbJSON) == 0 {
			continue
		}
		if ob, err := sjson.UnmarshalExtendedContext[option.Outbound](ctx, sbJSON); err == nil {
			outbounds = append(outbounds, ob)
		}
	}

	if len(outbounds) == 0 {
		return nil, fmt.Errorf("未能成功转换任何 Clash 节点")
	}
	return outbounds, nil
}

// convertClashProxyToSingBoxJSON 将单个 Clash 节点字典转为 sing-box outbound JSON 结构
func convertClashProxyToSingBoxJSON(p ClashProxyRaw) ([]byte, error) {
	pType, _ := p["type"].(string)
	name, _ := p["name"].(string)
	server, _ := p["server"].(string)
	port := getInt(p["port"])

	if pType == "" || name == "" || server == "" || port == 0 {
		return nil, nil
	}

	sb := map[string]any{
		"tag":         name,
		"server":      server,
		"server_port": port,
	}

	switch strings.ToLower(pType) {
	case "ss", "shadowsocks":
		sb["type"] = "shadowsocks"
		sb["method"], _ = p["cipher"].(string)
		sb["password"], _ = p["password"].(string)
		if plugin, ok := p["plugin"].(string); ok && plugin != "" {
			sb["plugin"] = plugin
			if opts, ok := p["plugin-opts"].(map[string]any); ok {
				sb["plugin_opts"] = opts
			}
		}

	case "vmess":
		sb["type"] = "vmess"
		sb["uuid"], _ = p["uuid"].(string)
		sb["alter_id"] = getInt(p["alterId"])
		sb["security"], _ = p["cipher"].(string)
		if sb["security"] == "" {
			sb["security"] = "auto"
		}
		addClashTransport(sb, p)

	case "vless":
		sb["type"] = "vless"
		sb["uuid"], _ = p["uuid"].(string)
		if flow, ok := p["flow"].(string); ok && flow != "" {
			sb["flow"] = flow
		}
		addClashTransport(sb, p)

	case "trojan":
		sb["type"] = "trojan"
		sb["password"], _ = p["password"].(string)
		addClashTransport(sb, p)

	case "hysteria2", "hy2":
		sb["type"] = "hysteria2"
		sb["password"], _ = p["password"].(string)
		if auth, ok := p["auth"].(string); ok && auth != "" {
			sb["password"] = auth
		}
		tls := map[string]any{"enabled": true}
		if sni, ok := p["sni"].(string); ok && sni != "" {
			tls["server_name"] = sni
		}
		if getBool(p["skip-cert-verify"]) {
			tls["insecure"] = true
		}
		if alpn, ok := p["alpn"].([]any); ok && len(alpn) > 0 {
			tls["alpn"] = alpn
		}
		sb["tls"] = tls
		if obfs, ok := p["obfs"].(string); ok && obfs != "" {
			sb["obfs"] = map[string]any{
				"type":     obfs,
				"password": p["obfs-password"],
			}
		}

	case "tuic":
		sb["type"] = "tuic"
		sb["uuid"], _ = p["uuid"].(string)
		sb["password"], _ = p["password"].(string)
		sb["congestion_control"], _ = p["congestion-controller"].(string)
		if sb["congestion_control"] == "" {
			sb["congestion_control"] = "bbr"
		}
		tls := map[string]any{"enabled": true}
		if sni, ok := p["sni"].(string); ok && sni != "" {
			tls["server_name"] = sni
		}
		if getBool(p["skip-cert-verify"]) {
			tls["insecure"] = true
		}
		if alpn, ok := p["alpn"].([]any); ok && len(alpn) > 0 {
			tls["alpn"] = alpn
		}
		sb["tls"] = tls

	default:
		return nil, nil
	}

	return json.Marshal(sb)
}

func addClashTransport(sb map[string]any, p ClashProxyRaw) {
	tlsEnabled := getBool(p["tls"])
	if sni, ok := p["sni"].(string); ok && sni != "" {
		tlsEnabled = true
	}

	if tlsEnabled {
		tls := map[string]any{"enabled": true}
		if sni, ok := p["sni"].(string); ok && sni != "" {
			tls["server_name"] = sni
		} else if servername, ok := p["servername"].(string); ok && servername != "" {
			tls["server_name"] = servername
		}
		if getBool(p["skip-cert-verify"]) {
			tls["insecure"] = true
		}
		if alpn, ok := p["alpn"].([]any); ok && len(alpn) > 0 {
			tls["alpn"] = alpn
		}
		if realityOpts, ok := p["reality-opts"].(map[string]any); ok {
			reality := map[string]any{"enabled": true}
			reality["public_key"] = realityOpts["public-key"]
			reality["short_id"] = realityOpts["short-id"]
			tls["reality"] = reality
		}
		sb["tls"] = tls
	}

	network, _ := p["network"].(string)
	switch strings.ToLower(network) {
	case "ws", "websocket":
		transport := map[string]any{"type": "ws"}
		if wsOpts, ok := p["ws-opts"].(map[string]any); ok {
			if path, ok := wsOpts["path"].(string); ok && path != "" {
				transport["path"] = path
			}
			if headers, ok := wsOpts["headers"].(map[string]any); ok {
				transport["headers"] = headers
			}
		}
		sb["transport"] = transport
	case "grpc":
		transport := map[string]any{"type": "grpc"}
		if grpcOpts, ok := p["grpc-opts"].(map[string]any); ok {
			if serviceName, ok := grpcOpts["grpc-service-name"].(string); ok {
				transport["service_name"] = serviceName
			}
		}
		sb["transport"] = transport
	case "http":
		transport := map[string]any{"type": "http"}
		if hOpts, ok := p["http-opts"].(map[string]any); ok {
			if path, ok := hOpts["path"].(string); ok {
				transport["path"] = path
			}
		}
		sb["transport"] = transport
	}
}

func decodeBase64Flexible(cleanStr string) ([]byte, error) {
	cleanStr = strings.TrimSpace(cleanStr)
	cleanStr = strings.ReplaceAll(cleanStr, " ", "")
	cleanStr = strings.ReplaceAll(cleanStr, "\n", "")
	cleanStr = strings.ReplaceAll(cleanStr, "\r", "")

	if decoded, err := base64.StdEncoding.DecodeString(cleanStr); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(cleanStr); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(cleanStr); err == nil {
		return decoded, nil
	}
	return base64.RawURLEncoding.DecodeString(cleanStr)
}

// ParseBase64URI 解析 Base64 编码的节点 URI 列表
func ParseBase64URI(data []byte) ([]option.Outbound, error) {
	decoded, err := decodeBase64Flexible(string(data))
	if err != nil {
		return nil, fmt.Errorf("Base64 解码失败: %w", err)
	}
	return ParseURIList(string(decoded))
}

// ParseURIList 解析每行一个的代理链接（vless://, vmess://, ss://, trojan://, hy2://, tuic://）
func ParseURIList(text string) ([]option.Outbound, error) {
	ctx := include.Context(context.Background())
	lines := strings.Split(text, "\n")
	outbounds := make([]option.Outbound, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sbJSON, err := parseProxyURIToSingBox(line)
		if err != nil || len(sbJSON) == 0 {
			continue
		}
		if ob, err := sjson.UnmarshalExtendedContext[option.Outbound](ctx, sbJSON); err == nil {
			outbounds = append(outbounds, ob)
		}
	}

	if len(outbounds) == 0 {
		return nil, fmt.Errorf("未提取到任何有效代理 URI")
	}
	return outbounds, nil
}

// parseProxyURIToSingBox 解析单个 URI (如 vless://..., vmess://..., ss://...)
func parseProxyURIToSingBox(rawURI string) ([]byte, error) {
	lowerURI := strings.ToLower(rawURI)
	if strings.HasPrefix(lowerURI, "vmess://") {
		return parseVmessURI(rawURI)
	}

	u, err := url.Parse(rawURI)
	if err != nil {
		return nil, err
	}

	tag := u.Fragment
	if tag == "" {
		tag = u.Host
	} else {
		if unescaped, err := url.QueryUnescape(tag); err == nil {
			tag = unescaped
		}
	}

	host := u.Hostname()
	portStr := u.Port()
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 443
	}

	sb := map[string]any{
		"tag":         tag,
		"server":      host,
		"server_port": port,
	}

	q := u.Query()

	switch strings.ToLower(u.Scheme) {
	case "vless":
		sb["type"] = "vless"
		if u.User != nil {
			sb["uuid"] = u.User.Username()
		}
		if flow := q.Get("flow"); flow != "" {
			sb["flow"] = flow
		}
		if q.Get("security") == "tls" || q.Get("security") == "reality" {
			tls := map[string]any{"enabled": true}
			if sni := q.Get("sni"); sni != "" {
				tls["server_name"] = sni
			}
			if q.Get("security") == "reality" {
				tls["reality"] = map[string]any{
					"enabled":    true,
					"public_key": q.Get("pbk"),
					"short_id":   q.Get("sid"),
				}
			}
			sb["tls"] = tls
		}
		if netType := q.Get("type"); netType == "ws" || netType == "grpc" {
			sb["transport"] = map[string]any{
				"type":         netType,
				"path":         q.Get("path"),
				"service_name": q.Get("serviceName"),
			}
		}

	case "trojan":
		sb["type"] = "trojan"
		if u.User != nil {
			sb["password"] = u.User.Username()
		}
		tls := map[string]any{"enabled": true}
		if sni := q.Get("sni"); sni != "" {
			tls["server_name"] = sni
		}
		sb["tls"] = tls

	case "hysteria2", "hy2":
		sb["type"] = "hysteria2"
		if u.User != nil {
			sb["password"] = u.User.Username()
		}
		tls := map[string]any{"enabled": true}
		if sni := q.Get("sni"); sni != "" {
			tls["server_name"] = sni
		}
		sb["tls"] = tls
		if obfs := q.Get("obfs"); obfs != "" {
			sb["obfs"] = map[string]any{
				"type":     obfs,
				"password": q.Get("obfs-password"),
			}
		}

	case "tuic":
		sb["type"] = "tuic"
		if u.User != nil {
			sb["uuid"] = u.User.Username()
			sb["password"], _ = u.User.Password()
		}
		tls := map[string]any{"enabled": true}
		if sni := q.Get("sni"); sni != "" {
			tls["server_name"] = sni
		}
		if q.Get("insecure") == "1" || strings.EqualFold(q.Get("allowInsecure"), "true") {
			tls["insecure"] = true
		}
		if alpn := q.Get("alpn"); alpn != "" {
			tls["alpn"] = strings.Split(alpn, ",")
		}
		sb["tls"] = tls
		if cc := q.Get("congestion_control"); cc != "" {
			sb["congestion_control"] = cc
		} else {
			sb["congestion_control"] = "bbr"
		}

	case "ss":
		sb["type"] = "shadowsocks"
		if u.User != nil {
			username := u.User.Username()
			password, hasPass := u.User.Password()
			if hasPass {
				sb["method"] = username
				sb["password"] = password
			} else {
				// Base64-encoded method:password
				if decoded, err := decodeBase64Flexible(username); err == nil {
					parts := strings.SplitN(string(decoded), ":", 2)
					if len(parts) == 2 {
						sb["method"] = parts[0]
						sb["password"] = parts[1]
					}
				}
			}
		}

	default:
		return nil, nil
	}

	return json.Marshal(sb)
}

func parseVmessURI(rawURI string) ([]byte, error) {
	b64 := strings.TrimPrefix(rawURI, "vmess://")
	b64 = strings.TrimPrefix(b64, "VMESS://")
	decoded, err := decodeBase64Flexible(b64)
	if err != nil {
		return nil, err
	}
	var v map[string]any
	if err := json.Unmarshal(decoded, &v); err != nil {
		return nil, err
	}
	tag, _ := v["ps"].(string)
	server, _ := v["add"].(string)
	port := getInt(v["port"])
	uuid, _ := v["id"].(string)
	alterID := getInt(v["aid"])
	net, _ := v["net"].(string)
	tlsStr, _ := v["tls"].(string)
	sni, _ := v["sni"].(string)
	host, _ := v["host"].(string)
	path, _ := v["path"].(string)

	if tag == "" {
		tag = server
	}
	if port == 0 {
		port = 443
	}

	sb := map[string]any{
		"type":        "vmess",
		"tag":         tag,
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
		"alter_id":    alterID,
		"security":    "auto",
	}
	if strings.EqualFold(tlsStr, "tls") {
		tls := map[string]any{"enabled": true}
		if sni != "" {
			tls["server_name"] = sni
		} else if host != "" {
			tls["server_name"] = host
		}
		sb["tls"] = tls
	}
	if strings.EqualFold(net, "ws") || strings.EqualFold(net, "websocket") {
		transport := map[string]any{"type": "ws"}
		if path != "" {
			transport["path"] = path
		}
		if host != "" {
			transport["headers"] = map[string]any{"Host": host}
		}
		sb["transport"] = transport
	} else if strings.EqualFold(net, "grpc") {
		transport := map[string]any{"type": "grpc"}
		if path != "" {
			transport["service_name"] = path
		}
		sb["transport"] = transport
	}
	return json.Marshal(sb)
}

func getInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

func getBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return strings.ToLower(val) == "true" || val == "1"
	case int:
		return val != 0
	default:
		return false
	}
}
