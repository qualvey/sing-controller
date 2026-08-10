// Package api 提供 RESTful 配置管理 API。
// 设计原则：所有写操作走 store.Update 校验管线（解码 → box.New 干跑 → 原子写），
// 校验失败一律不落盘；reload 失败保留旧实例。
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/schema"
	"github.com/sagernet/sing-box-webui/internal/runner"
	"github.com/sagernet/sing-box-webui/internal/store"
)

// metaType 别名，简化 handler 签名书写
type metaType = store.Meta

type HandlerOptions struct {
	Store  *store.Store
	Runner *runner.Runner
	Secret string
	Static http.Handler
	NoRun  bool
}

type Handler struct {
	store  *store.Store
	runner *runner.Runner
	secret string
	static http.Handler

	schemaOnce sync.Once
	schemaJSON []byte
	schemaErr  error
}

func NewHandler(opts HandlerOptions) http.Handler {
	h := &Handler{
		store:  opts.Store,
		runner: opts.Runner,
		secret: opts.Secret,
		static: opts.Static,
	}
	mux := http.NewServeMux()

	// 状态 / 配置 / 工具
	mux.HandleFunc("GET /api/status", h.handleStatus)
	mux.HandleFunc("GET /api/config", h.handleGetConfig)
	mux.HandleFunc("PUT /api/config", h.handlePutConfig)
	mux.HandleFunc("GET /api/schema", h.handleSchema)
	mux.HandleFunc("GET /api/types", h.handleTypes)
	mux.HandleFunc("POST /api/reload", h.handleReload)
	mux.HandleFunc("POST /api/instance/start", h.handleInstanceStart)
	mux.HandleFunc("POST /api/instance/stop", h.handleInstanceStop)
	mux.HandleFunc("POST /api/tools/uuid", h.handleToolUUID)
	mux.HandleFunc("POST /api/tools/reality-keypair", h.handleToolRealityKeypair)

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

	// 静态资源（前端）
	mux.HandleFunc("/", h.serveStatic)

	return h.withMiddleware(mux)
}

func (h *Handler) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS（开发态前端 vite 跨端口）
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

func (h *Handler) serveStatic(w http.ResponseWriter, r *http.Request) {
	if h.static == nil {
		http.NotFound(w, r)
		return
	}
	h.static.ServeHTTP(w, r)
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

// commit 执行"内存修改 → 全量校验 → 原子写盘 → 实例 reload"管线。
func (h *Handler) commit(w http.ResponseWriter, r *http.Request, mutate func(*option.Options, *store.Meta) error) bool {
	ctx := h.ctx(r)
	if err := h.store.Update(ctx, mutate); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	if h.runner != nil {
		if err := h.runner.Reload(ctx, h.store.Options); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"saved":        true,
				"reload_error": err.Error(),
				"message":      "配置已保存，但实例重载失败，旧实例仍在运行",
			})
			return false
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": true, "reloaded": h.runner != nil})
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func readBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	content, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	if err := json.Unmarshal(content, dst); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

// ---------- status ----------

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	running := false
	if h.runner != nil {
		running = h.runner.Running()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"running":     running,
		"config_path": h.store.Path(),
		"inbounds":    len(h.store.Options.Inbounds),
		"outbounds":   len(h.store.Options.Outbounds),
		"rules":       len(h.store.Options.Route.Rules),
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

func (h *Handler) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 先完整校验，再落盘
	if err := store.Validate(h.ctx(r), body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ok := h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		optionsValue, err := store.Parse(h.ctx(r), body)
		if err != nil {
			return err
		}
		*options = optionsValue
		meta.RuleIDs = make([]string, len(optionsValue.Route.Rules))
		for i := range meta.RuleIDs {
			meta.RuleIDs[i] = newRuleID(meta, i)
		}
		return nil
	})
	_ = ok
}

// ---------- reload / instance ----------

func (h *Handler) handleReload(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx(r)
	if err := h.store.Load(ctx); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if h.runner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"reloaded": false, "message": "no-run 模式，无实例"})
		return
	}
	if err := h.runner.Reload(ctx, h.store.Options); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reloaded": true})
}

func (h *Handler) handleInstanceStart(w http.ResponseWriter, r *http.Request) {
	if h.runner == nil {
		writeError(w, http.StatusBadRequest, errors.New("no-run 模式，无实例"))
		return
	}
	if err := h.runner.Start(h.ctx(r), h.store.Options); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"running": true})
}

func (h *Handler) handleInstanceStop(w http.ResponseWriter, r *http.Request) {
	if h.runner == nil {
		writeError(w, http.StatusBadRequest, errors.New("no-run 模式，无实例"))
		return
	}
	if err := h.runner.Stop(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"running": false})
}

var _ = strings.TrimSpace
