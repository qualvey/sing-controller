package api

import (
	"errors"
	"net/http"

	"github.com/sagernet/sing-box/option"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing/common/json"
)

// 路由规则本身没有 tag/id 字段（sing-box 严格解码不允许加自定义字段），
// 因此用旁车 meta（config.json.meta）维护 id ↔ 数组下标的映射。

// newRuleID 返回下标 i 对应的稳定 id；缺失时生成新 uuid 并写入 meta。
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
			return errors.New("route 规则不存在: " + id)
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
			return errors.New("route 规则不存在: " + id)
		}
		options.Route.Rules = append(options.Route.Rules[:index], options.Route.Rules[index+1:]...)
		meta.RuleIDs = append(meta.RuleIDs[:index], meta.RuleIDs[index+1:]...)
		return nil
	})
}
