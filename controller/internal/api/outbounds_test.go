package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/sagernet/sing-box/include"
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

func TestSyncTagToGroups(t *testing.T) {
	opts := &option.Options{
		Outbounds: []option.Outbound{
			selectorOutbound("proxy", "node-1"),
			urlTestOutbound("auto", "node-1"),
			selectorOutbound("other", "node-2"),
		},
	}

	syncTagToGroups(opts, "node-new", []string{"proxy", "auto"})

	proxyMembers := opts.Outbounds[0].Options.(*option.SelectorOutboundOptions).Outbounds
	if !reflect.DeepEqual(proxyMembers, []string{"node-1", "node-new"}) {
		t.Errorf("proxy members = %v, want [node-1 node-new]", proxyMembers)
	}

	autoMembers := opts.Outbounds[1].Options.(*option.URLTestOutboundOptions).Outbounds
	if !reflect.DeepEqual(autoMembers, []string{"node-1", "node-new"}) {
		t.Errorf("auto members = %v, want [node-1 node-new]", autoMembers)
	}

	otherMembers := opts.Outbounds[2].Options.(*option.SelectorOutboundOptions).Outbounds
	if !reflect.DeepEqual(otherMembers, []string{"node-2"}) {
		t.Errorf("other members = %v, want [node-2]", otherMembers)
	}
}

func TestUpdateTagGroups(t *testing.T) {
	opts := &option.Options{
		Outbounds: []option.Outbound{
			selectorOutbound("g1", "node-a", "node-b"),
			selectorOutbound("g2", "node-a"),
			urlTestOutbound("g3", "node-c"),
		},
	}

	// 目标让 node-a 只存在于 g2 和 g3 中（从 g1 移除，在 g2 保留，加入 g3）
	updateTagGroups(opts, "node-a", []string{"g2", "g3"})

	g1Members := opts.Outbounds[0].Options.(*option.SelectorOutboundOptions).Outbounds
	if !reflect.DeepEqual(g1Members, []string{"node-b"}) {
		t.Errorf("g1 members = %v, want [node-b]", g1Members)
	}

	g2Members := opts.Outbounds[1].Options.(*option.SelectorOutboundOptions).Outbounds
	if !reflect.DeepEqual(g2Members, []string{"node-a"}) {
		t.Errorf("g2 members = %v, want [node-a]", g2Members)
	}

	g3Members := opts.Outbounds[2].Options.(*option.URLTestOutboundOptions).Outbounds
	if !reflect.DeepEqual(g3Members, []string{"node-c", "node-a"}) {
		t.Errorf("g3 members = %v, want [node-c node-a]", g3Members)
	}
}

func TestParseOutboundPayload(t *testing.T) {
	ctx := include.Context(context.Background())

	t.Run("with attach_groups", func(t *testing.T) {
		body := []byte(`{"type":"direct","tag":"my-direct","attach_groups":["Proxy","auto"]}`)
		outbound, attachGroups, err := parseOutboundPayload(ctx, body)
		if err != nil {
			t.Fatalf("parseOutboundPayload failed: %v", err)
		}
		if outbound.Tag != "my-direct" || outbound.Type != "direct" {
			t.Errorf("got outbound %v, want tag=my-direct type=direct", outbound)
		}
		if attachGroups == nil || !reflect.DeepEqual(*attachGroups, []string{"Proxy", "auto"}) {
			t.Errorf("got attachGroups %v, want [Proxy auto]", attachGroups)
		}
	})

	t.Run("without attach_groups", func(t *testing.T) {
		body := []byte(`{"type":"direct","tag":"my-direct-2"}`)
		outbound, attachGroups, err := parseOutboundPayload(ctx, body)
		if err != nil {
			t.Fatalf("parseOutboundPayload failed: %v", err)
		}
		if outbound.Tag != "my-direct-2" {
			t.Errorf("got tag %s, want my-direct-2", outbound.Tag)
		}
		if attachGroups != nil {
			t.Errorf("got attachGroups %v, want nil", attachGroups)
		}
	})
}


