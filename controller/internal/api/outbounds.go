package api

import (
	"errors"
	"net/http"

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
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, content)
	}
	writeJSON(w, http.StatusOK, map[string]any{"outbounds": items})
}

func (h *Handler) handleCreateOutbound(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var outbound option.Outbound
	if err := json.UnmarshalContext(h.ctx(r), body, &outbound); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if outbound.Tag == "" {
		writeError(w, http.StatusBadRequest, errors.New("outbound 必须包含 tag"))
		return
	}
	attached := false
	err = h.store.Update(h.ctx(r), func(options *option.Options, _ *metaType) error {
		if h.findOutboundIndex(outbound.Tag) >= 0 {
			return errors.New("outbound tag 已存在: " + outbound.Tag)
		}
		options.Outbounds = append(options.Outbounds, outbound)
		// 自动并入 Proxy selector（settings 默认开）
		if h.attachToSelector(options, outbound.Tag) {
			attached = true
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response := map[string]any{"saved": true}
	if attached {
		response["attached"] = true
	}
	writeJSON(w, http.StatusOK, response)
}

// attachToSelector 将新 outbound tag 追加到 settings 指定的 selector/urltest（去重）。
func (h *Handler) attachToSelector(options *option.Options, tag string) bool {
	values := h.settings.Values()
	if values.Defaults.AttachToSelector == nil || !*values.Defaults.AttachToSelector || values.Defaults.ProxySelector == "" {
		return false
	}
	target := values.Defaults.ProxySelector
	// 排除自身（新建的 selector 不应把自己加进成员）
	if tag == target {
		return false
	}
	for i := range options.Outbounds {
		outbound := &options.Outbounds[i]
		if outbound.Tag != target {
			continue
		}
		switch typed := outbound.Options.(type) {
		case *option.SelectorOutboundOptions:
			for _, existing := range typed.Outbounds {
				if existing == tag {
					return true
				}
			}
			typed.Outbounds = append(typed.Outbounds, tag)
			return true
		case *option.URLTestOutboundOptions:
			for _, existing := range typed.Outbounds {
				if existing == tag {
					return true
				}
			}
			typed.Outbounds = append(typed.Outbounds, tag)
			return true
		}
	}
	return false
}

// groupReferencesTag 检查所有 selector/urltest 组是否引用指定 tag。
func groupReferencesTag(options *option.Options, tag string) (string, bool) {
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
				return outbound.Tag, true
			}
		}
	}
	return "", false
}

func (h *Handler) handleGetOutbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	index := h.findOutboundIndex(tag)
	if index < 0 {
		writeError(w, http.StatusNotFound, errors.New("outbound 不存在: "+tag))
		return
	}
	content, err := json.MarshalContext(h.ctx(r), &h.store.Options.Outbounds[index])
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) handleUpdateOutbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var outbound option.Outbound
	if err := json.UnmarshalContext(h.ctx(r), body, &outbound); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if outbound.Tag == "" {
		outbound.Tag = tag
	}
	if outbound.Tag != tag {
		writeError(w, http.StatusBadRequest, errors.New("路径 tag 与 body tag 不一致"))
		return
	}
	h.commit(w, r, func(options *option.Options, _ *metaType) error {
		index := h.findOutboundIndex(tag)
		if index < 0 {
			return errors.New("outbound 不存在: " + tag)
		}
		options.Outbounds[index] = outbound
		return nil
	})
}

func (h *Handler) handleDeleteOutbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	h.commit(w, r, func(options *option.Options, _ *metaType) error {
		index := h.findOutboundIndex(tag)
		if index < 0 {
			return errors.New("outbound 不存在: " + tag)
		}
		// 引用检查：route.final 与规则 outbound 字段
		if options.Route != nil && options.Route.Final == tag {
			return errors.New("route.final 引用了该 outbound，请先修改路由")
		}
		for i, rule := range options.Route.Rules {
			if ruleReferencesOutbound(&rule, tag) {
				return errors.New("route 规则 #" + itoa(i+1) + " 引用了该 outbound，请先修改路由")
			}
		}
		// 引用检查：selector/urltest 组成员
		if groupTag, referenced := groupReferencesTag(options, tag); referenced {
			return errors.New("outbound 组 " + groupTag + " 引用了该 outbound，请先从组中移除")
		}
		options.Outbounds = append(options.Outbounds[:index], options.Outbounds[index+1:]...)
		return nil
	})
}
