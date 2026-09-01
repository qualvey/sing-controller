package api

import (
	"context"
	"strconv"
	"strings"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

func itoa(value int) string { return strconv.Itoa(value) }

// ruleReferencesOutbound 检查规则是否引用了指定 outbound tag。
// 简单实现：序列化规则 JSON 后查找 "outbound":"tag" 与嵌套 logical 规则。
func ruleReferencesOutbound(rule *option.Rule, tag string) bool {
	content, err := json.Marshal(rule)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), `"outbound":"`+tag+`"`)
}

// ruleReferencesInbound 检查规则是否引用了指定 inbound tag。
func ruleReferencesInbound(rule *option.Rule, tag string) bool {
	content, err := json.Marshal(rule)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), `"inbound":"`+tag+`"`)
}

// renameOutboundInRule 在 option.Rule 中将引用的 oldTag 替换为 newTag
func renameOutboundInRule(ctx context.Context, rule *option.Rule, oldTag, newTag string) error {
	content, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return err
	}
	changed := false
	if s, ok := decoded["outbound"].(string); ok && s == oldTag {
		decoded["outbound"] = newTag
		changed = true
	}
	if !changed {
		return nil
	}
	content, err = json.Marshal(decoded)
	if err != nil {
		return err
	}
	var fresh option.Rule
	if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
		return err
	}
	*rule = fresh
	return nil
}

// renameInboundInRule 在 option.Rule 中将引用的 oldTag 替换为 newTag
func renameInboundInRule(ctx context.Context, rule *option.Rule, oldTag, newTag string) error {
	content, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return err
	}
	changed := false
	if in, ok := decoded["inbound"]; ok {
		switch v := in.(type) {
		case string:
			if v == oldTag {
				decoded["inbound"] = newTag
				changed = true
			}
		case []any:
			for idx, item := range v {
				if s, ok := item.(string); ok && s == oldTag {
					v[idx] = newTag
					changed = true
				}
			}
		}
	}
	if !changed {
		return nil
	}
	content, err = json.Marshal(decoded)
	if err != nil {
		return err
	}
	var fresh option.Rule
	if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
		return err
	}
	*rule = fresh
	return nil
}

// renameServerInDNSRule 在 option.DNSRule 中将引用的 oldTag 替换为 newTag
func renameServerInDNSRule(ctx context.Context, rule *option.DNSRule, oldTag, newTag string) error {
	content, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	var decoded map[string]any
	if err := json.Unmarshal(content, &decoded); err != nil {
		return err
	}
	changed := false
	if s, ok := decoded["server"].(string); ok && s == oldTag {
		decoded["server"] = newTag
		changed = true
	}
	if !changed {
		return nil
	}
	content, err = json.Marshal(decoded)
	if err != nil {
		return err
	}
	var fresh option.DNSRule
	if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
		return err
	}
	*rule = fresh
	return nil
}

