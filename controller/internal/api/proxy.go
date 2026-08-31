package api

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/sagernet/sing-box/option"
)

var (
	errClashAPIUnavailable   = errors.New("clash API 不可用：settings 未配置 clash_api 且 sing-box 配置未启用 experimental.clash_api")
	errServiceAPIUnavailable = errors.New("service API 不可用：settings 未配置 service_api 且 sing-box 配置未启用 services[type=api]")
)

// CoreStatus 描述 sing-box 内核运行时的健康状态
type CoreStatus struct {
	Alive     bool   `json:"alive"`
	Type      string `json:"type"`
	Version   string `json:"version"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

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

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			// 1. 设置目标地址（内部会自动同步 Scheme、Host，并安全重写 Out.Host）
			pr.SetURL(target)

			// 2. 去掉路由前缀（如 /api/grpc 或 /api/clash）
			trimmed := strings.TrimPrefix(pr.Out.URL.Path, prefix)
			if trimmed == "" {
				trimmed = "/"
			}
			pr.Out.URL.Path = trimmed
			pr.Out.URL.RawPath = strings.TrimPrefix(pr.Out.URL.RawPath, prefix)

			// 3. 注入认证 Token（Rewrite 会保护添加的头不被客户端篡改）
			if secret != "" {
				pr.Out.Header.Set("Authorization", "Bearer "+secret)
			}

			// 4. 标准化 X-Forwarded-* 头部
			pr.SetXForwarded()
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Warn("反向代理上游请求失败 (prefix=%s, target=%s, path=%s): %v", prefix, target.String(), r.URL.Path, err)
			writeError(w, http.StatusBadGateway, err)
		},
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
		logger.Debug("clash API 代理不可用: 未配置或未启用")
		writeError(w, http.StatusNotFound, errClashAPIUnavailable)
		return
	}
	e.proxy.ServeHTTP(w, r)
}

func checkUpstreamReachable(rawTarget string, timeout time.Duration) error {
	u, err := url.Parse(rawTarget)
	if err != nil {
		return err
	}

	host := u.Host
	// 如果未显式带端口，补全默认 HTTP/HTTPS 端口
	if _, _, err := net.SplitHostPort(host); err != nil {
		if u.Scheme == "https" {
			host = net.JoinHostPort(host, "443")
		} else {
			host = net.JoinHostPort(host, "80")
		}
	}

	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
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
		logger.Debug("service API (gRPC) 代理不可用: 未配置或未启用")
		writeError(w, http.StatusNotFound, errServiceAPIUnavailable)
		return
	}
	e.proxy.ServeHTTP(w, r)
}

// probeCoreHealth 对 upstream sing-box 内核进行实际探活
func (h *Handler) probeCoreHealth(timeout time.Duration) CoreStatus {
	// 1. 优先尝试 Clash API /version 探活（最精确，能取到具体版本与内核类型）
	if target, secret, _ := h.clashTarget(); target != nil {
		start := time.Now()
		client := &http.Client{
			Timeout: timeout,
		}
		reqURL := strings.TrimRight(target.String(), "/") + "/version"
		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err == nil {
			if secret != "" {
				req.Header.Set("Authorization", "Bearer "+secret)
			}
			resp, reqErr := client.Do(req)
			if reqErr == nil {
				defer resp.Body.Close()
				latency := time.Since(start).Milliseconds()
				if resp.StatusCode == http.StatusOK {
					body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
					var verResp struct {
						Version string `json:"version"`
					}
					if err := json.Unmarshal(body, &verResp); err == nil && verResp.Version != "" {
						return CoreStatus{
							Alive:     true,
							Type:      "sing-box",
							Version:   verResp.Version,
							LatencyMS: latency,
						}
					}
					return CoreStatus{
						Alive:     true,
						Type:      "sing-box",
						Version:   "sing-box",
						LatencyMS: latency,
					}
				} else if resp.StatusCode == http.StatusUnauthorized {
					return CoreStatus{
						Alive:     true,
						Type:      "sing-box",
						Error:     "clash API 鉴权失败 (401 Unauthorized)",
						LatencyMS: latency,
					}
				}
				return CoreStatus{
					Alive:     false,
					Type:      "sing-box",
					Error:     "clash API 返回 HTTP " + resp.Status,
					LatencyMS: latency,
				}
			}
			return CoreStatus{
				Alive:     false,
				Type:      "sing-box",
				Error:     reqErr.Error(),
				LatencyMS: time.Since(start).Milliseconds(),
			}
		}
	}

	// 2. 备选尝试 service API（gRPC）TCP 探活
	if target, _, _ := h.serviceAPITarget(); target != nil {
		start := time.Now()
		if err := checkUpstreamReachable(target.String(), timeout); err != nil {
			return CoreStatus{
				Alive:     false,
				Type:      "sing-box",
				Error:     err.Error(),
				LatencyMS: time.Since(start).Milliseconds(),
			}
		}
		return CoreStatus{
			Alive:     true,
			Type:      "sing-box",
			Version:   "sing-box",
			LatencyMS: time.Since(start).Milliseconds(),
		}
	}

	return CoreStatus{
		Alive: false,
		Type:  "unknown",
		Error: "未配置 clash_api 或 service_api",
	}
}
