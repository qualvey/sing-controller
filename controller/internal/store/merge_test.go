package store

import (
	"encoding/json"
	"testing"
)

func TestFilterOverriddenTagsPrefersOverridingConfig(t *testing.T) {
	base := []byte(`{
		// Keep the base config's unrelated outbound.
		"outbounds": [
			{"type":"direct","tag":"direct"},
			{"type":"socks","tag":"base-only"}
		],
		"endpoints": [{"type":"wireguard","tag":"external-endpoint"}],
		"route": {"final":"direct"}
	}`)
	override := []byte(`{
		"outbounds": [
			{"type":"direct","tag":"direct"},
			{"type":"wireguard","tag":"external-endpoint"}
		]
	}`)

	filtered, changed, err := FilterOverriddenTags(base, override)
	if err != nil {
		t.Fatalf("FilterOverriddenTags returned error: %v", err)
	}
	if !changed {
		t.Fatal("expected duplicate base tags to be filtered")
	}

	var config struct {
		Outbounds []struct {
			Tag string `json:"tag"`
		} `json:"outbounds"`
		Endpoints []struct {
			Tag string `json:"tag"`
		} `json:"endpoints"`
		Route map[string]string `json:"route"`
	}
	if err := json.Unmarshal(filtered, &config); err != nil {
		t.Fatalf("decode filtered config: %v", err)
	}
	if len(config.Outbounds) != 1 || config.Outbounds[0].Tag != "base-only" {
		t.Fatalf("expected only non-conflicting base outbound, got %+v", config.Outbounds)
	}
	if len(config.Endpoints) != 0 {
		t.Fatalf("expected conflicting base endpoint to be removed, got %+v", config.Endpoints)
	}
	if config.Route["final"] != "direct" {
		t.Fatalf("expected unrelated route config to be preserved, got %+v", config.Route)
	}
	if string(base) == string(filtered) {
		t.Fatal("expected filtered output without mutating the base input")
	}
}

func TestFilterOverriddenTagsWithoutConflictsReturnsOriginal(t *testing.T) {
	base := []byte(`{"outbounds":[{"type":"direct","tag":"direct"}]}`)
	override := []byte(`{"outbounds":[{"type":"socks","tag":"proxy"}]}`)

	filtered, changed, err := FilterOverriddenTags(base, override)
	if err != nil {
		t.Fatalf("FilterOverriddenTags returned error: %v", err)
	}
	if changed {
		t.Fatal("did not expect config to change")
	}
	if string(filtered) != string(base) {
		t.Fatalf("expected original base bytes, got %s", filtered)
	}
}
