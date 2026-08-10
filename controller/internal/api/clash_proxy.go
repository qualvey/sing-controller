package api

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/sagernet/sing-box/option"
)

var errClashAPIUnavailable = errors.New("clash API 不可用：settings 未配置 clash_api 且 sing-box 配置未启用 experimental.clash_api")

// clashProxy 把 /api/clash/* 反向代理到 sing-box 的 clash API
// （experimental.clash_api.external_controller），自动注入 secret。
// 前端同源访问：无跨域、secret 不落浏览器。
type clashProxy struct {
	mu      sync.RWMutex
	target  *url.URL
	secret  string
	proxy   *httputil.ReverseProxy
	lastSig string // 目标签名（settings/配置变化时重建）
}

func (h *Handler) clashTarget() (*url.URL, string, string) {
	// 1. controller settings 显式配置优先
	if v := h.settings.Values(); v.ClashAPI != nil && v.ClashAPI.Address != "" {
		addr := v.ClashAPI.Address
		if !strings.Contains(addr, "://") {
			addr = "http://" + addr
		}
		if u, err := url.Parse(addr); err == nil {
			return u, v.ClashAPI.Secret, "settings:" + addr
		}
	}
	// 2. 从 sing-box 配置 experimental.clash_api 推断
	if ca := h.clashAPIConfig(); ca != nil && ca.ExternalController != "" {
		addr := "http://" + ca.ExternalController
		if u, err := url.Parse(addr); err == nil {
			return u, ca.Secret, "config:" + addr
		}
	}
	return nil, "", ""
}

// clashAPIConfig 读取当前 sing-box 配置中的 experimental.clash_api 段
// （与 api 包其他 handler 一致：直接读 store.Options，配置更新时随 store 重建）
func (h *Handler) clashAPIConfig() *option.ClashAPIOptions {
	if h.store.Options.Experimental == nil {
		return nil
	}
	return h.store.Options.Experimental.ClashAPI
}

func (h *Handler) getClashProxy() *clashProxy {
	target, secret, sig := h.clashTarget()
	if target == nil {
		return nil
	}
	h.clashProxyMu.Lock()
	defer h.clashProxyMu.Unlock()
	if h.clashProxyCache != nil && h.clashProxyCache.lastSig == sig {
		return h.clashProxyCache
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		director(r)
		// 去掉 /api/clash 前缀，转发到核心的对应路径
		trimmed := strings.TrimPrefix(r.URL.Path, "/api/clash")
		if trimmed == "" {
			trimmed = "/"
		}
		r.URL.Path = trimmed
		if secret != "" {
			r.Header.Set("Authorization", "Bearer "+secret)
		}
	}
	// 后端不可达时返回明确的 502 而非空白
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeError(w, http.StatusBadGateway, err)
	}
	h.clashProxyCache = &clashProxy{target: target, secret: secret, proxy: proxy, lastSig: sig}
	return h.clashProxyCache
}

func (h *Handler) handleClashProxy(w http.ResponseWriter, r *http.Request) {
	p := h.getClashProxy()
	if p == nil {
		writeError(w, http.StatusNotFound, errClashAPIUnavailable)
		return
	}
	p.proxy.ServeHTTP(w, r)
}
