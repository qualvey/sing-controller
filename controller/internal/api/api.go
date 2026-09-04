// Package api 提供 RESTful 配置管理 API。
// 设计原则：所有写操作走 store.Update 校验管线（解码 → box.New 干跑 → 原子写盘），
// 校验失败一律不落盘、内存回滚。
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/provider"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/qualvey/sing-controller/internal/supervisor"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/schema"
)

type HandlerOptions struct {
	Store       *store.Store
	Settings    *settings.Manager
	Supervisor  *supervisor.Manager
	ProviderMgr *provider.Manager
	Secret      string
	Version     string
	// Static 嵌入的 webui 静态资源（SPA）；nil 时根路径返回服务信息 JSON（API-only 模式）
	Static http.Handler
}

type Handler struct {
	store       *store.Store
	settings    *settings.Manager
	supervisor  *supervisor.Manager
	providerMgr *provider.Manager
	secret      string
	version     string
	static      http.Handler

	schemaOnce sync.Once
	schemaJSON []byte
	schemaErr  error

	proxies *proxyCache
}

// metaType 别名，简化 handler 签名书写
type metaType = store.Meta

func NewHandler(opts HandlerOptions) http.Handler {
	logger.Debug("New handler initing")
	h := &Handler{
		store:       opts.Store,
		settings:    opts.Settings,
		supervisor:  opts.Supervisor,
		providerMgr: opts.ProviderMgr,
		secret:      opts.Secret,
		version:     opts.Version,
		static:      opts.Static,
		proxies:     newProxyCache(),
	}
	mux := http.NewServeMux()

	// 根路径：嵌入的 webui（同端口）；未构建时返回服务信息
	if opts.Static != nil {
		mux.Handle("/", opts.Static)
	} else {
		mux.HandleFunc("/", h.handleRoot)
	}
	mux.HandleFunc("GET /healthz", h.handleHealthz)

	// 状态 / 配置 / 设置
	mux.HandleFunc("GET /api/status", h.handleStatus)
	mux.HandleFunc("GET /api/config", h.handleGetConfig)
	mux.HandleFunc("PUT /api/config", h.handlePutConfig)
	mux.HandleFunc("GET /api/config/raw", h.handleGetConfigRaw)
	mux.HandleFunc("PUT /api/config/raw", h.handlePutConfigRaw)
	mux.HandleFunc("GET /api/schema", h.handleSchema)
	mux.HandleFunc("GET /api/types", h.handleTypes)
	mux.HandleFunc("GET /api/settings", h.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", h.handlePutSettings)
	mux.HandleFunc("GET /api/ports/available", h.handlePortAvailable)

	// core supervisor 控制器（进程启停 / 状态 / 日志）
	mux.HandleFunc("GET /api/core/status", h.handleCoreStatus)
	mux.HandleFunc("POST /api/core/start", h.handleCoreStart)
	mux.HandleFunc("POST /api/core/stop", h.handleCoreStop)
	mux.HandleFunc("POST /api/core/restart", h.handleCoreRestart)
	mux.HandleFunc("GET /api/core/logs", h.handleCoreLogs)
	mux.HandleFunc("GET /api/core/logs/stream", h.handleCoreLogsStream)

	// Provider & Profile (节点订阅源与配置组)
	mux.HandleFunc("GET /api/providers", h.handleListProviders)
	mux.HandleFunc("POST /api/providers", h.handleCreateProvider)
	mux.HandleFunc("GET /api/providers/{id}", h.handleGetProvider)
	mux.HandleFunc("PUT /api/providers/{id}", h.handleUpdateProvider)
	mux.HandleFunc("DELETE /api/providers/{id}", h.handleDeleteProvider)
	mux.HandleFunc("POST /api/providers/{id}/refresh", h.handleRefreshProvider)
	mux.HandleFunc("GET /api/providers/{tag}/nodes", h.handleGetProviderNodes)

	mux.HandleFunc("GET /api/profiles", h.handleListProfiles)
	mux.HandleFunc("POST /api/profiles", h.handleCreateProfile)
	mux.HandleFunc("GET /api/profiles/{id}", h.handleGetProfile)
	mux.HandleFunc("PUT /api/profiles/{id}", h.handleUpdateProfile)
	mux.HandleFunc("DELETE /api/profiles/{id}", h.handleDeleteProfile)
	mux.HandleFunc("POST /api/profiles/{id}/apply", h.handleApplyProfile)

	// 用户池（全局用户，多对多绑定入站）
	mux.HandleFunc("GET /api/users", h.handleListUsers)
	mux.HandleFunc("POST /api/users", h.handleCreateUser)
	mux.HandleFunc("PUT /api/users/{name}", h.handleUpdateUser)
	mux.HandleFunc("DELETE /api/users/{name}", h.handleDeleteUser)

	// clash API ??(?? /api/clash/* ?? sing-box external_controller,?? secret)
	mux.HandleFunc("/api/clash/", h.handleClashProxy)
	// service API ??（?? /api/grpc/* ?? services[type=api],gRPC-Web / WS）
	mux.HandleFunc("/api/grpc/", h.handleServiceProxy)

	// 系统与权限工具
	mux.HandleFunc("POST /api/system/elevate", h.handleSystemElevate)
	mux.HandleFunc("POST /api/system/disable-tun", h.handleSystemDisableTun)

	// 工具
	mux.HandleFunc("POST /api/tools/uuid", h.handleToolUUID)
	mux.HandleFunc("POST /api/tools/reality-keypair", h.handleToolRealityKeypair)
	mux.HandleFunc("POST /api/tools/password", h.handleToolPassword)
	mux.HandleFunc("POST /api/tools/parse-json", h.handleToolParseJSON)
	mux.HandleFunc("POST /api/tools/parse-node", h.handleToolParseNode)

	// outbound CRUD
	mux.HandleFunc("GET /api/outbounds", h.handleListOutbounds)
	mux.HandleFunc("POST /api/outbounds", h.handleCreateOutbound)
	mux.HandleFunc("GET /api/outbounds/{tag}", h.handleGetOutbound)
	mux.HandleFunc("PUT /api/outbounds/{tag}", h.handleUpdateOutbound)
	mux.HandleFunc("DELETE /api/outbounds/{tag}", h.handleDeleteOutbound)

	// inbound CRUD
	mux.HandleFunc("GET /api/inbounds", h.handleListInbounds)
	mux.HandleFunc("POST /api/inbounds", h.handleCreateInbound)
	mux.HandleFunc("GET /api/inbounds/{tag}", h.handleGetInbound)
	mux.HandleFunc("PUT /api/inbounds/{tag}", h.handleUpdateInbound)
	mux.HandleFunc("DELETE /api/inbounds/{tag}", h.handleDeleteInbound)

	// route rule CRUD（简单规则，id 存于 meta 旁车）
	mux.HandleFunc("GET /api/routes", h.handleListRoutes)
	mux.HandleFunc("POST /api/routes", h.handleCreateRoute)
	mux.HandleFunc("GET /api/routes/{id}", h.handleGetRoute)
	mux.HandleFunc("PUT /api/routes/{id}", h.handleUpdateRoute)
	mux.HandleFunc("DELETE /api/routes/{id}", h.handleDeleteRoute)
	mux.HandleFunc("PUT /api/routes/{id}/toggle", h.handleToggleRoute)
	mux.HandleFunc("PUT /rules/{id}", h.handleToggleRoute)

	// DNS 管理
	mux.HandleFunc("GET /api/dns", h.handleGetDNS)
	mux.HandleFunc("POST /api/dns/servers", h.handleCreateDNSServer)
	mux.HandleFunc("PUT /api/dns/servers/{tag}", h.handleUpdateDNSServer)
	mux.HandleFunc("DELETE /api/dns/servers/{tag}", h.handleDeleteDNSServer)
	mux.HandleFunc("POST /api/dns/rules", h.handleCreateDNSRule)
	mux.HandleFunc("PUT /api/dns/rules/{id}", h.handleUpdateDNSRule)
	mux.HandleFunc("DELETE /api/dns/rules/{id}", h.handleDeleteDNSRule)
	mux.HandleFunc("PUT /api/dns/options", h.handlePutDNSOptions)
	mux.HandleFunc("POST /api/dns/lookup", h.handleDNSLookup)

	// 诊断
	mux.HandleFunc("GET /api/diagnostics", h.handleDiagnostics)

	// 规则集（route.rule_set 段）
	mux.HandleFunc("GET /api/rule-sets", h.handleListRuleSets)
	mux.HandleFunc("POST /api/rule-sets", h.handleCreateRuleSet)
	mux.HandleFunc("PUT /api/rule-sets/{id}", h.handleUpdateRuleSet)
	mux.HandleFunc("DELETE /api/rule-sets/{id}", h.handleDeleteRuleSet)

	// 证书
	mux.HandleFunc("GET /api/certificate", h.handleGetCertificate)
	mux.HandleFunc("PUT /api/certificate", h.handlePutCertificate)
	mux.HandleFunc("POST /api/certificate/providers", h.handleCreateCertProvider)
	mux.HandleFunc("PUT /api/certificate/providers/{id}", h.handleUpdateCertProvider)
	mux.HandleFunc("DELETE /api/certificate/providers/{id}", h.handleDeleteCertProvider)

	// 重载 sing-box
	mux.HandleFunc("POST /api/reload", h.handleReload)

	return h.withMiddleware(mux)
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *statusResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

func (rw *statusResponseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func (rw *statusResponseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *Handler) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// CORS（webui dev server 跨端口与桌面端直连）
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Secret, Authorization, Connect-Protocol-Version, X-User-Agent, X-Grpc-Web")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		defer func() {
			duration := time.Since(start)
			status := wrapped.statusCode
			ip := clientIP(r)
			path := r.URL.Path
			if r.URL.RawQuery != "" {
				path += "?" + r.URL.RawQuery
			}

			// 对健康检查接口在 Debug 级别输出，避免高频刷屏
			if r.URL.Path == "/healthz" {
				logger.Debug("[HTTP] %s %s %d (%s) - %s %dB", r.Method, path, status, duration, ip, wrapped.bytesWritten)
				return
			}

			if status >= 500 {
				logger.Error("[HTTP] %s %s %d (%s) - %s %dB", r.Method, path, status, duration, ip, wrapped.bytesWritten)
			} else if status >= 400 {
				logger.Warn("[HTTP] %s %s %d (%s) - %s %dB", r.Method, path, status, duration, ip, wrapped.bytesWritten)
			} else {
				logger.Info("[HTTP] %s %s %d (%s) - %s %dB", r.Method, path, status, duration, ip, wrapped.bytesWritten)
			}
		}()

		// 鉴权
		if h.secret != "" {
			if r.Header.Get("X-Secret") != h.secret && r.URL.Query().Get("token") != h.secret {
				logger.Warn("鉴权失败: IP=%s Path=%s", clientIP(r), r.URL.Path)
				writeError(wrapped, http.StatusUnauthorized, errors.New("invalid secret"))
				return
			}
		}
		next.ServeHTTP(wrapped, r)
	})
}

