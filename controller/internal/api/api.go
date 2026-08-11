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
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/schema"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
)

type HandlerOptions struct {
	Store    *store.Store
	Settings *settings.Manager
	Secret   string
	Version  string
	// Static 嵌入的 webui 静态资源（SPA）；nil 时根路径返回服务信息 JSON（API-only 模式）
	Static http.Handler
}

type Handler struct {
	store    *store.Store
	settings *settings.Manager
	secret   string
	version  string
	static   http.Handler

	schemaOnce sync.Once
	schemaJSON []byte
	schemaErr  error

	proxies *proxyCache
}

// metaType 别名，简化 handler 签名书写
type metaType = store.Meta

func NewHandler(opts HandlerOptions) http.Handler {
	h := &Handler{
		store:    opts.Store,
		settings: opts.Settings,
		secret:   opts.Secret,
		version:  opts.Version,
		static:   opts.Static,
		proxies:  newProxyCache(),
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

	// 用户池（全局用户，多对多绑定入站）
	mux.HandleFunc("GET /api/users", h.handleListUsers)
	mux.HandleFunc("POST /api/users", h.handleCreateUser)
	mux.HandleFunc("PUT /api/users/{name}", h.handleUpdateUser)
	mux.HandleFunc("DELETE /api/users/{name}", h.handleDeleteUser)

	// clash API ??(?? /api/clash/* ?? sing-box external_controller,?? secret)
	mux.HandleFunc("/api/clash/", h.handleClashProxy)
	// service API ??（?? /api/grpc/* ?? services[type=api],gRPC-Web / WS）
	mux.HandleFunc("/api/grpc/", h.handleServiceProxy)

	// 工具
	mux.HandleFunc("POST /api/tools/uuid", h.handleToolUUID)
	mux.HandleFunc("POST /api/tools/reality-keypair", h.handleToolRealityKeypair)
	mux.HandleFunc("POST /api/tools/password", h.handleToolPassword)
	mux.HandleFunc("POST /api/tools/parse-json", h.handleToolParseJSON)

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
	mux.HandleFunc("PUT /api/routes/{id}", h.handleUpdateRoute)
	mux.HandleFunc("DELETE /api/routes/{id}", h.handleDeleteRoute)

	// DNS 管理
	mux.HandleFunc("GET /api/dns", h.handleGetDNS)
	mux.HandleFunc("POST /api/dns/servers", h.handleCreateDNSServer)
	mux.HandleFunc("PUT /api/dns/servers/{tag}", h.handleUpdateDNSServer)
	mux.HandleFunc("DELETE /api/dns/servers/{tag}", h.handleDeleteDNSServer)
	mux.HandleFunc("POST /api/dns/rules", h.handleCreateDNSRule)
	mux.HandleFunc("PUT /api/dns/rules/{id}", h.handleUpdateDNSRule)
	mux.HandleFunc("DELETE /api/dns/rules/{id}", h.handleDeleteDNSRule)
	mux.HandleFunc("PUT /api/dns/options", h.handlePutDNSOptions)

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

func (h *Handler) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS（webui dev server 跨端口）
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Secret")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// 鉴权
		if h.secret != "" {
			if r.Header.Get("X-Secret") != h.secret && r.URL.Query().Get("token") != h.secret {
				writeError(w, http.StatusUnauthorized, errors.New("invalid secret"))
				return
			}
		}
		next.ServeHTTP(w, r)
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
		"listen":  values.Listen,
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
	response := map[string]any{"saved": true}
	// 保存后自动重载 sing-box（settings.reload.after_save）
	if executed, err := h.reloadIfEnabled(); err != nil {
		response["reload_error"] = err.Error()
		response["message"] = "配置已保存，但 sing-box 重载失败"
	} else if executed {
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
	writeJSON(w, http.StatusOK, map[string]any{
		"config_path":       h.store.Path(),
		"controller_config": h.settings.Path(),
		"listen":            values.Listen,
		"log_level":         values.Log.Level,
		"min_port":          values.MinPort,
		"defaults":          values.Defaults,
		"inbounds":          len(h.store.Options.Inbounds),
		"outbounds":         len(h.store.Options.Outbounds),
		"rules":             ruleCount,
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
	if newValues.Config == "" {
		writeError(w, http.StatusBadRequest, errors.New("config 路径不能为空"))
		return
	}
	if newValues.Listen != "" {
		_, portText, err := net.SplitHostPort(newValues.Listen)
		if err != nil {
			writeError(w, http.StatusBadRequest, errors.New("listen 格式应为 host:port，如 127.0.0.1:8080"))
			return
		}
		port, parseErr := strconv.Atoi(portText)
		if parseErr == nil && port > 0 && port < 1024 {
			// 特权端口：允许保存（systemd unit 已带 CAP_NET_BIND_SERVICE），但给出提示
			privilegedPortWarning = fmt.Sprintf("端口 %d 是特权端口（<1024）。deb 部署已支持（AmbientCapabilities），本机开发直跑需 root 或加 capabilities。", port)
		}
	}
	if newValues.Log != nil && newValues.Log.Level != "" {
		if _, err := log.ParseLevel(newValues.Log.Level); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if newValues.MinPort < 1024 || newValues.MinPort > 65535 {
		writeError(w, http.StatusBadRequest, errors.New("min_port 需在 1024-65535 之间"))
		return
	}
	if newValues.Reload == nil {
		newValues.Reload = &settings.ReloadOptions{Mode: "auto"}
	}
	if err := newValues.Reload.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx := h.ctx(r)
	if err := h.settings.Update(func(s *settings.Settings) error {
		*s = newValues
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// config 路径变更 → 切换主配置存储
	if h.store.Path() != newValues.Config {
		h.store.SetPath(newValues.Config)
		if err := h.store.Load(ctx, store.DefaultConfig{
			InboundType: newValues.Defaults.InboundType,
			Listen:      newValues.Defaults.Listen,
			ListenPort:  newValues.Defaults.ListenPort,
		}); err != nil {
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
		writeError(w, http.StatusInternalServerError, err)
		return
	}
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
	if err := h.store.RawSave(h.ctx(r), body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response := map[string]any{"saved": true}
	// 保存后自动重载 sing-box（settings.reload.after_save）
	if executed, err := h.reloadIfEnabled(); err != nil {
		response["reload_error"] = err.Error()
		response["message"] = "配置已保存，但 sing-box 重载失败"
	} else if executed {
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
	// 先完整校验，再落盘
	if err := store.Validate(h.ctx(r), body); err != nil {
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
