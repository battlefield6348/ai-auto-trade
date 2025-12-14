package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "ai-auto-trade/internal/interface/http"
)

// 目前驗證路由存在與統一錯誤格式，後續會用真實端到端流程替換。
func TestRoutesSkeleton(t *testing.T) {
	srv := httpapi.NewServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	t.Run("ping", func(t *testing.T) {
		res, err := http.Get(ts.URL + "/api/ping")
		if err != nil {
			t.Fatalf("ping request failed: %v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 got %d", res.StatusCode)
		}
		var body struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if !body.Success || body.Message != "pong" {
			t.Fatalf("unexpected ping response: %+v", body)
		}
	})

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/auth/login"},
		{http.MethodPost, "/api/admin/ingestion/daily"},
		{http.MethodPost, "/api/admin/analysis/daily"},
		{http.MethodGet, "/api/analysis/daily"},
		{http.MethodGet, "/api/screener/strong-stocks"},
	}
	for _, ep := range endpoints {
		t.Run(ep.path, func(t *testing.T) {
			req, err := http.NewRequest(ep.method, ts.URL+ep.path, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("do request: %v", err)
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusNotImplemented {
				t.Fatalf("expected 501 got %d", res.StatusCode)
			}
			var body struct {
				Success   bool   `json:"success"`
				ErrorCode string `json:"error_code"`
			}
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.ErrorCode != "NOT_IMPLEMENTED" {
				t.Fatalf("unexpected error_code: %s", body.ErrorCode)
			}
		})
	}
}
