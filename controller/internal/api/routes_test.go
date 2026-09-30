package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

type routeItemResponse struct {
	ID       string         `json:"id"`
	Rule     map[string]any `json:"rule"`
	Disabled bool           `json:"disabled"`
}

type listRoutesResponse struct {
	Routes []routeItemResponse `json:"routes"`
	Final  string              `json:"final"`
}

func TestRouteReorder(t *testing.T) {
	h := newTestHandler(t)

	// 1. 创建 3 条路由规则
	rule1 := `{"domain_suffix":[".google.com"],"outbound":"direct"}`
	rule2 := `{"domain_suffix":[".github.com"],"outbound":"direct"}`
	rule3 := `{"domain_suffix":[".example.com"],"outbound":"direct"}`

	for _, r := range []string{rule1, rule2, rule3} {
		rec := doRequest(t, h, http.MethodPost, "/api/routes", r)
		if rec.Code != http.StatusOK {
			t.Fatalf("create route rule failed: %d %s", rec.Code, rec.Body.String())
		}
	}

	// 2. 查询列表，获取初始顺序和 ID
	rec := doRequest(t, h, http.MethodGet, "/api/routes", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list routes failed: %d %s", rec.Code, rec.Body.String())
	}
	var listResp listRoutesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(listResp.Routes))
	}

	id1 := listResp.Routes[0].ID
	id2 := listResp.Routes[1].ID
	id3 := listResp.Routes[2].ID

	// 3. 错误校验测试
	// 3.1 数量不匹配
	badCountReq := `{"ids":["` + id1 + `","` + id2 + `"]}`
	badRec := doRequest(t, h, http.MethodPut, "/api/routes/order", badCountReq)
	if badRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for count mismatch, got %d", badRec.Code)
	}

	// 3.2 重复 ID
	dupReq := `{"ids":["` + id1 + `","` + id1 + `","` + id2 + `"]}`
	dupRec := doRequest(t, h, http.MethodPut, "/api/routes/order", dupReq)
	if dupRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate id, got %d", dupRec.Code)
	}

	// 3.3 未知 ID
	unknownReq := `{"ids":["` + id1 + `","` + id2 + `","nonexistent-uuid"]}`
	unknownRec := doRequest(t, h, http.MethodPut, "/api/routes/order", unknownReq)
	if unknownRec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown id, got %d", unknownRec.Code)
	}

	// 4. 执行成功重排: 将顺序调整为 [id3, id1, id2]
	reorderReq := `{"ids":["` + id3 + `","` + id1 + `","` + id2 + `"]}`
	okRec := doRequest(t, h, http.MethodPut, "/api/routes/order", reorderReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("reorder routes failed: %d %s", okRec.Code, okRec.Body.String())
	}

	// 5. 校验重排后的顺序与数据准确性
	recAfter := doRequest(t, h, http.MethodGet, "/api/routes", "")
	if recAfter.Code != http.StatusOK {
		t.Fatalf("list routes after reorder failed: %d %s", recAfter.Code, recAfter.Body.String())
	}
	var afterResp listRoutesResponse
	if err := json.Unmarshal(recAfter.Body.Bytes(), &afterResp); err != nil {
		t.Fatalf("decode after response: %v", err)
	}
	if len(afterResp.Routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(afterResp.Routes))
	}

	if afterResp.Routes[0].ID != id3 || afterResp.Routes[1].ID != id1 || afterResp.Routes[2].ID != id2 {
		t.Fatalf("order mismatch: got [%s, %s, %s], want [%s, %s, %s]",
			afterResp.Routes[0].ID, afterResp.Routes[1].ID, afterResp.Routes[2].ID,
			id3, id1, id2)
	}

	// 检查第一条的内容是否确实是 example.com
	rawSuffix := afterResp.Routes[0].Rule["domain_suffix"]
	hasSuffix := false
	if s, ok := rawSuffix.(string); ok && s == ".example.com" {
		hasSuffix = true
	} else if list, ok := rawSuffix.([]any); ok && len(list) > 0 && list[0] == ".example.com" {
		hasSuffix = true
	}
	if !hasSuffix {
		t.Fatalf("expected route 0 to have .example.com, got %v", afterResp.Routes[0].Rule)
	}

	// 6. 校验与 toggle 禁用状态的联动
	toggleRec := doRequest(t, h, http.MethodPut, "/api/routes/"+id3+"/toggle", `{"disabled":true}`)
	if toggleRec.Code != http.StatusOK {
		t.Fatalf("toggle failed: %d", toggleRec.Code)
	}

	recDisabled := doRequest(t, h, http.MethodGet, "/api/routes", "")
	var disabledResp listRoutesResponse
	_ = json.Unmarshal(recDisabled.Body.Bytes(), &disabledResp)
	if !disabledResp.Routes[0].Disabled {
		t.Errorf("expected route[0] (id3) to be disabled")
	}
	if disabledResp.Routes[1].Disabled {
		t.Errorf("expected route[1] (id1) to remain enabled")
	}
}
