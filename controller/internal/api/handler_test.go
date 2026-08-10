package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
)

const testConfigJSONC = `{
  // 主配置：日志与出站
  "log": { "level": "info", "timestamp": true },
  "inbounds": [],
  "outbounds": [
    { "type": "direct", "tag": "direct" } /* 直连出口 */
  ]
}
`

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(testConfigJSONC), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	st := store.New(cfgPath)
	set := settings.New(filepath.Join(dir, "settings.json"))
	return NewHandler(HandlerOptions{Store: st, Settings: set, Version: "test"})
}

func doRequest(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGetConfigRawKeepsComments(t *testing.T) {
	h := newTestHandler(t)
	rec := doRequest(t, h, http.MethodGet, "/api/config/raw", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "// 主配置") {
		t.Errorf("comments lost in GET raw:\n%s", body)
	}
	if !strings.Contains(body, "/* 直连出口 */") {
		t.Errorf("block comment lost in GET raw:\n%s", body)
	}
}

func TestPutGetConfigRawRoundTrip(t *testing.T) {
	h := newTestHandler(t)
	updated := `{
  // 修改后的配置
  "log": { "level": "debug" },
  "inbounds": [],
  "outbounds": [
    { "type": "direct", "tag": "direct" },
    { "type": "block", "tag": "block" }
  ]
}
`
	rec := doRequest(t, h, http.MethodPut, "/api/config/raw", updated)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"saved":true`) {
		t.Errorf("PUT response missing saved: %s", rec.Body.String())
	}
	// 重新读取：注释与内容原样保留
	rec2 := doRequest(t, h, http.MethodGet, "/api/config/raw", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET status = %d", rec2.Code)
	}
	got := rec2.Body.String()
	if !strings.Contains(got, "// 修改后的配置") {
		t.Errorf("comment lost after round trip:\n%s", got)
	}
	if !strings.Contains(got, `"level": "debug"`) {
		t.Errorf("content not updated:\n%s", got)
	}
}

func TestPutConfigRawRejectsInvalid(t *testing.T) {
	h := newTestHandler(t)
	cases := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"broken json", `{"log": `},
		{"bad outbound type", `{"inbounds":[],"outbounds":[{"type":"nonexistent","tag":"x"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, h, http.MethodPut, "/api/config/raw", tc.body)
			if rec.Code == http.StatusOK {
				t.Errorf("PUT accepted invalid config: %q", tc.body)
			}
		})
	}
}

func TestHealthz(t *testing.T) {
	h := newTestHandler(t)
	rec := doRequest(t, h, http.MethodGet, "/healthz", "")
	if rec.Code != http.StatusOK {
		t.Errorf("healthz status = %d, want 200", rec.Code)
	}
}
