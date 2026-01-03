package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-auto-trade/internal/infrastructure/config"
	httpapi "ai-auto-trade/internal/interface/http"
)

const (
	errUnauthorized     = "AUTH_UNAUTHORIZED"
	errForbidden        = "AUTH_FORBIDDEN"
	errInvalidCreds     = "AUTH_INVALID_CREDENTIALS"
	errAnalysisNotReady = "ANALYSIS_NOT_READY"
)

// TestMvpE2EFlow 覆蓋登入、回補（含分析）與健康檢查。
func TestMvpE2EFlow(t *testing.T) {
	cfg := config.Config{Auth: config.AuthConfig{Secret: "test-secret"}, Ingestion: config.IngestionConfig{UseSynthetic: true}}
	srv := httpapi.NewServer(cfg, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	adminToken := login(t, ts, "admin@example.com", "password123")
	postJSON(t, ts, "/api/admin/ingestion/backfill", adminToken, map[string]interface{}{
		"start_date":   "2025-12-01",
		"end_date":     "2025-12-01",
		"run_analysis": true,
	}, http.StatusOK)

	userToken := login(t, ts, "user@example.com", "password123")
	getJSON(t, ts, "/api/analysis/summary", userToken, http.StatusNotFound)

	res := getJSON(t, ts, "/api/health", "", http.StatusOK)
	if !res.Success {
		t.Fatalf("health should be success")
	}
}

// TestAuthErrors 檢查未帶 token、錯誤密碼、權限不足的行為。
func TestAuthErrors(t *testing.T) {
	cfg := config.Config{Auth: config.AuthConfig{Secret: "test-secret"}, Ingestion: config.IngestionConfig{UseSynthetic: true}}
	srv := httpapi.NewServer(cfg, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := getJSON(t, ts, "/api/analysis/summary", "", http.StatusUnauthorized)
	if resp.ErrorCode != errUnauthorized {
		t.Fatalf("expected error_code=%s got=%s", errUnauthorized, resp.ErrorCode)
	}

	fail := postJSON(t, ts, "/api/auth/login", "", map[string]string{
		"email":    "user@example.com",
		"password": "wrong",
	}, http.StatusUnauthorized)
	if fail.ErrorCode != errInvalidCreds {
		t.Fatalf("expected error_code=%s got=%s", errInvalidCreds, fail.ErrorCode)
	}

	userToken := login(t, ts, "user@example.com", "password123")
	forbidden := postJSON(t, ts, "/api/admin/ingestion/backfill", userToken, map[string]interface{}{
		"start_date":   "2025-12-01",
		"end_date":     "2025-12-01",
		"run_analysis": true,
	}, http.StatusForbidden)
	if forbidden.ErrorCode != errForbidden {
		t.Fatalf("expected forbidden for user")
	}
}

// --- helpers ---

type apiError struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
}

func login(t *testing.T, ts *httptest.Server, email, password string) string {
	resp := postJSON(t, ts, "/api/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	}, http.StatusOK)

	var body struct {
		Success     bool   `json:"success"`
		AccessToken string `json:"access_token"`
	}
	decode(t, resp.RawBody, &body)
	if !body.Success || body.AccessToken == "" {
		t.Fatalf("login failed for %s", email)
	}
	return body.AccessToken
}

type apiResponse struct {
	apiError
	Status  int
	RawBody []byte
}

func postJSON(t *testing.T, ts *httptest.Server, path, token string, payload interface{}, expect int) apiResponse {
	buf, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()
	body := decodeError(t, res)
	if res.StatusCode != expect {
		t.Fatalf("POST %s expected %d got %d (code=%s err=%s)", path, expect, res.StatusCode, body.ErrorCode, body.Error)
	}
	body.Status = res.StatusCode
	return body
}

func getJSON(t *testing.T, ts *httptest.Server, path, token string, expect int) apiResponse {
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()
	body := decodeError(t, res)
	if res.StatusCode != expect {
		t.Fatalf("GET %s expected %d got %d (code=%s err=%s)", path, expect, res.StatusCode, body.ErrorCode, body.Error)
	}
	body.Status = res.StatusCode
	return body
}

func decodeError(t *testing.T, res *http.Response) apiResponse {
	var body apiError
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return apiResponse{apiError: body, RawBody: raw}
}

func decode(t *testing.T, raw []byte, out interface{}) {
	if len(raw) == 0 {
		return
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func parse(t *testing.T, raw []byte, out interface{}) {
	decode(t, raw, out)
}

func requireSuccess(t *testing.T, resp apiResponse) {
	if !resp.Success && resp.Status < 400 {
		t.Fatalf("expected success but got error_code=%s err=%s", resp.ErrorCode, resp.Error)
	}
}

func floatPtr(f float64) *float64 { return &f }
