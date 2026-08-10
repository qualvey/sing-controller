// Package api æä¾› RESTful é…ç½®ç®¡ç† APIã€‚
// è®¾è®¡åŽŸåˆ™ï¼šæ‰€æœ‰å†™æ“ä½œèµ° store.Update æ ¡éªŒç®¡çº¿ï¼ˆè§£ç  â†’ box.New å¹²è·‘ â†’ åŽŸå­å†™ç›˜ï¼‰ï¼Œ
// æ ¡éªŒå¤±è´¥ä¸€å¾‹ä¸è½ç›˜ã€å†…å­˜å›žæ»šã€‚
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/schema"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
)

type HandlerOptions struct {
	Store    *store.Store
	Settings *settings.Manager
	Secret   string
}

type Handler struct {
	store    *store.Store
	settings *settings.Manager
	secret   string

	schemaOnce sync.Once
	schemaJSON []byte
	schemaErr  error
}

// metaType åˆ«åï¼Œç®€åŒ– handler ç­¾åä¹¦å†™
type metaType = store.Meta

func NewHandler(opts HandlerOptions) http.Handler {
	h := &Handler{
		store:    opts.Store,
		settings: opts.Settings,
		secret:   opts.Secret,
	}
	mux := http.NewServeMux()

	// çŠ¶æ€ / é…ç½® / è®¾ç½®
	mux.HandleFunc("GET /api/status", h.handleStatus)
	mux.HandleFunc("GET /api/config", h.handleGetConfig)
	mux.HandleFunc("PUT /api/config", h.handlePutConfig)
	mux.HandleFunc("GET /api/schema", h.handleSchema)
	mux.HandleFunc("GET /api/types", h.handleTypes)
	mux.HandleFunc("GET /api/settings", h.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", h.handlePutSettings)
	mux.HandleFunc("GET /api/ports/available", h.handlePortAvailable)

	// å·¥å…·
	mux.HandleFunc("POST /api/tools/uuid", h.handleToolUUID)
	mux.HandleFunc("POST /api/tools/reality-keypair", h.handleToolRealityKeypair)
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

	// route rule CRUDï¼ˆç®€å•è§„åˆ™ï¼Œid å­˜äºŽ meta æ—è½¦ï¼‰
	mux.HandleFunc("GET /api/routes", h.handleListRoutes)
	mux.HandleFunc("POST /api/routes", h.handleCreateRoute)
	mux.HandleFunc("PUT /api/routes/{id}", h.handleUpdateRoute)
	mux.HandleFunc("DELETE /api/routes/{id}", h.handleDeleteRoute)

	return h.withMiddleware(mux)
}

func (h *Handler) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORSï¼ˆwebui dev server è·¨ç«¯å£ï¼‰
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
		// é‰´æƒ
		if h.secret != "" {
			if r.Header.Get("X-Secret") != h.secret && r.URL.Query().Get("token") != h.secret {
				writeError(w, http.StatusUnauthorized, errors.New("invalid secret"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
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

// commit æ‰§è¡Œ"å†…å­˜ä¿®æ”¹ â†’ å…¨é‡æ ¡éªŒ â†’ åŽŸå­å†™ç›˜"ç®¡çº¿ã€‚
func (h *Handler) commit(w http.ResponseWriter, r *http.Request, mutate func(*option.Options, *store.Meta) error) {
	if err := h.store.Update(h.ctx(r), mutate); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": true})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
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
	writeJSON(w, http.StatusOK, map[string]any{
		"config_path":        h.store.Path(),
		"controller_config":  h.settings.Path(),
		"min_port":           values.MinPort,
		"defaults":           values.Defaults,
		"inbounds":           len(h.store.Options.Inbounds),
		"outbounds":          len(h.store.Options.Outbounds),
		"rules":              len(h.store.Options.Route.Rules),
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
	if newValues.Config == "" {
		writeError(w, http.StatusBadRequest, errors.New("config è·¯å¾„ä¸èƒ½ä¸ºç©º"))
		return
	}
	if newValues.MinPort < 1024 || newValues.MinPort > 65535 {
		writeError(w, http.StatusBadRequest, errors.New("min_port éœ€åœ¨ 1024-65535 ä¹‹é—´"))
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
	// config è·¯å¾„å˜æ›´ â†’ åˆ‡æ¢ä¸»é…ç½®å­˜å‚¨
	if h.store.Path() != newValues.Config {
		h.store.SetPath(newValues.Config)
		if err := h.store.Load(ctx, store.DefaultConfig{
			InboundType: newValues.Defaults.InboundType,
			Listen:      newValues.Defaults.Listen,
			ListenPort:  newValues.Defaults.ListenPort,
		}); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"saved":     true,
				"load_error": err.Error(),
				"message":   "controller é…ç½®å·²ä¿å­˜ï¼Œä½†æ–°ä¸»é…ç½®è·¯å¾„åŠ è½½å¤±è´¥",
			})
			return
		}
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

func (h *Handler) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// å…ˆå®Œæ•´æ ¡éªŒï¼Œå†è½ç›˜
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
