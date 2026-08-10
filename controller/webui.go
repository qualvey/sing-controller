package main

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:web/dist
var webFS embed.FS

// webHandler 提供嵌入的 webui 静态资源（与 API 同端口）。
// - /assets/* 带 hash → 长缓存；index.html 不缓存
// - 非 /api 路径 SPA fallback 到 index.html（createWebHistory 路由刷新不 404）
// - web/dist 未构建（仅 .gitkeep）时退化为 API-only 模式：根路径返回提示
func webHandler() (http.Handler, error) {
	dist, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API 路径不归静态层管（路由已先匹配 /api/*，此处兜底）
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" {
			http.NotFound(w, r)
			return
		}
		if _, err := fs.Stat(dist, "index.html"); err != nil {
			// webui 未构建：仅 API 模式
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "sing-controller API is running. Web UI not built: run `npm run build` in controller/web. See /api/status.")
			return
		}
		// 缓存头
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		// SPA fallback
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "." || name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(dist, name); err != nil {
			name = "index.html"
		}
		http.ServeFileFS(w, r, dist, name)
	}), nil
}
