package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/sagernet/sing-box/option"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badoption"
)

// DNS 管理：servers CRUD（多态 transport，registry 解码）、rules CRUD（id 存旁车 meta）、
// 基础选项（final/strategy/cache）。写操作全部走统一校验管线（box.New 干跑 + 原子写盘）。

// ensureDNS 确保 DNS 段非 nil（写操作统一入口）。
func ensureDNS(options *option.Options) *option.DNSOptions {
	if options.DNS == nil {
		options.DNS = &option.DNSOptions{}
	}
	return options.DNS
}

func dnsServerIndex(options *option.Options, tag string) int {
	dns := options.DNS
	if dns == nil {
		return -1
	}
	for i := range dns.Servers {
		if dns.Servers[i].Tag == tag {
			return i
		}
	}
	return -1
}

// dnsReferencesTag 收集所有引用 dns server tag 的位置：dns.final + dns.rules 的 server/outbound。
func dnsReferencesTag(options *option.Options, tag string) []string {
	var refs []string
	dns := options.DNS
	if dns == nil {
		return refs
	}
	if dns.Final == tag {
		refs = append(refs, "dns.final")
	}
	for i := range dns.Rules {
		if ruleReferencesDNS(&dns.Rules[i], tag) {
			refs = append(refs, "dns 规则 #"+itoa(i+1))
		}
	}
	return refs
}

// ruleReferencesDNS 检查 DNS 规则是否引用 tag：新模型 server 字段 + 旧 outbound 字段（兼容）。
func ruleReferencesDNS(rule *option.DNSRule, tag string) bool {
	content, err := json.Marshal(rule)
	if err != nil {
		return false
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return false
	}
	if s, ok := decoded["server"].(string); ok && s == tag {
		return true
	}
	out, ok := decoded["outbound"]
	if !ok {
		return false
	}
	switch v := out.(type) {
	case string:
		return v == tag
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == tag {
				return true
			}
		}
	}
	return false
}

// removeDNSRuleRefs 从 DNS 规则中移除对 tag 的引用（server + 旧 outbound 字段；force 删除用）。
func removeDNSRuleRefs(ctx context.Context, rule *option.DNSRule, tag string) error {
	content, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return err
	}
	changed := false
	if s, ok := decoded["server"].(string); ok && s == tag {
		delete(decoded, "server")
		changed = true
	}
	if out, ok := decoded["outbound"]; ok {
		switch v := out.(type) {
		case string:
			if v == tag {
				delete(decoded, "outbound")
				changed = true
			}
		case []any:
			filtered := make([]any, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); !ok || s != tag {
					filtered = append(filtered, item)
				}
			}
			if len(filtered) == 0 {
				delete(decoded, "outbound")
			} else {
				decoded["outbound"] = filtered
			}
			changed = true
		}
	}
	if !changed {
		return nil
	}
	content, err = json.Marshal(decoded)
	if err != nil {
		return err
	}
	// 解码到新对象再回写（复用对象时未出现的字段会保留旧值）
	var fresh option.DNSRule
	if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
		return err
	}
	*rule = fresh
	return nil
}

func (h *Handler) handleGetDNS(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx(r)
	h.store.AlignMeta()
	dns := h.store.Options.DNS
	if dns == nil {
		writeJSON(w, http.StatusOK, map[string]any{"servers": []any{}, "rules": []any{}, "options": map[string]any{}})
		return
	}
	servers := make([]json.RawMessage, 0, len(dns.Servers))
	for i := range dns.Servers {
		content, err := json.MarshalContext(ctx, &dns.Servers[i])
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		servers = append(servers, content)
	}
	rules := make([]map[string]any, 0, len(dns.Rules))
	for i := range dns.Rules {
		content, err := json.MarshalContext(ctx, &dns.Rules[i])
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		rules = append(rules, map[string]any{
			"id":   newDNSRuleID(&h.store.Meta, i),
			"rule": json.RawMessage(content),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"servers": servers,
		"rules":   rules,
		"options": map[string]any{
			"final":             dns.Final,
			"strategy":          dns.Strategy,
			"timeout":           dns.Timeout,
			"disable_cache":     dns.DisableCache,
			"independent_cache": dns.IndependentCache,
			"reverse_mapping":   dns.ReverseMapping,
			"client_subnet":     dns.ClientSubnet,
		},
	})
}

func (h *Handler) handleCreateDNSServer(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var server option.DNSServerOptions
	if err := json.UnmarshalContext(h.ctx(r), body, &server); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if server.Tag == "" {
		writeError(w, http.StatusBadRequest, errors.New("DNS server 必须指定 tag"))
		return
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		if dnsServerIndex(options, server.Tag) >= 0 {
			return errors.New("DNS server tag 重复: " + server.Tag)
		}
		ensureDNS(options).Servers = append(options.DNS.Servers, server)
		return nil
	})
}

