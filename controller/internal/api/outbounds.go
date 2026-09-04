package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

func (h *Handler) findOutboundIndex(tag string) int {
	for i, outbound := range h.store.Options.Outbounds {
		if outbound.Tag == tag {
			return i
		}
	}
	return -1
}

func (h *Handler) handleListOutbounds(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx(r)
	items := make([]json.RawMessage, 0, len(h.store.Options.Outbounds))
	for i := range h.store.Options.Outbounds {
		content, err := json.MarshalContext(ctx, &h.store.Options.Outbounds[i])
		if err != nil {
			logger.Error("序列化 outbound [%s] 失败: %v", h.store.Options.Outbounds[i].Tag, err)
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, content)
	}
	logger.Debug("查询出站列表成功 (count=%d)", len(items))
	writeJSON(w, http.StatusOK, map[string]any{"outbounds": items})
}

// parseOutboundPayload 从请求体中剥离 attach_groups 扩展字段，并反序列化 option.Outbound，
// 避免 option.Outbound 严格反序列化时触发 unknown field "attach_groups" 错误。
func parseOutboundPayload(ctx context.Context, body []byte) (option.Outbound, *[]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return option.Outbound{}, nil, err
	}
	var attachGroups *[]string
	if grpRaw, ok := raw["attach_groups"]; ok {
		var grps []string
		if err := json.Unmarshal(grpRaw, &grps); err != nil {
			return option.Outbound{}, nil, err
		}
		attachGroups = &grps
		delete(raw, "attach_groups")
	}
	cleanBody, err := json.Marshal(raw)
	if err != nil {
		return option.Outbound{}, nil, err
	}
	var outbound option.Outbound
	if err := json.UnmarshalContext(ctx, cleanBody, &outbound); err != nil {
		return option.Outbound{}, nil, err
	}
	return outbound, attachGroups, nil
}

