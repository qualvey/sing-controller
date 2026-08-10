package api

import (
	"context"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

func mustRule(t *testing.T, s string) *option.Rule {
	t.Helper()
	var r option.Rule
	if err := json.UnmarshalContext(context.Background(), []byte(s), &r); err != nil {
		t.Fatalf("unmarshal rule %q: %v", s, err)
	}
	return &r
}

func TestRuleReferencesOutbound(t *testing.T) {
	cases := []struct {
		name string
		rule string
		tag  string
		want bool
	}{
		{"direct match", `{"outbound":"proxy"}`, "proxy", true},
		{"other tag", `{"outbound":"proxy"}`, "direct", false},
		// sing-box 语义：logical 嵌套规则只放条件，action（outbound）只能在最外层
		{"logical outer action match", `{"type":"logical","mode":"or","rules":[{"protocol":"http"}],"outbound":"proxy"}`, "proxy", true},
		{"logical outer action miss", `{"type":"logical","mode":"and","rules":[{"protocol":"http"}],"outbound":"direct"}`, "proxy", false},
		{"logical no action", `{"type":"logical","mode":"or","rules":[{"protocol":"http"}]}`, "proxy", false},
		{"no outbound field", `{"protocol":"http"}`, "proxy", false},
		{"tag with space", `{"outbound":"my proxy"}`, "my proxy", true},
		{"empty rule", `{}`, "proxy", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ruleReferencesOutbound(mustRule(t, tc.rule), tc.tag)
			if got != tc.want {
				t.Errorf("ruleReferencesOutbound(%s, %q) = %v, want %v", tc.rule, tc.tag, got, tc.want)
			}
		})
	}
}

func TestRuleReferencesInbound(t *testing.T) {
	cases := []struct {
		name string
		rule string
		tag  string
		want bool
	}{
		{"direct match", `{"inbound":"vmess-in"}`, "vmess-in", true},
		{"other tag", `{"inbound":"vmess-in"}`, "socks-in", false},
		// inbound 是 match 条件：logical 场景下出现在嵌套规则内
		{"logical nested condition match", `{"type":"logical","mode":"or","rules":[{"inbound":"vmess-in","protocol":"http"}]}`, "vmess-in", true},
		{"no inbound field", `{"outbound":"proxy"}`, "vmess-in", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ruleReferencesInbound(mustRule(t, tc.rule), tc.tag)
			if got != tc.want {
				t.Errorf("ruleReferencesInbound(%s, %q) = %v, want %v", tc.rule, tc.tag, got, tc.want)
			}
		})
	}
}
