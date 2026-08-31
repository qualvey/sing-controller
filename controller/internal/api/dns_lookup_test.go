package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDNSLookup(t *testing.T) {
	h := &Handler{}

	t.Run("empty domains", func(t *testing.T) {
		body := bytes.NewBufferString(`{"domains":[]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/dns/lookup", body)
		w := httptest.NewRecorder()

		h.handleDNSLookup(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var res struct {
			Results map[string]string `json:"results"`
		}
		if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(res.Results) != 0 {
			t.Fatalf("expected empty results, got %v", res.Results)
		}
	})

	t.Run("ip address directly returned", func(t *testing.T) {
		body := bytes.NewBufferString(`{"domains":["127.0.0.1", "1.1.1.1"]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/dns/lookup", body)
		w := httptest.NewRecorder()

		h.handleDNSLookup(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var res struct {
			Results map[string]string `json:"results"`
		}
		if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.Results["127.0.0.1"] != "127.0.0.1" {
			t.Errorf("expected 127.0.0.1 -> 127.0.0.1, got %s", res.Results["127.0.0.1"])
		}
		if res.Results["1.1.1.1"] != "1.1.1.1" {
			t.Errorf("expected 1.1.1.1 -> 1.1.1.1, got %s", res.Results["1.1.1.1"])
		}
	})

	t.Run("cached resolution", func(t *testing.T) {
		setCachedDNS("mock-test-domain.internal", "10.0.0.99", 5*time.Minute)

		body := bytes.NewBufferString(`{"domains":["mock-test-domain.internal"]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/dns/lookup", body)
		w := httptest.NewRecorder()

		h.handleDNSLookup(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var res struct {
			Results map[string]string `json:"results"`
		}
		if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if res.Results["mock-test-domain.internal"] != "10.0.0.99" {
			t.Errorf("expected mock-test-domain.internal -> 10.0.0.99, got %s", res.Results["mock-test-domain.internal"])
		}
	})
}
