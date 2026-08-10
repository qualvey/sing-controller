package api

import (
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/sagernet/sing-box/option"
)

var (
	errClashAPIUnavailable = errors.New("clash API 不可用：settings 未配置 clash_api 且 sing-box 配置未启用 experimental.clash_api")
	errServiceAPIUnavailable = errors.New("service API 不可用：settings 未配置 service_api 且 sing-box 配置未启用 services[type=api]")
)

// reverseProxyEntry 一个反向代理目标（clash API / service API 共用）
type reverseProxyEntry struct {
	target  *url.URL
	secret  string
	proxy   *httputil.ReverseProxy
	lastSig string // 目标签名（settings/配置变化时重建）
}

// proxyCache 按前缀缓存反向代理实例
type proxyCache struct {
	mu    sync.Mutex
	items map[string]*reverseProxyEntry
}

func newProxyCache() *proxyCache { return &proxyCache{items: map[string]*reverseProxyEntry{}} }

// get 返回指定前缀的代理；resolve 返回 (target, secret, 签名)，nil target 表示未配置
func (c *proxyCache) get(prefix string, resolve func() (*url.URL, string, string)) *reverseProxyEntry {
	target, secret, sig := resolve()
	if target == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[prefix]; ok && e.lastSig == sig {
		return e
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		director(r)
		// 去掉前缀（如 /api/clash、/api/grpc），转发到核心对应路径
		trimmed := strings.TrimPrefix(r.URL.Path, prefix)
		if trimmed == "" {
			trimmed = "/"
		}
		r.URL.Path = trimmed
		if secret != "" {
			r.Header.Set("Authorization", "Bearer "+secret)
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeError(w, http.StatusBadGateway, err)
	}
	e := &reverseProxyEntry{target: target, secret: secret, proxy: proxy, lastSig: sig}
	c.items[prefix] = e
	return e
}

func normalizeAddr(addr string) string {
	if !strings.Contains(addr, "://") {
		return "http://" + addr
	}
	return addr
}

// ============ clash API（experimental.clash_api） ============

func (h *Handler) clashTarget() (*url.URL, string, string) {
	// 1. controller settings 显式配置优先
	if v := h.settings.Values(); v.ClashAPI != nil && v.ClashAPI.Address != "" {
		addr := normalizeAddr(v.ClashAPI.Address)
		if u, err := url.Parse(addr); err == nil {
			return u, v.ClashAPI.Secret, "settings:" + addr
		}
	}
	// 2. 从 sing-box 配置 experimental.clash_api 推断
	if ca := h.clashAPIConfig(); ca != nil && ca.ExternalController != "" {
		addr := normalizeAddr(ca.ExternalController)
		if u, err := url.Parse(addr); err == nil {
			return u, ca.Secret, "config:" + addr
		}
	}
	return nil, "", ""
}

func (h *Handler) clashAPIConfig() *option.ClashAPIOptions {
	if h.store.Options.Experimental == nil {
		return nil
	}
	return h.store.Options.Experimental.ClashAPI
}

func (h *Handler) handleClashProxy(w http.ResponseWriter, r *http.Request) {
	e := h.proxies.get("/api/clash", h.clashTarget)
	if e == nil {
		writeError(w, http.StatusNotFound, errClashAPIUnavailable)
		return
	}
	e.proxy.ServeHTTP(w, r)
}

// ============ service API（顶层 services[type=api]，gRPC-Web / WS） ============

func (h *Handler) serviceAPITarget() (*url.URL, string, string) {
	// 1. controller settings 显式配置优先
	if v := h.settings.Values(); v.ServiceAPI != nil && v.ServiceAPI.Address != "" {
		addr := normalizeAddr(v.ServiceAPI.Address)
		if u, err := url.Parse(addr); err == nil {
			return u, v.ServiceAPI.Secret, "settings:" + addr
		}
	}
	// 2. 从 sing-box 配置顶层 services 段推断（type=api）
	if opt := h.serviceAPIConfig(); opt != nil {
		host := ""
		if opt.Listen != nil {
			host = opt.Listen.Build(netip.IPv4Unspecified()).String()
		}
		port := opt.ListenPort
		if host == "" {
			host = "127.0.0.1"
		}
		// "::" / "0.0.0.0" 等任意地址 → 用回环地址（controller 与 sing-box 同机）
		switch host {
		case "::", "0.0.0.0", "":
			host = "127.0.0.1"
		}
		addr := "http://" + net.JoinHostPort(host, strconv.Itoa(int(port)))
		if u, err := url.Parse(addr); err == nil {
			return u, opt.Secret, "config:" + addr
		}
	}
	return nil, "", ""
}

func (h *Handler) serviceAPIConfig() *option.APIServiceOptions {
	for i := range h.store.Options.Services {
		s := &h.store.Options.Services[i]
		if s.Type == "api" {
			if opt, ok := s.Options.(*option.APIServiceOptions); ok {
				return opt
			}
		}
	}
	return nil
}

func (h *Handler) handleServiceProxy(w http.ResponseWriter, r *http.Request) {
	e := h.proxies.get("/api/grpc", h.serviceAPITarget)
	if e == nil {
		writeError(w, http.StatusNotFound, errServiceAPIUnavailable)
		return
	}
	e.proxy.ServeHTTP(w, r)
}
