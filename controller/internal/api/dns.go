package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing-box/option"
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

	h.store.RLock()
	defer h.store.RUnlock()

	dns := h.store.Options.DNS
	if dns == nil {
		logger.Debug("查询 DNS 配置成功 (未配置 DNS 段)")
		writeJSON(w, http.StatusOK, map[string]any{"servers": []any{}, "rules": []any{}, "options": map[string]any{}})
		return
	}
	servers := make([]json.RawMessage, 0, len(dns.Servers))
	for i := range dns.Servers {
		content, err := json.MarshalContext(ctx, &dns.Servers[i])
		if err != nil {
			logger.Error("序列化 DNS server [%s] 失败: %v", dns.Servers[i].Tag, err)
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		servers = append(servers, content)
	}
	rules := make([]map[string]any, 0, len(dns.Rules))
	for i := range dns.Rules {
		content, err := json.MarshalContext(ctx, &dns.Rules[i])
		if err != nil {
			logger.Error("序列化 DNS 规则 #%d 失败: %v", i+1, err)
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		rules = append(rules, map[string]any{
			"id":   newDNSRuleID(&h.store.Meta, i),
			"rule": json.RawMessage(content),
		})
	}
	logger.Debug("查询 DNS 配置成功 (servers=%d, rules=%d)", len(dns.Servers), len(dns.Rules))
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
		logger.Warn("解析创建 DNS server 请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if server.Tag == "" {
		writeError(w, http.StatusBadRequest, errors.New("DNS server 必须指定 tag"))
		return
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		if dnsServerIndex(options, server.Tag) >= 0 {
			logger.Warn("创建 DNS server 失败: tag 已存在 [%s]", server.Tag)
			return errors.New("DNS server tag 重复: " + server.Tag)
		}
		ensureDNS(options).Servers = append(options.DNS.Servers, server)
		logger.Info("成功创建 DNS server [tag: %s, type: %s]", server.Tag, server.Type)
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
		logger.Warn("解析更新 DNS server [%s] 请求体失败: %v", tag, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if server.Tag == "" {
		server.Tag = tag
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		index := dnsServerIndex(options, tag)
		if index < 0 {
			logger.Warn("更新 DNS server 失败: tag 不存在 [%s]", tag)
			return errors.New("DNS server 不存在: " + tag)
		}
		if server.Tag != tag {
			if dnsServerIndex(options, server.Tag) >= 0 {
				logger.Warn("更新 DNS server 失败: 新 tag 已存在 [%s]", server.Tag)
				return errors.New("DNS server tag 已存在: " + server.Tag)
			}
			if options.DNS != nil {
				if options.DNS.Final == tag {
					options.DNS.Final = server.Tag
				}
				for i := range options.DNS.Rules {
					_ = renameServerInDNSRule(h.ctx(r), &options.DNS.Rules[i], tag, server.Tag)
				}
			}
		}
		options.DNS.Servers[index] = server
		logger.Info("更新 DNS server 成功 [oldTag: %s, newTag: %s, type: %s]", tag, server.Tag, server.Type)
		return nil
	})
}

func (h *Handler) handleDeleteDNSServer(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	force := r.URL.Query().Get("force") == "true"
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		index := dnsServerIndex(options, tag)
		if index < 0 {
			logger.Warn("删除 DNS server 失败: tag 不存在 [%s]", tag)
			return errors.New("DNS server 不存在: " + tag)
		}
		if refs := dnsReferencesTag(options, tag); len(refs) > 0 {
			if !force {
				logger.Warn("删除 DNS server 存在引用阻止 [%s]: refs=%v", tag, refs)
				return &GroupReferenceError{Tag: tag, References: refs}
			}
			logger.Info("强制删除 DNS server [%s]: 清理引用 %v", tag, refs)
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
		logger.Info("成功删除 DNS server [tag: %s, force=%v]", tag, force)
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
		logger.Warn("解析创建 DNS 规则请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		newID := store.NewUUID()
		ensureDNS(options).Rules = append(options.DNS.Rules, rule)
		meta.DNSRuleIDs = append(meta.DNSRuleIDs, newID)
		logger.Info("成功创建 DNS 规则 [id=%s, rules_count=%d]", newID, len(options.DNS.Rules))
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
		logger.Warn("解析更新 DNS 规则 [%s] 请求体失败: %v", id, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findDNSRuleIndexByID(id)
		if index < 0 {
			logger.Warn("更新 DNS 规则失败: id 不存在 [%s]", id)
			return errors.New("DNS 规则不存在: " + id)
		}
		options.DNS.Rules[index] = rule
		logger.Info("更新 DNS 规则成功 [id=%s]", id)
		return nil
	})
}

func (h *Handler) handleDeleteDNSRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findDNSRuleIndexByID(id)
		if index < 0 {
			logger.Warn("删除 DNS 规则失败: id 不存在 [%s]", id)
			return errors.New("DNS 规则不存在: " + id)
		}
		options.DNS.Rules = append(options.DNS.Rules[:index], options.DNS.Rules[index+1:]...)
		meta.DNSRuleIDs = append(meta.DNSRuleIDs[:index], meta.DNSRuleIDs[index+1:]...)
		logger.Info("成功删除 DNS 规则 [id=%s]", id)
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
		logger.Warn("解析更新 DNS 选项请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		dns := ensureDNS(options)
		if patch.Final != nil {
			if *patch.Final != "" && dnsServerIndex(options, *patch.Final) < 0 {
				logger.Warn("更新 DNS 选项失败: dns.final 引用不存在 [%s]", *patch.Final)
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
		logger.Info("成功更新 DNS 基础选项")
		return nil
	})
}

type dnsCacheEntry struct {
	ip        string
	expiresAt time.Time
}

var (
	dnsLookupCache   = make(map[string]dnsCacheEntry)
	dnsLookupCacheMu sync.RWMutex
)

func getCachedDNS(domain string) (string, bool) {
	dnsLookupCacheMu.RLock()
	defer dnsLookupCacheMu.RUnlock()
	entry, ok := dnsLookupCache[domain]
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.ip, true
	}
	return "", false
}

func setCachedDNS(domain, ip string, ttl time.Duration) {
	dnsLookupCacheMu.Lock()
	defer dnsLookupCacheMu.Unlock()
	dnsLookupCache[domain] = dnsCacheEntry{
		ip:        ip,
		expiresAt: time.Now().Add(ttl),
	}
}

// handleDNSLookup 并发批量解析域名（前端地球仪/工具查询用）
// POST /api/dns/lookup { "domains": ["google.com", "example.com"] }
// 返回: { "results": { "google.com": "142.250.190.46", ... } }
func (h *Handler) handleDNSLookup(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req struct {
		Domains []string `json:"domains"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if len(req.Domains) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"results": map[string]string{}})
		return
	}

	// 限制单次批量查询数量，避免超大包
	if len(req.Domains) > 100 {
		req.Domains = req.Domains[:100]
	}

	results := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	for _, rawDomain := range req.Domains {
		domain := strings.TrimSpace(rawDomain)
		if domain == "" {
			continue
		}
		// 去掉末尾的点，如 google.com. -> google.com
		domain = strings.TrimSuffix(domain, ".")

		// 如果已经是 IP，直接返回
		if ip := net.ParseIP(domain); ip != nil {
			mu.Lock()
			results[rawDomain] = domain
			mu.Unlock()
			continue
		}

		// 查缓存
		if ip, ok := getCachedDNS(domain); ok {
			mu.Lock()
			results[rawDomain] = ip
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(d, original string) {
			defer wg.Done()

			addrs, err := net.DefaultResolver.LookupIPAddr(ctx, d)
			if err != nil || len(addrs) == 0 {
				return
			}

			// 优先选 IPv4
			var chosen string
			for _, addr := range addrs {
				if addr.IP.To4() != nil {
					chosen = addr.IP.String()
					break
				}
			}
			if chosen == "" && len(addrs) > 0 {
				chosen = addrs[0].IP.String()
			}

			if chosen != "" {
				setCachedDNS(d, chosen, 10*time.Minute)
				mu.Lock()
				results[original] = chosen
				mu.Unlock()
			}
		}(domain, rawDomain)
	}

	wg.Wait()

	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}

