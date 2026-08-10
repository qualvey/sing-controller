package api

import (
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