// ---------- root / health ----------

func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	values := h.settings.Values()
	version := h.version
	if version == "" {
		version = "dev"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "sing-controller",
		"version": version,
		"api":     "/api",
		"status":  "/api/status",
		"schema":  "/api/schema",
		"listen":  values.Server.Addr(),
		"config":  h.store.Path(),
		"webui":   "独立部署（web/ 目录，npm run dev 或构建后静态托管）",
	})
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// ---------- helpers ----------

func (h *Handler) ctx(r *http.Request) context.Context {
	return include.Context(r.Context())
}

func (h *Handler) schema() ([]byte, error) {
	h.schemaOnce.Do(func() {
		h.schemaJSON, h.schemaErr = schema.Generate(include.Context(context.Background()), reflect.TypeFor[option.Options]())
	})
	return h.schemaJSON, h.schemaErr
}

// commit 执行"内存修改 → 全量校验 → 原子写盘"管线。
func (h *Handler) commit(w http.ResponseWriter, r *http.Request, mutate func(*option.Options, *store.Meta) error) {
	if err := h.store.Update(h.ctx(r), mutate); err != nil {
		logger.Warn("配置更新提交失败 (path=%s): %v", r.URL.Path, err)
		var refErr *GroupReferenceError
		if errors.As(err, &refErr) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":      refErr.Error(),
				"references": refErr.References,
			})
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	logger.Info("配置更新已原子落盘 (path=%s)", r.URL.Path)
	response := map[string]any{"saved": true}
	// 保存后自动重载 sing-box（settings.reload.after_save 或 ?reload=true）
	if executed, err := h.reloadIfRequested(r); err != nil {
		logger.Error("配置保存后自动重载 sing-box 失败: %v", err)
		response["reload_error"] = err.Error()
		response["message"] = "配置已保存，但 sing-box 重载失败"
	} else if executed {
		logger.Info("配置保存后自动重载 sing-box 成功")
		response["reloaded"] = true
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

// GroupReferenceError 表示对象被 selector/urltest 组引用：
// 删除需前端确认，确认后带 force=true 重试（后端自动从引用组拔除 tag）。
// commit 会将此类错误转成 409 + references 列表，供前端弹确认框。
type GroupReferenceError struct {
	Tag        string
	References []string
}

func (e *GroupReferenceError) Error() string {
	return "outbound 被组引用: " + strings.Join(e.References, ", ")
}

func readRawBody(r *http.Request) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, errors.New("empty body")
	}
	return content, nil
}

