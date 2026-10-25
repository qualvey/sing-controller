package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing-box/option"
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

	h.store.RLock()
	defer h.store.RUnlock()

	var rules []option.Rule
	finalOutbound := ""
	if h.store.Options.Route != nil {
		rules = h.store.Options.Route.Rules
		finalOutbound = h.store.Options.Route.Final
	}

	items := make([]map[string]any, 0, len(rules))
	for i := range rules {
		content, err := json.MarshalContext(ctx, &rules[i])
		if err != nil {
			logger.Error("序列化 route 规则 #%d 失败: %v", i+1, err)
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		id := newRuleID(&h.store.Meta, i)
		items = append(items, map[string]any{
			"id":       id,
			"rule":     json.RawMessage(content),
			"disabled": h.store.Meta.IsRuleDisabled(id),
		})
	}
	logger.Debug("查询 route 规则列表成功 (count=%d)", len(items))
	writeJSON(w, http.StatusOK, map[string]any{"routes": items, "final": finalOutbound})
}

func (h *Handler) handleGetRoute(w http.ResponseWriter, r *http.Request) {
	h.store.RLock()
	defer h.store.RUnlock()

	id := r.PathValue("id")
	index := h.findRuleIndexByID(id)
	if h.store.Options.Route == nil || index < 0 || index >= len(h.store.Options.Route.Rules) {
		logger.Debug("获取 route 规则失败: id 未找到 [%s]", id)
		writeError(w, http.StatusNotFound, errors.New("route 规则不存在: "+id))
		return
	}
	content, err := json.MarshalContext(h.ctx(r), &h.store.Options.Route.Rules[index])
	if err != nil {
		logger.Error("序列化 route 规则 [%s] 失败: %v", id, err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Debug("获取 route 规则详情成功 [id=%s]", id)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"rule":     json.RawMessage(content),
		"disabled": h.store.Meta.IsRuleDisabled(id),
	})
}

func (h *Handler) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var rule option.Rule
	if err := json.UnmarshalContext(h.ctx(r), body, &rule); err != nil {
		logger.Warn("解析创建 route 规则请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		if options.Route == nil {
			options.Route = &option.RouteOptions{}
		}
		newID := store.NewUUID()
		options.Route.Rules = append(options.Route.Rules, rule)
		meta.RuleIDs = append(meta.RuleIDs, newID)
		logger.Info("成功创建 route 规则 [id=%s, rules_count=%d]", newID, len(options.Route.Rules))
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
		logger.Warn("解析更新 route 规则 [%s] 请求体失败: %v", id, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findRuleIndexByID(id)
		if index < 0 {
			logger.Warn("更新 route 规则失败: id 不存在 [%s]", id)
			return errors.New("route 规则不存在: " + id)
		}
		options.Route.Rules[index] = rule
		logger.Info("更新 route 规则成功 [id=%s]", id)
		return nil
	})
}

func (h *Handler) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findRuleIndexByID(id)
		if index < 0 {
			logger.Warn("删除 route 规则失败: id 不存在 [%s]", id)
			return errors.New("route 规则不存在: " + id)
		}
		options.Route.Rules = append(options.Route.Rules[:index], options.Route.Rules[index+1:]...)
		meta.RuleIDs = append(meta.RuleIDs[:index], meta.RuleIDs[index+1:]...)
		meta.SetRuleDisabled(id, false)
		logger.Info("成功删除 route 规则 [id=%s]", id)
		return nil
	})
}

func (h *Handler) handleToggleRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("missing route id"))
		return
	}
	var req struct {
		Disabled *bool `json:"disabled"`
	}
	body, _ := readRawBody(r)
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}

	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findRuleIndexByID(id)
		if index < 0 {
			logger.Warn("切换 route 规则状态失败: id 不存在 [%s]", id)
			return errors.New("route 规则不存在: " + id)
		}
		if req.Disabled != nil {
			meta.SetRuleDisabled(id, *req.Disabled)
		} else {
			meta.ToggleRuleDisabled(id)
		}
		logger.Info("切换 route 规则状态 [id=%s, disabled=%v]", id, meta.IsRuleDisabled(id))
		return nil
	})
}

// handleReorderRoutes 批量调整路由规则执行顺序。
// @Summary 调整路由规则顺序
// @Description 按传入的 ID 数组全量调整路由规则的执行顺序
// @Tags 路由规则
// @Accept json
// @Produce json
// @Param body body object true "有序规则 ID 列表: { \"ids\": [\"uuid1\", \"uuid2\"] }"
// @Success 200 {object} map[string]any
// @Failure 400 {object} map[string]any
// @Router /api/routes/order [put]
func (h *Handler) handleReorderRoutes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Warn("解析重排 route 规则请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		if options.Route == nil || len(options.Route.Rules) == 0 {
			if len(req.IDs) == 0 {
				return nil
			}
			return errors.New("route 规则列表为空")
		}

		currentCount := len(options.Route.Rules)
		if len(req.IDs) != currentCount {
			return fmt.Errorf("规则数量不匹配: 期望 %d, 收到 %d", currentCount, len(req.IDs))
		}

		// 确保所有规则在当前 meta 中具备稳定 ID
		for i := range options.Route.Rules {
			_ = newRuleID(meta, i)
		}

		// 构建已有 ID -> Rule 映射并检查传入 ID 的合法性与唯一性
		ruleMap := make(map[string]option.Rule, currentCount)
		for i, id := range meta.RuleIDs {
			ruleMap[id] = options.Route.Rules[i]
		}

		seen := make(map[string]struct{}, currentCount)
		newRules := make([]option.Rule, currentCount)
		for i, id := range req.IDs {
			if _, exists := seen[id]; exists {
				return fmt.Errorf("重复的规则 ID: %s", id)
			}
			seen[id] = struct{}{}

			rule, ok := ruleMap[id]
			if !ok {
				return fmt.Errorf("未知的规则 ID: %s", id)
			}
			newRules[i] = rule
		}

		options.Route.Rules = newRules
		meta.RuleIDs = req.IDs
		logger.Info("成功重排 route 规则顺序 (count=%d)", currentCount)
		return nil
	})
}

