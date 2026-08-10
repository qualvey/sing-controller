package api

import (
	"fmt"
	"net/http"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// 配置静态诊断：重复 tag、悬空引用、端口冲突、未使用资源等。
// 纯只读分析（内存态），不校验、不写盘；深度校验仍由写操作管线的 box.New 干跑负责。

type Diagnostic struct {
	Level   string `json:"level"` // error | warning | info
	Message string `json:"message"`
}

func (h *Handler) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	options := h.store.Options
	var out []Diagnostic

	add := func(level, format string, args ...any) {
		out = append(out, Diagnostic{Level: level, Message: fmt.Sprintf(format, args...)})
	}

	// ---------- 重复 tag ----------
	outboundTags := map[string]int{}
	for _, ob := range options.Outbounds {
		outboundTags[ob.Tag]++
	}
	inboundTags := map[string]int{}
	for _, ib := range options.Inbounds {
		inboundTags[ib.Tag]++
	}
	dnsTags := map[string]int{}
	if options.DNS != nil {
		for _, s := range options.DNS.Servers {
			dnsTags[s.Tag]++
		}
	}
	for tag, n := range outboundTags {
		if n > 1 {
			add("error", "outbound tag 重复: %s（%d 次）", tag, n)
		}
	}
	for tag, n := range inboundTags {
		if n > 1 {
			add("error", "inbound tag 重复: %s（%d 次）", tag, n)
		}
	}
	for tag, n := range dnsTags {
		if n > 1 {
			add("error", "DNS server tag 重复: %s（%d 次）", tag, n)
		}
	}

	// ---------- route 引用 ----------
	if options.Route == nil {
		add("warning", "未配置 route 段：所有流量将使用默认出站")
	} else {
		if options.Route.Final == "" {
			add("warning", "route.final 为空：未匹配规则的流量将失败")
		} else if outboundTags[options.Route.Final] == 0 {
			add("error", "route.final 引用的 outbound 不存在: %s", options.Route.Final)
		}
		for i, rule := range options.Route.Rules {
			content, err := json.Marshal(&rule)
			if err != nil {
				continue
			}
			var decoded map[string]any
			if json.Unmarshal(content, &decoded) != nil {
				continue
			}
			if out, ok := decoded["outbound"].(string); ok && out != "" && outboundTags[out] == 0 {
				add("error", "route 规则 #%d 引用的 outbound 不存在: %s", i+1, out)
			}
			if refs, ok := decoded["rule_set"].([]any); ok {
				for _, ref := range refs {
					if s, ok := ref.(string); ok {
						add("warning", "route 规则 #%d 引用的 rule_set: %s（1.14 起 rule set 以独立类型定义，请确认已配置）", i+1, s)
					}
				}
			}
		}
	}

	// ---------- selector/urltest 成员 ----------
	for i := range options.Outbounds {
		ob := &options.Outbounds[i]
		var members []string
		switch typed := ob.Options.(type) {
		case *option.SelectorOutboundOptions:
			members = typed.Outbounds
		case *option.URLTestOutboundOptions:
			members = typed.Outbounds
		default:
			continue
		}
		for _, m := range members {
			if m == ob.Tag {
				add("error", "组 %s 的成员包含自身", ob.Tag)
			} else if outboundTags[m] == 0 {
				add("error", "组 %s 引用的成员 outbound 不存在: %s", ob.Tag, m)
			}
		}
	}

	// ---------- DNS 引用 ----------
	if options.DNS == nil {
		add("warning", "未配置 dns 段：域名解析使用系统默认")
	} else {
		if options.DNS.Final != "" && dnsTags[options.DNS.Final] == 0 {
			add("error", "dns.final 引用的 server 不存在: %s", options.DNS.Final)
		}
		for i := range options.DNS.Rules {
			content, err := json.Marshal(&options.DNS.Rules[i])
			if err != nil {
				continue
			}
			var decoded map[string]any
			if json.Unmarshal(content, &decoded) != nil {
				continue
			}
			refs := dnsRuleOutboundRefs(decoded)
			for _, ref := range refs {
				if dnsTags[ref] == 0 {
					add("error", "dns 规则 #%d 引用的 server 不存在: %s", i+1, ref)
				}
			}
		}
		// dns server detour 引用
		for i := range options.DNS.Servers {
			content, err := json.Marshal(&options.DNS.Servers[i])
			if err != nil {
				continue
			}
			var decoded map[string]any
			if json.Unmarshal(content, &decoded) != nil {
				continue
			}
			if detour, ok := decoded["detour"].(string); ok && detour != "" && outboundTags[detour] == 0 {
				add("error", "DNS server %s 的 detour 引用的 outbound 不存在: %s", options.DNS.Servers[i].Tag, detour)
			}
		}
	}

	// ---------- inbound 监听冲突 ----------
	seen := map[string]string{}
	for i := range options.Inbounds {
		ib := &options.Inbounds[i]
		content, err := json.Marshal(ib)
		if err != nil {
			continue
		}
		var decoded map[string]any
		if json.Unmarshal(content, &decoded) != nil {
			continue
		}
		port, _ := decoded["listen_port"].(float64)
		listen, _ := decoded["listen"].(string)
		if int(port) == 0 {
			continue
		}
		key := fmt.Sprintf("%s:%d", listen, int(port))
		if prev, ok := seen[key]; ok {
			add("error", "inbound 监听冲突: %s 与 %s 都监听 %s", ib.Tag, prev, key)
		} else {
			seen[key] = ib.Tag
		}
	}

	// ---------- 未使用 outbound ----------
	used := map[string]bool{}
	if options.Route != nil {
		if options.Route.Final != "" {
			used[options.Route.Final] = true
		}
		for _, rule := range options.Route.Rules {
			content, _ := json.Marshal(&rule)
			var decoded map[string]any
			if json.Unmarshal(content, &decoded) == nil {
				if out, ok := decoded["outbound"].(string); ok {
					used[out] = true
				}
			}
		}
	}
	for i := range options.Outbounds {
		ob := &options.Outbounds[i]
		switch typed := ob.Options.(type) {
		case *option.SelectorOutboundOptions:
			used[ob.Tag] = true
			for _, m := range typed.Outbounds {
				used[m] = true
			}
		case *option.URLTestOutboundOptions:
			used[ob.Tag] = true
			for _, m := range typed.Outbounds {
				used[m] = true
			}
		case *option.DirectOutboundOptions:
			used[ob.Tag] = true
		}
	}
	if options.DNS != nil {
		for i := range options.DNS.Servers {
			content, _ := json.Marshal(&options.DNS.Servers[i])
			var decoded map[string]any
			if json.Unmarshal(content, &decoded) == nil {
				if detour, ok := decoded["detour"].(string); ok {
					used[detour] = true
				}
			}
		}
	}
	for tag := range outboundTags {
		if !used[tag] {
			add("info", "outbound 未被引用: %s", tag)
		}
	}

	// ---------- 统计 ----------
	add("info", "inbounds: %d，outbounds: %d，route 规则: %d，DNS servers: %d，DNS 规则: %d",
		len(options.Inbounds), len(options.Outbounds), routeRuleCount(options), dnsServerCount(options), dnsRuleCount(options))

	writeJSON(w, http.StatusOK, map[string]any{"diagnostics": out})
}

func routeRuleCount(options option.Options) int {
	if options.Route == nil {
		return 0
	}
	return len(options.Route.Rules)
}

func dnsServerCount(options option.Options) int {
	if options.DNS == nil {
		return 0
	}
	return len(options.DNS.Servers)
}

func dnsRuleCount(options option.Options) int {
	if options.DNS == nil {
		return 0
	}
	return len(options.DNS.Rules)
}

// dnsRuleOutboundRefs 从解码后的规则对象中提取 server/outbound 引用（新模型 server + 旧 outbound 兼容）。
func dnsRuleOutboundRefs(decoded map[string]any) []string {
	var refs []string
	if s, ok := decoded["server"].(string); ok && s != "" {
		refs = append(refs, s)
	}
	out, ok := decoded["outbound"]
	if !ok {
		return refs
	}
	switch v := out.(type) {
	case string:
		if v != "" {
			refs = append(refs, v)
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				refs = append(refs, s)
			}
		}
	}
	return refs
}
