package api

import (
	"reflect"
	"testing"

	"github.com/sagernet/sing-box/option"
)

func TestRemoveString(t *testing.T) {
	cases := []struct {
		name string
		list []string
		s    string
		want []string
	}{
		{"remove existing", []string{"a", "b", "c"}, "b", []string{"a", "c"}},
		{"remove first", []string{"a", "b"}, "a", []string{"b"}},
		{"remove last", []string{"a", "b"}, "b", []string{"a"}},
		{"not present", []string{"a", "b"}, "z", []string{"a", "b"}},
		{"empty list", nil, "a", nil},
		{"duplicates", []string{"a", "a", "b", "a"}, "a", []string{"b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := removeString(tc.list, tc.s)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("removeString(%v, %q) = %v, want %v", tc.list, tc.s, got, tc.want)
			}
		})
	}
}

func selectorOutbound(tag string, members ...string) option.Outbound {
	return option.Outbound{Type: "selector", Tag: tag, Options: &option.SelectorOutboundOptions{Outbounds: members}}
}

func urlTestOutbound(tag string, members ...string) option.Outbound {
	return option.Outbound{Type: "urltest", Tag: tag, Options: &option.URLTestOutboundOptions{Outbounds: members}}
}

func TestGroupReferencesTag(t *testing.T) {
	opts := &option.Options{
		Outbounds: []option.Outbound{
			selectorOutbound("g1", "a", "b"),
			urlTestOutbound("g2", "b", "c"),
			{Type: "direct", Tag: "d"},
		},
	}
	t.Run("selector member", func(t *testing.T) {
		got := groupReferencesTag(opts, "a")
		if !reflect.DeepEqual(got, []string{"g1"}) {
			t.Errorf("got %v, want [g1]", got)
		}
	})
	t.Run("both groups", func(t *testing.T) {
		got := groupReferencesTag(opts, "b")
		if !reflect.DeepEqual(got, []string{"g1", "g2"}) {
			t.Errorf("got %v, want [g1 g2]", got)
		}
	})
	t.Run("no reference", func(t *testing.T) {
		got := groupReferencesTag(opts, "d")
		if len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
	t.Run("empty options", func(t *testing.T) {
		if got := groupReferencesTag(&option.Options{}, "a"); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
}

func TestRemoveTagFromGroups(t *testing.T) {
	opts := &option.Options{
		Outbounds: []option.Outbound{
			selectorOutbound("g1", "a", "b", "a"),
			urlTestOutbound("g2", "a"),
			{Type: "direct", Tag: "a"},
		},
	}
	removeTagFromGroups(opts, "a")
	got1 := opts.Outbounds[0].Options.(*option.SelectorOutboundOptions).Outbounds
	if !reflect.DeepEqual(got1, []string{"b"}) {
		t.Errorf("selector members = %v, want [b]", got1)
	}
	got2 := opts.Outbounds[1].Options.(*option.URLTestOutboundOptions).Outbounds
	if len(got2) != 0 {
		t.Errorf("urltest members = %v, want empty", got2)
	}
	// 非 selector/urltest 的 outbound 不应被改动
	if opts.Outbounds[2].Tag != "a" {
		t.Errorf("direct outbound tag changed to %q", opts.Outbounds[2].Tag)
	}
}
