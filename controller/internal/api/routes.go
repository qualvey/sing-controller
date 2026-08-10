package api

import (
	"errors"
	"net/http"

	"github.com/sagernet/sing-box/option"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing/common/json"
)

// è·¯ç”±è§„åˆ™æœ¬èº«æ²¡æœ‰ tag/id å­—æ®µï¼ˆsing-box ä¸¥æ ¼è§£ç ä¸å…è®¸åŠ è‡ªå®šä¹‰å­—æ®µï¼‰ï¼Œ
// å› æ­¤ç”¨æ—è½¦ metaï¼ˆconfig.json.metaï¼‰ç»´æŠ¤ id â†” æ•°ç»„ä¸‹æ ‡ çš„æ˜ å°„ã€‚

// newRuleID è¿”å›žä¸‹æ ‡ i å¯¹åº”çš„ç¨³å®š idï¼›ç¼ºå¤±æ—¶ç”Ÿæˆæ–° uuid å¹¶å†™å…¥ metaã€‚
func newRuleID(meta *store.Meta, index int) string {
	for len(meta.RuleIDs) <= index {
		meta.RuleIDs = append(meta.RuleIDs, "")
	}
	if meta.RuleIDs[index] == "" {
		meta.RuleIDs[index] = store.NewUUID()
	}
	return meta.RuleIDs[index]
}

func (h *Handler) findRuleIndexByID(id string) int {
	meta := &h.store.Meta
	for i, ruleID := range meta.RuleIDs {
		if ruleID == id {
			return i
		}
	}
	return -1
}

// ruleResponse ç»„è£… {id, rule} è¿”å›žç»“æž„ï¼ˆæœªä½¿ç”¨ï¼Œä¿ç•™å ä½ï¼‰ã€‚

func (h *Handler) handleListRoutes(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx(r)
	h.store.AlignMeta()
	items := make([]map[string]any, 0, len(h.store.Options.Route.Rules))
	for i := range h.store.Options.Route.Rules {
		content, err := json.MarshalContext(ctx, &h.store.Options.Route.Rules[i])
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, map[string]any{
			"id":   newRuleID(&h.store.Meta, i),
			"rule": json.RawMessage(content),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"routes": items, "final": h.store.Options.Route.Final})
}

func (h *Handler) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var rule option.Rule
	if err := json.UnmarshalContext(h.ctx(r), body, &rule); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		options.Route.Rules = append(options.Route.Rules, rule)
		meta.RuleIDs = append(meta.RuleIDs, store.NewUUID())
		return nil
	})
}

func (h *Handler) handleUpdateRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var rule option.Rule
	if err := json.UnmarshalContext(h.ctx(r), body, &rule); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findRuleIndexByID(id)
		if index < 0 {
			return errors.New("route è§„åˆ™ä¸å­˜åœ¨: " + id)
		}
		options.Route.Rules[index] = rule
		return nil
	})
}

func (h *Handler) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findRuleIndexByID(id)
		if index < 0 {
			return errors.New("route è§„åˆ™ä¸å­˜åœ¨: " + id)
		}
		options.Route.Rules = append(options.Route.Rules[:index], options.Route.Rules[index+1:]...)
		meta.RuleIDs = append(meta.RuleIDs[:index], meta.RuleIDs[index+1:]...)
		return nil
	})
}
