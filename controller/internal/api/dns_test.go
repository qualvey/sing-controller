package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

func mustDNSRule(t *testing.T, s string) *option.DNSRule {
	t.Helper()
	var r option.DNSRule
	if err := json.UnmarshalContext(context.Background(), []byte(s), &r); err != nil {
		t.Fatalf("unmarshal dns rule %q: %v", s, err)
	}
	return &r
}

func dnsOptionsWith(servers []option.DNSServerOptions, rules []option.DNSRule, final string) *option.Options {
	dns := &option.DNSOptions{RawDNSOptions: option.RawDNSOptions{Final: final, Servers: servers, Rules: rules}}
	return &option.Options{DNS: dns}
}

func TestDNSServerIndex(t *testing.T) {
	servers := []option.DNSServerOptions{{Tag: "local"}, {Tag: "remote"}}
	opts := dnsOptionsWith(servers, nil, "")

	t.Run("found", func(t *testing.T) {
		if got := dnsServerIndex(opts, "remote"); got != 1 {
			t.Errorf("got %d, want 1", got)
		}
	})
	t.Run("not found", func(t *testing.T) {
		if got := dnsServerIndex(opts, "none"); got != -1 {
			t.Errorf("got %d, want -1", got)
		}
	})
	t.Run("nil dns", func(t *testing.T) {
		if got := dnsServerIndex(&option.Options{}, "local"); got != -1 {
			t.Errorf("got %d, want -1", got)
		}
	})
}

func TestDNSReferencesTag(t *testing.T) {
	t.Run("final reference", func(t *testing.T) {
		opts := dnsOptionsWith(nil, nil, "local")
		got := dnsReferencesTag(opts, "local")
		if !reflect.DeepEqual(got, []string{"dns.final"}) {
			t.Errorf("got %v, want [dns.final]", got)
		}
	})
	t.Run("rule server reference", func(t *testing.T) {
		opts := dnsOptionsWith(nil, []option.DNSRule{*mustDNSRule(t, `{"server":"remote"}`)}, "")
		got := dnsReferencesTag(opts, "remote")
		if len(got) != 1 {
			t.Errorf("got %v, want 1 ref", got)
		}
	})
	t.Run("rule outbound reference", func(t *testing.T) {
		opts := dnsOptionsWith(nil, []option.DNSRule{*mustDNSRule(t, `{"outbound":"proxy"}`)}, "")
		got := dnsReferencesTag(opts, "proxy")
		if len(got) != 1 {
			t.Errorf("got %v, want 1 ref", got)
		}
	})
	t.Run("no reference", func(t *testing.T) {
		opts := dnsOptionsWith(nil, []option.DNSRule{*mustDNSRule(t, `{"server":"local"}`)}, "")
		if got := dnsReferencesTag(opts, "remote"); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
	t.Run("nil dns", func(t *testing.T) {
		if got := dnsReferencesTag(&option.Options{}, "local"); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
}

func TestRuleReferencesDNS(t *testing.T) {
	cases := []struct {
		name string
		rule string
		tag  string
		want bool
	}{
		{"server match", `{"server":"remote"}`, "remote", true},
		{"server miss", `{"server":"local"}`, "remote", false},
		{"outbound single match", `{"outbound":"proxy"}`, "proxy", true},
		{"outbound single miss", `{"outbound":"proxy"}`, "direct", false},
		{"outbound array contains", `{"outbound":["proxy","direct"]}`, "proxy", true},
		{"outbound array miss", `{"outbound":["proxy","direct"]}`, "block", false},
		{"no ref fields", `{"protocol":"http"}`, "proxy", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ruleReferencesDNS(mustDNSRule(t, tc.rule), tc.tag)
			if got != tc.want {
				t.Errorf("ruleReferencesDNS(%s, %q) = %v, want %v", tc.rule, tc.tag, got, tc.want)
			}
		})
	}
}

func TestRemoveDNSRuleRefs(t *testing.T) {
	ctx := context.Background()
	t.Run("server removed, other fields kept", func(t *testing.T) {
		rule := mustDNSRule(t, `{"server":"remote","protocol":"http","port":443}`)
		if err := removeDNSRuleRefs(ctx, rule, "remote"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out, _ := json.Marshal(rule)
		got := string(out)
		if contain(got, "server") {
			t.Errorf("server not removed: %s", got)
		}
		// 其余字段必须保留（json.Unmarshal 复用对象不清零的回归）
		if !contain(got, `"protocol":"http"`) || !contain(got, `"port":443`) {
			t.Errorf("unrelated fields lost: %s", got)
		}
	})
	t.Run("outbound single removed", func(t *testing.T) {
		rule := mustDNSRule(t, `{"outbound":"proxy"}`)
		if err := removeDNSRuleRefs(ctx, rule, "proxy"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if contain(string(mustMarshal(t, rule)), "outbound") {
			t.Errorf("outbound not removed: %s", string(mustMarshal(t, rule)))
		}
	})
	t.Run("outbound array partial filter", func(t *testing.T) {
		rule := mustDNSRule(t, `{"outbound":["proxy","direct","proxy"]}`)
		if err := removeDNSRuleRefs(ctx, rule, "proxy"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := string(mustMarshal(t, rule))
		if !contain(out, `["direct"]`) && !contain(out, `"direct"`) {
			t.Errorf("filtered outbound wrong: %s", out)
		}
		if contain(out, "proxy") {
			t.Errorf("proxy still present: %s", out)
		}
	})
	t.Run("no reference unchanged", func(t *testing.T) {
		rule := mustDNSRule(t, `{"server":"local"}`)
		if err := removeDNSRuleRefs(ctx, rule, "remote"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !contain(string(mustMarshal(t, rule)), `"server":"local"`) {
			t.Errorf("rule changed unexpectedly: %s", string(mustMarshal(t, rule)))
		}
	})
}

func contain(s, sub string) bool { return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0) }

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}