func (h *Handler) handleUpdateDNSServer(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var server option.DNSServerOptions
	if err := json.UnmarshalContext(h.ctx(r), body, &server); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if server.Tag != tag {
		writeError(w, http.StatusBadRequest, errors.New("路径 tag 与 body tag 不一致"))
		return
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		index := dnsServerIndex(options, tag)
		if index < 0 {
			return errors.New("DNS server 不存在: " + tag)
		}
		options.DNS.Servers[index] = server
		return nil
	})
}

func (h *Handler) handleDeleteDNSServer(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	force := r.URL.Query().Get("force") == "true"
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		index := dnsServerIndex(options, tag)
		if index < 0 {
			return errors.New("DNS server 不存在: " + tag)
		}
		if refs := dnsReferencesTag(options, tag); len(refs) > 0 {
			if !force {
				return &GroupReferenceError{Tag: tag, References: refs}
			}
			// force：清空 final 引用 + 从规则中移除 outbound 引用
			if options.DNS.Final == tag {
				options.DNS.Final = ""
			}
			for i := range options.DNS.Rules {
				if err := removeDNSRuleRefs(h.ctx(r), &options.DNS.Rules[i], tag); err != nil {
					return err
				}
			}
		}
		options.DNS.Servers = append(options.DNS.Servers[:index], options.DNS.Servers[index+1:]...)
		return nil
	})
}

func newDNSRuleID(meta *store.Meta, index int) string {
	for len(meta.DNSRuleIDs) <= index {
		meta.DNSRuleIDs = append(meta.DNSRuleIDs, "")
	}
	if meta.DNSRuleIDs[index] == "" {
		meta.DNSRuleIDs[index] = store.NewUUID()
	}
	return meta.DNSRuleIDs[index]
}

func (h *Handler) findDNSRuleIndexByID(id string) int {
	for i, ruleID := range h.store.Meta.DNSRuleIDs {
		if ruleID == id {
			return i
		}
	}
	return -1
}

func (h *Handler) handleCreateDNSRule(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var rule option.DNSRule
	if err := json.UnmarshalContext(h.ctx(r), body, &rule); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		ensureDNS(options).Rules = append(options.DNS.Rules, rule)
		meta.DNSRuleIDs = append(meta.DNSRuleIDs, store.NewUUID())
		return nil
	})
}

func (h *Handler) handleUpdateDNSRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var rule option.DNSRule
	if err := json.UnmarshalContext(h.ctx(r), body, &rule); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findDNSRuleIndexByID(id)
		if index < 0 {
			return errors.New("DNS 规则不存在: " + id)
		}
		options.DNS.Rules[index] = rule
		return nil
	})
}

func (h *Handler) handleDeleteDNSRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findDNSRuleIndexByID(id)
		if index < 0 {
			return errors.New("DNS 规则不存在: " + id)
		}
		options.DNS.Rules = append(options.DNS.Rules[:index], options.DNS.Rules[index+1:]...)
		meta.DNSRuleIDs = append(meta.DNSRuleIDs[:index], meta.DNSRuleIDs[index+1:]...)
		return nil
	})
}

// handlePutDNSOptions 部分更新 DNS 基础选项；未提供的字段保持不变。
func (h *Handler) handlePutDNSOptions(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var patch struct {
		Final            *string                `json:"final"`
		Strategy         *option.DomainStrategy `json:"strategy"`
		Timeout          *badoption.Duration    `json:"timeout"`
		DisableCache     *bool                  `json:"disable_cache"`
		IndependentCache *bool                  `json:"independent_cache"`
		ReverseMapping   *bool                  `json:"reverse_mapping"`
	}
	if err := json.Unmarshal(body, &patch); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		dns := ensureDNS(options)
		if patch.Final != nil {
			if *patch.Final != "" && dnsServerIndex(options, *patch.Final) < 0 {
				return errors.New("dns.final 引用的 server 不存在: " + *patch.Final)
			}
			dns.Final = *patch.Final
		}
		if patch.Strategy != nil {
			dns.Strategy = *patch.Strategy
		}
		if patch.Timeout != nil {
			dns.Timeout = *patch.Timeout
		}
		if patch.DisableCache != nil {
			dns.DisableCache = *patch.DisableCache
		}
		if patch.IndependentCache != nil {
			dns.IndependentCache = *patch.IndependentCache
		}
		if patch.ReverseMapping != nil {
			dns.ReverseMapping = *patch.ReverseMapping
		}
		return nil
	})
}
