package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestRuleSetRefs(t *testing.T) {
	t.Run("single string", func(t *testing.T) {
		got := ruleSetRefs(map[string]any{"rule_set": "geo-cn"})
		if !reflect.DeepEqual(got, []string{"geo-cn"}) {
			t.Errorf("got %v, want [geo-cn]", got)
		}
	})
	t.Run("array", func(t *testing.T) {
		got := ruleSetRefs(map[string]any{"rule_set": []any{"a", "b"}})
		if !reflect.DeepEqual(got, []string{"a", "b"}) {
			t.Errorf("got %v, want [a b]", got)
		}
	})
	t.Run("absent", func(t *testing.T) {
		if got := ruleSetRefs(map[string]any{"protocol": "http"}); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
	t.Run("non-string items skipped", func(t *testing.T) {
		got := ruleSetRefs(map[string]any{"rule_set": []any{"a", 42}})
		if !reflect.DeepEqual(got, []string{"a"}) {
			t.Errorf("got %v, want [a]", got)
		}
	})
}

func TestRuleSetsReferencedBy(t *testing.T) {
	opts := &option.Options{
		Route: &option.RouteOptions{
			Rules: []option.Rule{
				*mustRule(t, `{"rule_set":"geo-cn","outbound":"proxy"}`),
				*mustRule(t, `{"rule_set":["geo-us","geo-jp"],"outbound":"proxy"}`),
				*mustRule(t, `{"protocol":"http","outbound":"proxy"}`),
			},
		},
		DNS: &option.DNSOptions{RawDNSOptions: option.RawDNSOptions{
			Rules: []option.DNSRule{
				*mustDNSRule(t, `{"rule_set":"geo-cn"}`),
			},
		}},
	}
	t.Run("referenced in route", func(t *testing.T) {
		got := ruleSetsReferencedBy(opts, "geo-cn")
		if len(got) != 2 { // route rule 1 + dns rule
			t.Errorf("geo-cn refs = %v, want 2", got)
		}
	})
	t.Run("referenced in array", func(t *testing.T) {
		got := ruleSetsReferencedBy(opts, "geo-jp")
		if len(got) != 1 {
			t.Errorf("geo-jp refs = %v, want 1", got)
		}
	})
	t.Run("not referenced", func(t *testing.T) {
		if got := ruleSetsReferencedBy(opts, "geo-ru"); len(got) != 0 {
			t.Errorf("geo-ru refs = %v, want empty", got)
		}
	})
	t.Run("empty options", func(t *testing.T) {
		if got := ruleSetsReferencedBy(&option.Options{}, "geo-cn"); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
}

func TestRemoveRuleSetRef(t *testing.T) {
	ctx := context.Background()
	t.Run("single value removed", func(t *testing.T) {
		rule := mustRule(t, `{"rule_set":"geo-cn","outbound":"proxy"}`)
		if err := removeRuleSetRef(ctx, rule, "geo-cn"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := string(mustMarshal(t, rule))
		if contain(out, "rule_set") {
			t.Errorf("rule_set not removed: %s", out)
		}
		if !contain(out, `"outbound":"proxy"`) {
			t.Errorf("action lost: %s", out)
		}
	})
	t.Run("array partial filter", func(t *testing.T) {
		rule := mustRule(t, `{"rule_set":["a","geo-cn","b"],"outbound":"proxy"}`)
		if err := removeRuleSetRef(ctx, rule, "geo-cn"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := string(mustMarshal(t, rule))
		if contain(out, "geo-cn") {
			t.Errorf("geo-cn still present: %s", out)
		}
		if !contain(out, `"a"`) || !contain(out, `"b"`) {
			t.Errorf("other refs lost: %s", out)
		}
	})
	t.Run("array fully removed", func(t *testing.T) {
		rule := mustRule(t, `{"rule_set":["geo-cn"],"outbound":"proxy"}`)
		if err := removeRuleSetRef(ctx, rule, "geo-cn"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if contain(string(mustMarshal(t, rule)), "rule_set") {
			t.Errorf("rule_set not removed: %s", string(mustMarshal(t, rule)))
		}
	})
	t.Run("not referenced unchanged", func(t *testing.T) {
		rule := mustRule(t, `{"rule_set":"geo-us","outbound":"proxy"}`)
		if err := removeRuleSetRef(ctx, rule, "geo-cn"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !contain(string(mustMarshal(t, rule)), `"rule_set":"geo-us"`) {
			t.Errorf("rule changed: %s", string(mustMarshal(t, rule)))
		}
	})
	t.Run("unsupported type", func(t *testing.T) {
		var obj = struct {
		RuleSet string `json:"rule_set"`
	}{RuleSet: "geo-cn"}
		if err := removeRuleSetRef(ctx, &obj, "geo-cn"); err == nil {
			t.Error("expected error for unsupported type")
		}
	})
}

func TestRemoveKeyRecursive(t *testing.T) {
	obj := map[string]any{
		"a": map[string]any{"certificate_provider": "cp1", "keep": "x"},
		"b": []any{
			map[string]any{"certificate_provider": "cp1"},
			map[string]any{"certificate_provider": "cp2"},
			"plain",
		},
		"c": "certificate_provider", // 非 map/数组的值不应被触碰
	}
	removeKeyRecursive(obj, "certificate_provider", "cp1")
	a := obj["a"].(map[string]any)
	if _, ok := a["certificate_provider"]; ok {
		t.Errorf("nested map key not removed: %v", a)
	}
	if a["keep"] != "x" {
		t.Errorf("sibling key lost: %v", a)
	}
	b := obj["b"].([]any)
	first := b[0].(map[string]any)
	if _, ok := first["certificate_provider"]; ok {
		t.Errorf("array item key not removed: %v", first)
	}
	second := b[1].(map[string]any)
	if second["certificate_provider"] != "cp2" {
		t.Errorf("non-matching value removed: %v", second)
	}
	if obj["c"] != "certificate_provider" {
		t.Errorf("plain string value altered: %v", obj["c"])
	}
}