// ---------- status ----------

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	values := h.settings.Values()
	ruleCount := 0
	if h.store.Options.Route != nil {
		ruleCount = len(h.store.Options.Route.Rules)
	}
	coreStatus := h.probeCoreHealth(1 * time.Second)
	tunDenied := false
	if h.supervisor != nil {
		tunDenied = h.supervisor.TunAccessDenied()
	}

	logLevel := "info"
	if values.Log != nil && values.Log.Level != "" {
		logLevel = values.Log.Level
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"config_path":        h.store.Path(),
		"config_exists":      h.store.Exists(),
		"controller_config":  h.settings.Path(),
		"listen":             values.Server.Addr(),
		"log_level":          logLevel,
		"inbounds":           len(h.store.Options.Inbounds),
		"outbounds":          len(h.store.Options.Outbounds),
		"rules":              ruleCount,
		"core":               coreStatus,
		"core_alive":         coreStatus.Alive,
		"core_version":       coreStatus.Version,
		"core_type":          coreStatus.Type,
		"core_error":         coreStatus.Error,
		"tun_access_denied":  tunDenied,
	})
}

// handleSystemElevate 触发 UAC 提权并以管理员身份重启 controller 进程
func (h *Handler) handleSystemElevate(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "windows" {
		writeError(w, http.StatusBadRequest, errors.New("仅 Windows 系统支持 UAC 提权"))
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("获取程序路径失败: %w", err))
		return
	}

	args := os.Args[1:]
	argStr := strings.Join(args, " ")

	psCmd := fmt.Sprintf("Start-Process -FilePath '%s' -ArgumentList '%s' -Verb RunAs", exePath, argStr)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
	if err := cmd.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("启动提权进程失败: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "已成功拉起管理员提权窗口",
	})

	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
}

