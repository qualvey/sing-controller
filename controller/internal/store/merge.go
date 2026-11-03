package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/qualvey/sing-controller/internal/jsonc"
	"github.com/sagernet/sing/common/json/badjson"
)

var tagSections = []string{"outbounds", "endpoints"}

// FilterOverriddenTags removes base outbounds and endpoints whose tags are
// supplied by the overriding config. The source configs remain unchanged.
func FilterOverriddenTags(baseConfig, overridingConfig []byte) ([]byte, bool, error) {
	base, err := decodeConfigObject(baseConfig)
	if err != nil {
		return nil, false, fmt.Errorf("decode base config: %w", err)
	}
	override, err := decodeConfigObject(overridingConfig)
	if err != nil {
		return nil, false, fmt.Errorf("decode overriding config: %w", err)
	}

	overrideTags := make(map[string]struct{})
	for _, section := range tagSections {
		tags, err := collectTags(override[section])
		if err != nil {
			return nil, false, fmt.Errorf("read overriding %s: %w", section, err)
		}
		for tag := range tags {
			overrideTags[tag] = struct{}{}
		}
	}
	if len(overrideTags) == 0 {
		return baseConfig, false, nil
	}

	changed := false
	for _, section := range tagSections {
		itemsRaw, ok := base[section]
		if !ok || bytes.Equal(bytes.TrimSpace(itemsRaw), []byte("null")) {
			continue
		}

		var items []json.RawMessage
		if err := json.Unmarshal(itemsRaw, &items); err != nil {
			return nil, false, fmt.Errorf("read base %s: %w", section, err)
		}

		filtered := make([]json.RawMessage, 0, len(items))
		sectionChanged := false
		for _, item := range items {
			tag, err := itemTag(item)
			if err != nil {
				return nil, false, fmt.Errorf("read base %s item: %w", section, err)
			}
			if _, overridden := overrideTags[tag]; tag != "" && overridden {
				changed = true
				sectionChanged = true
				continue
			}
			filtered = append(filtered, item)
		}
		if sectionChanged {
			encoded, err := json.Marshal(filtered)
			if err != nil {
				return nil, false, fmt.Errorf("encode base %s: %w", section, err)
			}
			base[section] = encoded
		}
	}
	if !changed {
		return baseConfig, false, nil
	}

	filteredConfig, err := json.MarshalIndent(base, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("encode filtered base config: %w", err)
	}
	return filteredConfig, true, nil
}

// MergeConfigs creates the effective runtime configuration from a base config
// and an overriding fragment.  sing-box accepts multiple -c files, but that
// leaves duplicate tags to the core's merger when a generated group (for
// example Proxy) replaces a group in the base config.  Merging the objects
// here makes the replacement policy explicit and gives the core one config to
// validate.
//
// Outbounds and endpoints are merged by tag: an item in the overriding
// fragment replaces the item with the same tag in the base config.  Other
// sections use sing-box's normal merge semantics (objects are merged and
// arrays are appended).  Neither input is modified.
func MergeConfigs(baseConfig, overridingConfig []byte) ([]byte, error) {
	base, err := decodeConfigObject(baseConfig)
	if err != nil {
		return nil, fmt.Errorf("decode base config: %w", err)
	}
	override, err := decodeConfigObject(overridingConfig)
	if err != nil {
		return nil, fmt.Errorf("decode overriding config: %w", err)
	}

	baseJSON, err := json.Marshal(base)
	if err != nil {
		return nil, fmt.Errorf("encode base config for merge: %w", err)
	}
	overrideJSON, err := json.Marshal(override)
	if err != nil {
		return nil, fmt.Errorf("encode overriding config for merge: %w", err)
	}
	mergedRaw, err := badjson.MergeJSON(context.Background(), overrideJSON, baseJSON, false)
	if err != nil {
		return nil, fmt.Errorf("merge config sections: %w", err)
	}
	mergedObject, err := decodeConfigObject([]byte(mergedRaw))
	if err != nil {
		return nil, fmt.Errorf("decode merged config: %w", err)
	}

	for _, section := range tagSections {
		overrideItems, ok := override[section]
		if !ok {
			continue
		}
		merged, err := mergeTaggedSection(base[section], overrideItems)
		if err != nil {
			return nil, fmt.Errorf("merge %s: %w", section, err)
		}
		mergedObject[section] = merged
	}

	mergedConfig, err := json.MarshalIndent(mergedObject, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode merged config: %w", err)
	}
	return mergedConfig, nil
}

func mergeTaggedSection(baseRaw, overrideRaw json.RawMessage) ([]byte, error) {
	var baseItems []json.RawMessage
	if len(baseRaw) > 0 && !bytes.Equal(bytes.TrimSpace(baseRaw), []byte("null")) {
		if err := json.Unmarshal(baseRaw, &baseItems); err != nil {
			return nil, err
		}
	}
	var overrideItems []json.RawMessage
	if len(overrideRaw) > 0 && !bytes.Equal(bytes.TrimSpace(overrideRaw), []byte("null")) {
		if err := json.Unmarshal(overrideRaw, &overrideItems); err != nil {
			return nil, err
		}
	}

	overrideTags := make(map[string]struct{}, len(overrideItems))
	for _, item := range overrideItems {
		tag, err := itemTag(item)
		if err != nil {
			return nil, err
		}
		if tag != "" {
			overrideTags[tag] = struct{}{}
		}
	}

	merged := make([]json.RawMessage, 0, len(baseItems)+len(overrideItems))
	for _, item := range baseItems {
		tag, err := itemTag(item)
		if err != nil {
			return nil, err
		}
		if _, replaced := overrideTags[tag]; tag != "" && replaced {
			continue
		}
		merged = append(merged, item)
	}
	merged = append(merged, overrideItems...)
	return json.Marshal(merged)
}

func decodeConfigObject(content []byte) (map[string]json.RawMessage, error) {
	cleanContent, err := jsonc.StripComments(content)
	if err != nil {
		return nil, err
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(stripTrailingCommas(cleanContent), &config); err != nil {
		return nil, err
	}
	if config == nil {
		return nil, fmt.Errorf("expected a JSON object")
	}
	return config, nil
}

func stripTrailingCommas(input []byte) []byte {
	output := append([]byte(nil), input...)
	inString := false
	escaped := false
	for i := 0; i < len(output); i++ {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch output[i] {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		if output[i] == '"' {
			inString = true
			continue
		}
		if output[i] != ',' {
			continue
		}

		next := i + 1
		for next < len(output) && (output[next] == ' ' || output[next] == '\t' || output[next] == '\r' || output[next] == '\n') {
			next++
		}
		if next < len(output) && (output[next] == ']' || output[next] == '}') {
			output[i] = ' '
		}
	}
	return output
}

func collectTags(section json.RawMessage) (map[string]struct{}, error) {
	tags := make(map[string]struct{})
	if len(section) == 0 || bytes.Equal(bytes.TrimSpace(section), []byte("null")) {
		return tags, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(section, &items); err != nil {
		return nil, err
	}
	for _, item := range items {
		tag, err := itemTag(item)
		if err != nil {
			return nil, err
		}
		if tag != "" {
			tags[tag] = struct{}{}
		}
	}
	return tags, nil
}

func itemTag(item json.RawMessage) (string, error) {
	var tagged struct {
		Tag string `json:"tag"`
	}
	if err := json.Unmarshal(item, &tagged); err != nil {
		return "", err
	}
	return tagged.Tag, nil
}
