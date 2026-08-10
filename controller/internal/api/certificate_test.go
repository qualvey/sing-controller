package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/service"
)

// fakeRegistry 提供 unmarshal 所需的类型注册表（真实 handler 用 box.Context 注入全量 registry）
type fakeOutboundRegistry struct{}

func (fakeOutboundRegistry) OptionTypes() []string { return []string{"vless", "direct"} }
func (fakeOutboundRegistry) CreateOptions(t string) (any, bool) {
	switch t {
	case "vless":
		return &option.VLESSOutboundOptions{}, true
	case "direct":
		return &option.DirectOutboundOptions{}, true
	}
	return nil, false
}

type fakeEndpointRegistry struct{}

func (fakeEndpointRegistry) OptionTypes() []string { return []string{"hysteria2", "tun"} }
func (fakeEndpointRegistry) CreateOptions(t string) (any, bool) {
	switch t {
	case "hysteria2":
		return &option.HysteriaRealmServiceOptions{}, true
	case "tun":
		return &option.TunInboundOptions{}, true
	}
	return nil, false
}

func registryCtx() context.Context {
	ctx := service.ContextWith[option.OutboundOptionsRegistry](context.Background(), fakeOutboundRegistry{})
	return service.ContextWith[option.EndpointOptionsRegistry](ctx, fakeEndpointRegistry{})
}

// certEndpoint 构造带 certificate_provider 引用的 endpoint（服务端 TLS，如 hysteria2 realm）
func certEndpoint(tag, provider string) option.Endpoint {
	tls := &option.InboundTLSOptions{}
	if provider != "" {
		tls.CertificateProvider = &option.CertificateProviderOptions{Tag: provider}
	}
	return option.Endpoint{
		Type: "hysteria2",
		Tag:  tag,
		Options: &option.HysteriaRealmServiceOptions{
			InboundTLSOptionsContainer: option.InboundTLSOptionsContainer{TLS: tls},
		},
	}
}

func endpointTLSProvider(t *testing.T, e option.Endpoint) string {
	t.Helper()
	realm, ok := e.Options.(*option.HysteriaRealmServiceOptions)
	if !ok {
		t.Fatalf("unexpected endpoint options type %T", e.Options)
	}
	if realm.TLS == nil || realm.TLS.CertificateProvider == nil {
		return ""
	}
	return realm.TLS.CertificateProvider.Tag
}

func TestCertProviderReferencedBy(t *testing.T) {
	opts := &option.Options{
		Endpoints: []option.Endpoint{
			certEndpoint("e1", "cp1"),
			certEndpoint("e2", "cp2"),
			{Type: "tun", Tag: "e3", Options: &option.TunInboundOptions{}},
		},
	}
	t.Run("referenced", func(t *testing.T) {
		got := certProviderReferencedBy(opts, "cp1")
		if !reflect.DeepEqual(got, []string{"endpoint #1"}) {
			t.Errorf("got %v, want [endpoint #1]", got)
		}
	})
	t.Run("not referenced", func(t *testing.T) {
		if got := certProviderReferencedBy(opts, "cp9"); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
	t.Run("empty tag", func(t *testing.T) {
		if got := certProviderReferencedBy(opts, ""); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
	t.Run("no endpoints", func(t *testing.T) {
		if got := certProviderReferencedBy(&option.Options{}, "cp1"); len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
}

func TestRemoveCertProviderRefs(t *testing.T) {
	ctx := registryCtx()
	opts := &option.Options{
		Endpoints: []option.Endpoint{
			certEndpoint("e1", "cp1"),
			certEndpoint("e2", "cp2"),
			{Type: "tun", Tag: "e3", Options: &option.TunInboundOptions{}},
		},
	}
	if err := removeCertProviderRefs(ctx, opts, "cp1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// cp1 的引用被删除（TLS 清空）
	if got := endpointTLSProvider(t, opts.Endpoints[0]); got != "" {
		t.Errorf("cp1 reference not removed, still %q", got)
	}
	// 其余字段保留
	if opts.Endpoints[0].Tag != "e1" || opts.Endpoints[0].Type != "hysteria2" {
		t.Errorf("unrelated fields lost: %+v", opts.Endpoints[0])
	}
	// cp2 不受影响
	if got := endpointTLSProvider(t, opts.Endpoints[1]); got != "cp2" {
		t.Errorf("cp2 reference lost: %q", got)
	}
}

func TestRemoveCertProviderRefsNoMatch(t *testing.T) {
	ctx := registryCtx()
	opts := &option.Options{Endpoints: []option.Endpoint{certEndpoint("e1", "cp1")}}
	if err := removeCertProviderRefs(ctx, opts, "cp9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := endpointTLSProvider(t, opts.Endpoints[0]); got != "cp1" {
		t.Errorf("endpoint changed without match: %q", got)
	}
}