// handleSystemDisableTun 移除主配置中的 TUN 入站并自动重启内核
func (h *Handler) handleSystemDisableTun(w http.ResponseWriter, r *http.Request) {
	if h.providerMgr == nil {
		writeError(w, http.StatusInternalServerError, errors.New("provider manager 未就绪"))
		return
	}

	if err := h.providerMgr.DisableTun(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("禁用 TUN 模式失败: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "已成功移除 TUN 模式并重新加载配置",
	})
}

// ---------- types / schema ----------

func (h *Handler) handleTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"inbounds":  include.InboundRegistry().OptionTypes(),
		"outbounds": include.OutboundRegistry().OptionTypes(),
		"endpoints": include.EndpointRegistry().OptionTypes(),
		"services":  include.ServiceRegistry().OptionTypes(),
	})
}

func (h *Handler) handleSchema(w http.ResponseWriter, r *http.Request) {
	content, err := h.schema()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

// ---------- settings ----------

func (h *Handler) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.settings.Values())
}

func (h *Handler) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var newValues settings.Settings
	if err := json.Unmarshal(body, &newValues); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var privilegedPortWarning string
	if newValues.Core.ConfigPath == "" {
		writeError(w, http.StatusBadRequest, errors.New("core.config_path 不能为空"))
		return
	}
	if newValues.Server.ListenPort > 0 && newValues.Server.ListenPort < 1024 {
		privilegedPortWarning = fmt.Sprintf("端口 %d 是特权端口（<1024）。deb 部署已支持（AmbientCapabilities），本机开发直跑需 root 或加 capabilities。", newValues.Server.ListenPort)
	}
	if newValues.Log != nil {
		if newValues.Log.Level != "" {
			if _, err := log.ParseLevel(newValues.Log.Level); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
		}
		// 动态调整 logger 实时生效
		logger.Init(newValues.Log.Level, newValues.Log.Output, newValues.Log.Outfile)
	}
	if newValues.Core.Reload == nil {
		newValues.Core.Reload = &settings.ReloadOptions{Mode: "auto"}
	}
	if err := newValues.Core.Reload.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx := h.ctx(r)
	if err := h.settings.Update(func(s *settings.Settings) error {
		*s = newValues
		return nil
	}); err != nil {
		logger.Error("更新 controller 设置失败: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Info("Controller 设置已成功更新 (listen=%s, log_level=%s, config_path=%s)",
		newValues.Server.Addr(), logger.GetLevel().String(), newValues.Core.ConfigPath)
	// config 路径变更 → 切换主配置存储
	if h.store.Path() != newValues.Core.ConfigPath {
		logger.Info("切换 sing-box 配置文件路径: %s -> %s", h.store.Path(), newValues.Core.ConfigPath)
		h.store.SetPath(newValues.Core.ConfigPath)
		if err := h.store.Load(ctx, store.DefaultConfig{
			InboundType: "mixed",
			Listen:      "127.0.0.1",
			ListenPort:  2080,
		}); err != nil {
			logger.Error("加载新 sing-box 主配置失败: %v", err)
			writeJSON(w, http.StatusOK, map[string]any{
				"saved":      true,
				"load_error": err.Error(),
				"message":    "controller 配置已保存，但新主配置路径加载失败",
			})
			return
		}
	}
	if privilegedPortWarning != "" {
		writeJSON(w, http.StatusOK, map[string]any{"saved": true, "warning": privilegedPortWarning})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": true})
}

// ---------- config ----------

func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	content, err := h.store.Content(h.ctx(r))
	if err != nil {
		logger.Error("获取 sing-box 格式化配置失败: %v", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Debug("读取 sing-box 格式化配置成功 (bytes=%d)", len(content))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

// handleGetConfigRaw 返回主配置文件原始内容（保留注释/格式/字段顺序）。
func (h *Handler) handleGetConfigRaw(w http.ResponseWriter, r *http.Request) {
	content := h.store.RawContent()
	if len(content) == 0 {
		content = []byte("{}\n")
	}
	logger.Debug("读取 sing-box 原始配置成功 (bytes=%d)", len(content))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

// handlePutConfigRaw 原样保存配置文本（注释/格式保留）：sing-box 解析 + 干跑校验通过后写盘。
func (h *Handler) handlePutConfigRaw(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(bytes.TrimSpace(body)) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("配置内容为空"))
		return
	}
	logger.Info("尝试原样保存 sing-box 配置 (bytes=%d)", len(body))
	if err := h.store.RawSave(h.ctx(r), body); err != nil {
		logger.Warn("原样保存 sing-box 配置校验或写盘失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	logger.Info("原样保存 sing-box 配置成功")
	response := map[string]any{"saved": true}
	// 保存后自动重载 sing-box（settings.reload.after_save 或 ?reload=true）
	if executed, err := h.reloadIfRequested(r); err != nil {
		logger.Error("保存后自动重载 sing-box 失败: %v", err)
		response["reload_error"] = err.Error()
		response["message"] = "配置已保存，但 sing-box 重载失败"
	} else if executed {
		logger.Info("保存后自动重载 sing-box 成功")
		response["reloaded"] = true
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	logger.Info("尝试全量更新 sing-box 配置 (bytes=%d)", len(body))
	// 先完整校验，再落盘
	if err := store.Validate(h.ctx(r), body); err != nil {
		logger.Warn("全量更新 sing-box 配置校验失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		optionsValue, err := store.Parse(h.ctx(r), body)
		if err != nil {
			return err
		}
		*options = optionsValue
		meta.RuleIDs = make([]string, len(optionsValue.Route.Rules))
		for i := range meta.RuleIDs {
			meta.RuleIDs[i] = store.NewUUID()
		}
		return nil
	})
}
