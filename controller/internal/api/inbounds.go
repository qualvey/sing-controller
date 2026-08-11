package api

import (
	"errors"
	"net/http"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

func (h *Handler) findInboundIndex(tag string) int {
	for i, inbound := range h.store.Options.Inbounds {
		if inbound.Tag == tag {
			return i
		}
	}
	return -1
}

func (h *Handler) handleListInbounds(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx(r)
	items := make([]json.RawMessage, 0, len(h.store.Options.Inbounds))
	for i := range h.store.Options.Inbounds {
		content, err := json.MarshalContext(ctx, &h.store.Options.Inbounds[i])
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, content)
	}
	writeJSON(w, http.StatusOK, map[string]any{"inbounds": items})
}

func (h *Handler) handleCreateInbound(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var inbound option.Inbound
	if err := json.UnmarshalContext(h.ctx(r), body, &inbound); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if inbound.Tag == "" {
		writeError(w, http.StatusBadRequest, errors.New("inbound 必须包含 tag"))
		return
	}
	h.commit(w, r, func(options *option.Options, meta *metaType) error {
		if h.findInboundIndex(inbound.Tag) >= 0 {
			return errors.New("inbound tag 已存在: " + inbound.Tag)
		}
		options.Inbounds = append(options.Inbounds, inbound)
		// 用户池绑定投影：把绑定到该入站的池用户注入 users[]
		return syncUsersToInbounds(h.ctx(r), options, meta.Users)
	})
}

func (h *Handler) handleGetInbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	index := h.findInboundIndex(tag)
	if index < 0 {
		writeError(w, http.StatusNotFound, errors.New("inbound 不存在: "+tag))
		return
	}
	content, err := json.MarshalContext(h.ctx(r), &h.store.Options.Inbounds[index])
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) handleUpdateInbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var inbound option.Inbound
	if err := json.UnmarshalContext(h.ctx(r), body, &inbound); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if inbound.Tag == "" {
		inbound.Tag = tag
	}
	if inbound.Tag != tag {
		writeError(w, http.StatusBadRequest, errors.New("路径 tag 与 body tag 不一致"))
		return
	}
	h.commit(w, r, func(options *option.Options, meta *metaType) error {
		index := h.findInboundIndex(tag)
		if index < 0 {
			return errors.New("inbound 不存在: " + tag)
		}
		options.Inbounds[index] = inbound
		// 用户池绑定投影
		return syncUsersToInbounds(h.ctx(r), options, meta.Users)
	})
}

func (h *Handler) handleDeleteInbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	h.commit(w, r, func(options *option.Options, _ *metaType) error {
		index := h.findInboundIndex(tag)
		if index < 0 {
			return errors.New("inbound 不存在: " + tag)
		}
		for i, rule := range options.Route.Rules {
			if ruleReferencesInbound(&rule, tag) {
				return errors.New("route 规则 #" + itoa(i+1) + " 引用了该 inbound，请先修改路由")
			}
		}
		options.Inbounds = append(options.Inbounds[:index], options.Inbounds[index+1:]...)
		return nil
	})
}
