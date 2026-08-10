package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/sagernet/sing-box/option"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing/common/json"
)

// 证书管理：
//   - 顶层 certificate 段（option/certificate.go）：store（system/mozilla/chrome/none）、
//     certificate / certificate_path / certificate_directory_path（Listable）
//   - certificate_providers 段（option/certificate_provider.go）：多态 provider（acme），
//     引用方式为 tag 字符串或内联对象 {type, ...}；id 存旁车 meta

func (h *Handler) handleGetCertificate(w http.ResponseWriter, r *http.Request) {
	ctx := h.ctx(r)
	h.store.AlignMeta()
	cert := h.store.Options.Certificate
	providers := make([]map[string]any, 0, len(h.store.Options.CertificateProviders))
	for i := range h.store.Options.CertificateProviders {
		content, err := json.MarshalContext(ctx, &h.store.Options.CertificateProviders[i])
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		providers = append(providers, map[string]any{
			"id":       newCertProviderID(&h.store.Meta, i),
			"provider": json.RawMessage(content),
		})
	}
	var certContent json.RawMessage
	if cert != nil {
		content, err := json.Marshal(cert)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		certContent = content
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"certificate": certContent,
		"providers":   providers,
	})
}

// handlePutCertificate 整体替换顶层 certificate 段；body 为 null 时清空。
func (h *Handler) handlePutCertificate(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		if string(body) == "null" {
			options.Certificate = nil
			return nil
		}
		var cert option.CertificateOptions
		if err := json.Unmarshal(body, &cert); err != nil {
			return err
		}
		options.Certificate = &cert
		return nil
	})
}

func newCertProviderID(meta *store.Meta, index int) string {
	for len(meta.CertificateProviderIDs) <= index {
		meta.CertificateProviderIDs = append(meta.CertificateProviderIDs, "")
	}
	if meta.CertificateProviderIDs[index] == "" {
		meta.CertificateProviderIDs[index] = store.NewUUID()
	}
	return meta.CertificateProviderIDs[index]
}

func (h *Handler) findCertProviderIndexByID(id string) int {
	for i, pid := range h.store.Meta.CertificateProviderIDs {
		if pid == id {
			return i
		}
	}
	return -1
}

// certProviderReferencedBy 检查配置中是否引用该 provider tag（tls.certificate_provider 等字段）。
func certProviderReferencedBy(options *option.Options, tag string) []string {
	if tag == "" {
		return nil
	}
	var refs []string
	// 扫描所有含 tls 配置的段（outbound/endpoint/inbound），检查 "certificate_provider":"tag"
	scan := func(name string, obj any) {
		content, err := json.Marshal(obj)
		if err != nil {
			return
		}
		if strings.Contains(string(content), `"certificate_provider":"`+tag+`"`) {
			refs = append(refs, name)
		}
	}
	for i := range options.Outbounds {
		scan("outbound #"+itoa(i+1), &options.Outbounds[i])
	}
	for i := range options.Endpoints {
		scan("endpoint #"+itoa(i+1), &options.Endpoints[i])
	}
	return refs
}

func (h *Handler) handleCreateCertProvider(w http.ResponseWriter, r *http.Request) {
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var provider option.CertificateProvider
	if err := json.UnmarshalContext(h.ctx(r), body, &provider); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		options.CertificateProviders = append(options.CertificateProviders, provider)
		meta.CertificateProviderIDs = append(meta.CertificateProviderIDs, store.NewUUID())
		return nil
	})
}

func (h *Handler) handleUpdateCertProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	body, err := readRawBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var provider option.CertificateProvider
	if err := json.UnmarshalContext(h.ctx(r), body, &provider); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.commit(w, r, func(options *option.Options, _ *store.Meta) error {
		index := h.findCertProviderIndexByID(id)
		if index < 0 {
			return errors.New("certificate provider 不存在: " + id)
		}
		options.CertificateProviders[index] = provider
		return nil
	})
}

func (h *Handler) handleDeleteCertProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	force := r.URL.Query().Get("force") == "true"
	h.commit(w, r, func(options *option.Options, meta *store.Meta) error {
		index := h.findCertProviderIndexByID(id)
		if index < 0 {
			return errors.New("certificate provider 不存在: " + id)
		}
		tag := options.CertificateProviders[index].Tag
		if refs := certProviderReferencedBy(options, tag); len(refs) > 0 {
			if !force {
				return &GroupReferenceError{Tag: tag, References: refs}
			}
			// force：清掉所有 tls.certificate_provider 引用（JSON map 操作）
			if err := removeCertProviderRefs(h.ctx(r), options, tag); err != nil {
				return err
			}
		}
		options.CertificateProviders = append(options.CertificateProviders[:index], options.CertificateProviders[index+1:]...)
		meta.CertificateProviderIDs = append(meta.CertificateProviderIDs[:index], meta.CertificateProviderIDs[index+1:]...)
		return nil
	})
}

// removeCertProviderRefs 从 outbound/endpoint 的 JSON 中删除 certificate_provider 字段。
// 解码到新对象再回写（复用对象时未出现的字段会保留旧值）。
func removeCertProviderRefs(ctx context.Context, options *option.Options, tag string) error {
	strip := func(obj any) error {
		content, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		var decoded map[string]any
		if err := json.Unmarshal(content, &decoded); err != nil {
			return err
		}
		removeKeyRecursive(decoded, "certificate_provider", tag)
		content, err = json.Marshal(decoded)
		if err != nil {
			return err
		}
		switch ptr := obj.(type) {
		case *option.Outbound:
			var fresh option.Outbound
			if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
				return err
			}
			*ptr = fresh
		case *option.Endpoint:
			var fresh option.Endpoint
			if err := json.UnmarshalContext(ctx, content, &fresh); err != nil {
				return err
			}
			*ptr = fresh
		default:
			return errors.New("removeCertProviderRefs: unsupported type")
		}
		return nil
	}
	for i := range options.Outbounds {
		if err := strip(&options.Outbounds[i]); err != nil {
			return err
		}
	}
	for i := range options.Endpoints {
		if err := strip(&options.Endpoints[i]); err != nil {
			return err
		}
	}
	return nil
}

func removeKeyRecursive(obj map[string]any, key, value string) {
	for k, v := range obj {
		if k == key {
			if s, ok := v.(string); ok && s == value {
				delete(obj, k)
			}
			continue
		}
		switch child := v.(type) {
		case map[string]any:
			removeKeyRecursive(child, key, value)
		case []any:
			for _, item := range child {
				if m, ok := item.(map[string]any); ok {
					removeKeyRecursive(m, key, value)
				}
			}
		}
	}
}
