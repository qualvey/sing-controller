package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/sagernet/sing-box/option"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing/common/json"
)

// 规则集（route.rule_set 段）CRUD。
// sing-box 1.14 规则集三种类型（option/rule_set.go）：
//   - inline：内联规则（tag 单值，rules 为 HeadlessRule 数组）
//   - local：本地文件（path，format 按扩展名推断 .json→source / .srs→binary）
//   - remote：远程 URL（url/initial_path/update_interval/http_client）
// 多 tag 时 local/remote 的 path/url 必须包含 {tag} 占位符。

func (h *Handler) ensureRuleSet(options *option.Options) {
	if options.Route == nil {
		options.Route = &option.RouteOptions{}
	}
}

func newRuleSetID(meta *store.Meta, index int) string {
	for len(meta.RuleSetIDs) <= index {
		meta.RuleSetIDs = append(meta.RuleSetIDs, "")
	}
	if meta.RuleSetIDs[index] == "" {
		meta.RuleSetIDs[index] = store.NewUUID()
	}
	return meta.RuleSetIDs[index]
}

func (h *Handler) findRuleSetIndexByID(id string) int {
	for i, ruleSetID := range h.store.Meta.RuleSetIDs {
		if ruleSetID == id {
			return i
		}
	}
	return -1
}

// ruleSetsReferencedBy 收集引用 rule_set tag 的位置（route/dns 规则的 rule_set 字段）。
func ruleSetsReferencedBy(options *option.Options, tag string) []string {
	var refs []string
	if options.Route != nil {
		for i := range options.Route.Rules {
			content, err := json.Marshal(&options.Route.Rules[i])
			if err != nil {
				continue
			}
			var decoded map[string]any
			if json.Unmarshal(content, &decoded) != nil {
				continue
			}
			for _, ref := range ruleSetRefs(decoded) {
				if ref == tag {
					refs = append(refs, "route 规则 #"+itoa(i+1))
					break
				}
			}
		}
	}
	if options.DNS != nil {
		for i := range options.DNS.Rules {
			content, err := json.Marshal(&options.DNS.Rules[i])
			if err != nil {
				continue
			}
			var decoded map[string]any
			if json.Unmarshal(content, &decoded) != nil {
				continue
			}
			for _, ref := range ruleSetRefs(decoded) {
				if ref == tag {
					refs = append(refs, "dns 规则 #"+itoa(i+1))
					break
				}
			}
		}
	}
	return refs
}

// removeRuleSetRef 从规则对象中移除对 tag 的 rule_set 引用（force 删除用，JSON map 操作保多态）。
// 注意：必须解码到新对象再回写——json 解码复用对象时，content 未出现的字段会保留旧值。
func removeRuleSetRef(ctx context.Context, rule any, tag string) error {
	content, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return err
	}
	v, ok := decoded["rule_set"]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case string:
		if t == tag {
			delete(decoded, "rule_set")
		}
	case []any:
		filtered := make([]any, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); !ok || s != tag {
				filtered = append(filtered, item)
			}
		}
		if len(filtered) == 0 {
			delete(decoded, "rule_set")
		} else {
			decoded["rule_set"] = filtered
		}
	default:
		return nil
	}
	content, err = json.Marshal(decoded)
	if err != nil {
		return err
	}
	switch ptr := rule.(type) {
	case *option.Rule:
		var fresh option.Rule
		if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
			return err
		}
		*ptr = fresh
	case *option.DNSRule:
		var fresh option.DNSRule
		if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
			return err
		}
		*ptr = fresh
	default:
		return errors.New("removeRuleSetRef: unsupported rule type")
	}
	return nil
}

func (h *Handler) handleListRuleSets(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx(r)
	h.store.AlignMeta()
	items := make([]map[string]any, 0)
	if h.store.Options.Route != nil {
		for i := range h.store.Options.Route.RuleSet {
			content, err := json.MarshalContext(ctx, &h.store.Options.Route.RuleSet[i])
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			items = append(items, map[string]any{
				"id":       newRuleSetID(&h.store.Meta, i),
				"rule_set": json.RawMessage(content),
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"rule_sets": items})
}

func (h *Handler) handleCreateRuleSet(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var ruleSet option.RuleSet
	if err := json.Unmarshal(body, &ruleSet); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		h.ensureRuleSet(options)
		options.Route.RuleSet = append(options.Route.RuleSet, ruleSet)
		meta.RuleSetIDs = append(meta.RuleSetIDs, store.NewUUID())
		return nil
	})
}

func (h *Handler) handleUpdateRuleSet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var ruleSet option.RuleSet
	if err := json.Unmarshal(body, &ruleSet); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		index := h.findRuleSetIndexByID(id)
		if index < 0 {
			return errors.New("rule_set 不存在: " + id)
		}
		options.Route.RuleSet[index] = ruleSet
		return nil
	})
}

func (h *Handler) handleDeleteRuleSet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	force := r.URL.Query().Get("force") == "true"
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findRuleSetIndexByID(id)
		if index < 0 {
			return errors.New("rule_set 不存在: " + id)
		}
		// 收集被引用 tag（一个定义可带多个 tag）
		var tags []string
		for _, tag := range options.Route.RuleSet[index].Tag {
			tags = append(tags, tag)
		}
		var refs []string
		for _, tag := range tags {
			refs = append(refs, ruleSetsReferencedBy(options, tag)...)
		}
		if len(refs) > 0 {
			if !force {
				return &GroupReferenceError{Tag: tags[0], References: refs}
			}
			for _, tag := range tags {
				if options.Route != nil {
					for i := range options.Route.Rules {
						if err := removeRuleSetRef(h.ctx(r), &options.Route.Rules[i], tag); err != nil {
							return err
						}
					}
				}
				if options.DNS != nil {
					for i := range options.DNS.Rules {
						if err := removeRuleSetRef(h.ctx(r), &options.DNS.Rules[i], tag); err != nil {
							return err
						}
					}
				}
			}
		}
		options.Route.RuleSet = append(options.Route.RuleSet[:index], options.Route.RuleSet[index+1:]...)
		meta.RuleSetIDs = append(meta.RuleSetIDs[:index], meta.RuleSetIDs[index+1:]...)
		return nil
	})
}
