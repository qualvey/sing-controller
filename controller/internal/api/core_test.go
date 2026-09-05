package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCoreStatusUnmanaged(t *testing.T) {
	h := newTestHandler(t)
	rec := doRequest(t, h, http.MethodGet, "/api/core/status", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if res["managed"] != false {
		t.Errorf("expected managed=false, got %v", res["managed"])
	}
}

func TestCoreRestartUnmanagedFallback(t *testing.T) {
	h := newTestHandler(t)
	// 在没有 supervisor 且没有 systemd 的单元测试环境下，触发 restart 会尝试执行系统服务或热重载
	rec := doRequest(t, h, http.MethodPost, "/api/core/restart", "")
	// 应返回合理的响应（成功或带有未检测到守护管理器或系统服务明确错误，而非直接 panic）
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Errorf("expected 200 or 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
