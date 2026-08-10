package api

import (
	"reflect"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestRouteRuleCount(t *testing.T) {
	t.Run("nil route", func(t *testing.T) {
		if got := routeRuleCount(option.Options{}); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})
	t.Run("with rules", func(t *testing.T) {
		opts := option.Options{Route: &option.RouteOptions{Rules: make([]option.Rule, 3)}}
		if got := routeRuleCount(opts); got != 3 {
			t.Errorf("got %d, want 3", got)
		}
	})
}

func TestDNSServerCount(t *testing.T) {
	t.Run("nil dns", func(t *testing.T) {
		if got := dnsServerCount(option.Options{}); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})
	t.Run("with servers", func(t *testing.T) {
		opts := option.Options{DNS: &option.DNSOptions{RawDNSOptions: option.RawDNSOptions{
			Servers: make([]option.DNSServerOptions, 2),
		}}}
		if got := dnsServerCount(opts); got != 2 {
			t.Errorf("got %d, want 2", got)
		}
	})
}

func TestDNSRuleCount(t *testing.T) {
	t.Run("nil dns", func(t *testing.T) {
		if got := dnsRuleCount(option.Options{}); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})
	t.Run("with rules", func(t *testing.T) {
		opts := option.Options{DNS: &option.DNSOptions{RawDNSOptions: option.RawDNSOptions{
			Rules: make([]option.DNSRule, 5),
		}}}
		if got := dnsRuleCount(opts); got != 5 {
			t.Errorf("got %d, want 5", got)
		}
	})
}

func TestDNSRuleOutboundRefs(t *testing.T) {
	t.Run("server only", func(t *testing.T) {
		got := dnsRuleOutboundRefs(map[string]any{"server": "dns1"})
		if !reflect.DeepEqual(got, []string{"dns1"}) {
			t.Errorf("got %v, want [dns1]", got)
		}
	})
	t.Run("server and outbound", func(t *testing.T) {
		got := dnsRuleOutboundRefs(map[string]any{"server": "dns1", "outbound": "proxy"})
		if !reflect.DeepEqual(got, []string{"dns1", "proxy"}) {
			t.Errorf("got %v, want [dns1 proxy]", got)
		}
	})
	t.Run("outbound array", func(t *testing.T) {
		got := dnsRuleOutboundRefs(map[string]any{"outbound": []any{"a", "b"}})
		if !reflect.DeepEqual(got, []string{"a", "b"}) {
			t.Errorf("got %v, want [a b]", got)
		}
	})
	t.Run("empty values skipped", func(t *testing.T) {
		got := dnsRuleOutboundRefs(map[string]any{"server": "", "outbound": []any{"", "a"}})
		if !reflect.DeepEqual(got, []string{"a"}) {
			t.Errorf("got %v, want [a]", got)
		}
	})
	t.Run("no refs", func(t *testing.T) {
		if got := dnsRuleOutboundRefs(map[string]any{"protocol": "http"}); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
}
