package store

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/qualvey/sing-controller/internal/jsonc"
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
		for _, item := range items {
			tag, err := itemTag(item)
			if err != nil {
				return nil, false, fmt.Errorf("read base %s item: %w", section, err)
			}
			if _, overridden := overrideTags[tag]; tag != "" && overridden {
				changed = true
				continue
			}
			filtered = append(filtered, item)
		}
		if changed {
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
