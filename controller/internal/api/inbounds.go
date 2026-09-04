package api

import (
	"errors"
	"net/http"

	"github.com/qualvey/sing-controller/internal/logger"

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
	h.store.RLock()
	defer h.store.RUnlock()

	ctx := h.ctx(r)
	items := make([]json.RawMessage, 0, len(h.store.Options.Inbounds))
	for i := range h.store.Options.Inbounds {
		content, err := json.MarshalContext(ctx, &h.store.Options.Inbounds[i])
		if err != nil {
			logger.Error("序列化 inbound [%s] 失败: %v", h.store.Options.Inbounds[i].Tag, err)
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		items = append(items, content)
	}
	logger.Debug("查询入站列表成功 (count=%d)", len(items))
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
		logger.Warn("解析创建 inbound 请求体失败: %v", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if inbound.Tag == "" {
		writeError(w, http.StatusBadRequest, errors.New("inbound 必须包含 tag"))
		return
	}
	h.commit(w, r, func(options *option.Options, meta *metaType) error {
		if h.findInboundIndex(inbound.Tag) >= 0 {
			logger.Warn("创建 inbound 失败: tag 已存在 [%s]", inbound.Tag)
			return errors.New("inbound tag 已存在: " + inbound.Tag)
		}
		options.Inbounds = append(options.Inbounds, inbound)

		// 用户池绑定投影：把绑定到该入站的池用户注入 users[]
		if err := syncUsersToInbounds(h.ctx(r), options, meta.Users); err != nil {
			logger.Error("同步用户到 inbound 失败, tag=%s: %v", inbound.Tag, err)
			return err
		}
		logger.Info("成功创建 inbound [tag: %s, type: %s]", inbound.Tag, inbound.Type)
		return nil
	})
}

func (h *Handler) handleGetInbound(w http.ResponseWriter, r *http.Request) {
	h.store.RLock()
	defer h.store.RUnlock()

	tag := r.PathValue("tag")
	index := h.findInboundIndex(tag)
	if index < 0 {
		logger.Debug("获取 inbound 失败: tag 未找到 [%s]", tag)
		writeError(w, http.StatusNotFound, errors.New("inbound 不存在: "+tag))
		return
	}
	content, err := json.MarshalContext(h.ctx(r), &h.store.Options.Inbounds[index])
	if err != nil {
		logger.Error("序列化 inbound [%s] 失败: %v", tag, err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	logger.Debug("获取 inbound 详情成功 [tag: %s]", tag)
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
		logger.Warn("解析更新 inbound [%s] 请求体失败: %v", tag, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if inbound.Tag == "" {
		inbound.Tag = tag
	}
	h.commit(w, r, func(options *option.Options, meta *metaType) error {
		index := h.findInboundIndex(tag)
		if index < 0 {
			logger.Warn("更新 inbound 失败: tag 不存在 [%s]", tag)
			return errors.New("inbound 不存在: " + tag)
		}
		if inbound.Tag != tag {
			if h.findInboundIndex(inbound.Tag) >= 0 {
				logger.Warn("更新 inbound 失败: 新 tag 已存在 [%s]", inbound.Tag)
				return errors.New("inbound tag 已存在: " + inbound.Tag)
			}
			if options.Route != nil {
				for i := range options.Route.Rules {
					_ = renameInboundInRule(h.ctx(r), &options.Route.Rules[i], tag, inbound.Tag)
				}
			}
			if meta != nil {
				for i := range meta.Users {
					for idx, bTag := range meta.Users[i].BoundInbounds {
						if bTag == tag {
							meta.Users[i].BoundInbounds[idx] = inbound.Tag
						}
					}
				}
			}
		}
		options.Inbounds[index] = inbound
		logger.Info("更新 inbound [oldTag: %s, newTag: %s, type: %s]", tag, inbound.Tag, inbound.Type)
		// 用户池绑定投影
		return syncUsersToInbounds(h.ctx(r), options, meta.Users)
	})
}

func (h *Handler) handleDeleteInbound(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	h.commit(w, r, func(options *option.Options, _ *metaType) error {
		index := h.findInboundIndex(tag)
		if index < 0 {
			logger.Warn("删除 inbound 失败: tag 不存在 [%s]", tag)
			return errors.New("inbound 不存在: " + tag)
		}
		if options.Route != nil {
			for i, rule := range options.Route.Rules {
				if ruleReferencesInbound(&rule, tag) {
					logger.Warn("删除 inbound 失败 [%s]: 被 route 规则 #%d 引用", tag, i+1)
					return errors.New("route 规则 #" + itoa(i+1) + " 引用了该 inbound，请先修改路由")
				}
			}
		}
		options.Inbounds = append(options.Inbounds[:index], options.Inbounds[index+1:]...)
		logger.Info("成功删除 inbound [tag: %s]", tag)
		return nil
	})
}