func (h *Handler) handleCreateOutbound(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	outbound, attachGroups, err := parseOutboundPayload(h.ctx(r), body)
	if err != nil {
		logger.Warn("解析创建 outbound 请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if outbound.Tag == "" {
		writeError(w, http.StatusBadRequest, errors.New("outbound 必须包含 tag"))
		return
	}
	h.commit(w, r, func(options *option.Options, _ *metaType) error {
		if h.findOutboundIndex(outbound.Tag) >= 0 {
			logger.Warn("创建 outbound 失败: tag 已存在 [%s]", outbound.Tag)
			return errors.New("outbound tag 已存在: " + outbound.Tag)
		}
		options.Outbounds = append(options.Outbounds, outbound)
		if attachGroups != nil && len(*attachGroups) > 0 {
			syncTagToGroups(options, outbound.Tag, *attachGroups)
		}
		var attached []string
		if attachGroups != nil {
			attached = *attachGroups
		}
		logger.Info("成功创建 outbound [tag: %s, type: %s, attached_groups=%v]", outbound.Tag, outbound.Type, attached)
		return nil
	})
}

// syncTagToGroups 保持 tag 与目标 selector/urltest 分组列表同步
func syncTagToGroups(options *option.Options, tag string, targetGroups []string) {
	targetMap := make(map[string]bool)
	for _, g := range targetGroups {
		targetMap[g] = true
	}
	for i := range options.Outbounds {
		ob := &options.Outbounds[i]
		switch typed := ob.Options.(type) {
		case *option.SelectorOutboundOptions:
			if targetMap[ob.Tag] {
				if !containsString(typed.Outbounds, tag) {
					typed.Outbounds = append(typed.Outbounds, tag)
				}
			}
		case *option.URLTestOutboundOptions:
			if targetMap[ob.Tag] {
				if !containsString(typed.Outbounds, tag) {
					typed.Outbounds = append(typed.Outbounds, tag)
				}
			}
		}
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// groupReferencesTag 返回所有引用 tag 的 selector/urltest 组 tag（按配置顺序，去重）。
func groupReferencesTag(options *option.Options, tag string) []string {
	var refs []string
	for i := range options.Outbounds {
		outbound := &options.Outbounds[i]
		var members []string
		switch typed := outbound.Options.(type) {
		case *option.SelectorOutboundOptions:
			members = typed.Outbounds
		case *option.URLTestOutboundOptions:
			members = typed.Outbounds
		}
		for _, member := range members {
			if member == tag {
				refs = append(refs, outbound.Tag)
				break
			}
		}
	}
	return refs
}

// removeTagFromGroups 从所有 selector/urltest 组中移除 tag。
func removeTagFromGroups(options *option.Options, tag string) {
	for i := range options.Outbounds {
		outbound := &options.Outbounds[i]
		switch typed := outbound.Options.(type) {
		case *option.SelectorOutboundOptions:
			typed.Outbounds = removeString(typed.Outbounds, tag)
		case *option.URLTestOutboundOptions:
			typed.Outbounds = removeString(typed.Outbounds, tag)
		}
	}
}

func removeString(list []string, s string) []string {
	out := list[:0]
	for _, v := range list {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

func (h *Handler) handleGetOutbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	index := h.findOutboundIndex(tag)
	if index < 0 {
		logger.Debug("获取 outbound 失败: tag 未找到 [%s]", tag)
		writeError(w, http.StatusNotFound, errors.New("outbound 不存在: "+tag))
		return
	}
	content, err := json.MarshalContext(h.ctx(r), &h.store.Options.Outbounds[index])
	if err != nil {
		logger.Error("序列化 outbound [%s] 失败: %v", tag, err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	var resMap map[string]any
	if err := json.Unmarshal(content, &resMap); err == nil {
		resMap["attach_groups"] = groupReferencesTag(&h.store.Options, tag)
		writeJSON(w, http.StatusOK, resMap)
		return
	}
	logger.Debug("获取 outbound 详情成功 [tag: %s]", tag)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

// renameTagInGroups 将所有 selector/urltest 组中的 oldTag 替换为 newTag。
func renameTagInGroups(options *option.Options, oldTag, newTag string) {
	for i := range options.Outbounds {
		outbound := &options.Outbounds[i]
		switch typed := outbound.Options.(type) {
		case *option.SelectorOutboundOptions:
			for idx, m := range typed.Outbounds {
				if m == oldTag {
					typed.Outbounds[idx] = newTag
				}
			}
			if typed.Default == oldTag {
				typed.Default = newTag
			}
		case *option.URLTestOutboundOptions:
			for idx, m := range typed.Outbounds {
				if m == oldTag {
					typed.Outbounds[idx] = newTag
				}
			}
		}
	}
}

func (h *Handler) handleUpdateOutbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	outbound, attachGroups, err := parseOutboundPayload(h.ctx(r), body)
	if err != nil {
		logger.Warn("解析更新 outbound [%s] 请求体失败: %v", tag, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if outbound.Tag == "" {
		outbound.Tag = tag
	}
	h.commit(w, r, func(options *option.Options, _ *metaType) error {
		index := h.findOutboundIndex(tag)
		if index < 0 {
			logger.Warn("更新 outbound 失败: tag 不存在 [%s]", tag)
			return errors.New("outbound 不存在: " + tag)
		}
		effectiveTag := tag
		if outbound.Tag != tag {
			if h.findOutboundIndex(outbound.Tag) >= 0 {
				logger.Warn("更新 outbound 失败: 新 tag 已存在 [%s]", outbound.Tag)
				return errors.New("outbound tag 已存在: " + outbound.Tag)
			}
			renameTagInGroups(options, tag, outbound.Tag)
			if options.Route != nil {
				if options.Route.Final == tag {
					options.Route.Final = outbound.Tag
				}
				for i := range options.Route.Rules {
					_ = renameOutboundInRule(h.ctx(r), &options.Route.Rules[i], tag, outbound.Tag)
				}
			}
			effectiveTag = outbound.Tag
		}
		if attachGroups != nil {
			updateTagGroups(options, effectiveTag, *attachGroups)
		}
		options.Outbounds[index] = outbound
		logger.Info("更新 outbound [oldTag: %s, newTag: %s, type: %s]", tag, outbound.Tag, outbound.Type)
		return nil
	})
}

// updateTagGroups 精确更新 tag 在所有分组中的存在状态（在列表中则加入，不在列表中则移除）
func updateTagGroups(options *option.Options, tag string, targetGroups []string) {
	targetMap := make(map[string]bool)
	for _, g := range targetGroups {
		targetMap[g] = true
	}
	for i := range options.Outbounds {
		ob := &options.Outbounds[i]
		switch typed := ob.Options.(type) {
		case *option.SelectorOutboundOptions:
			if targetMap[ob.Tag] {
				if !containsString(typed.Outbounds, tag) {
					typed.Outbounds = append(typed.Outbounds, tag)
				}
			} else {
				typed.Outbounds = removeString(typed.Outbounds, tag)
			}
		case *option.URLTestOutboundOptions:
			if targetMap[ob.Tag] {
				if !containsString(typed.Outbounds, tag) {
					typed.Outbounds = append(typed.Outbounds, tag)
				}
			} else {
				typed.Outbounds = removeString(typed.Outbounds, tag)
			}
		}
	}
}

func (h *Handler) handleDeleteOutbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	force := r.URL.Query().Get("force") == "true"
	h.commit(w, r, func(options *option.Options, _ *metaType) error {
		index := h.findOutboundIndex(tag)
		if index < 0 {
			logger.Warn("删除 outbound 失败: tag 不存在 [%s]", tag)
			return errors.New("outbound 不存在: " + tag)
		}
		// 引用检查：route.final 与规则 outbound 字段（始终拒绝，需手动改路由）
		if options.Route != nil {
			if options.Route.Final == tag {
				logger.Warn("删除 outbound 失败 [%s]: 被 route.final 引用", tag)
				return errors.New("route.final 引用了该 outbound，请先修改路由")
			}
			for i, rule := range options.Route.Rules {
				if ruleReferencesOutbound(&rule, tag) {
					logger.Warn("删除 outbound 失败 [%s]: 被 route 规则 #%d 引用", tag, i+1)
					return errors.New("route 规则 #" + itoa(i+1) + " 引用了该 outbound，请先修改路由")
				}
			}
		}
		// 引用检查：selector/urltest 组成员；force=true 时自动从所有引用组拔除 tag
		if refs := groupReferencesTag(options, tag); len(refs) > 0 {
			if !force {
				logger.Warn("删除 outbound 存在组引用阻止 [%s]: 引用组=%v", tag, refs)
				return &GroupReferenceError{Tag: tag, References: refs}
			}
			logger.Info("强制删除 outbound [%s]: 从引用组 %v 中拔除", tag, refs)
			removeTagFromGroups(options, tag)
		}
		options.Outbounds = append(options.Outbounds[:index], options.Outbounds[index+1:]...)
		logger.Info("成功删除 outbound [tag: %s, force=%v]", tag, force)
		return nil
	})
}

