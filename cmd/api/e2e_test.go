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
)

// TestMvpE2EFlow 模擬文件中的 T01、T04、T05、T08、T10、T15 主要路徑。
func TestMvpE2EFlow(t *testing.T) {
	srv := newServer(config.Config{})
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	// Admin 登入
	adminToken := login(t, ts, "admin@example.com", "password123")

	// Admin 觸發 ingestion
	postJSON(t, ts, "/api/admin/ingestion/daily", adminToken, map[string]string{
		"trade_date": "2025-12-01",
	}, http.StatusOK)

	// Admin 觸發 analysis
	postJSON(t, ts, "/api/admin/analysis/daily", adminToken, map[string]string{
		"trade_date": "2025-12-01",
	}, http.StatusOK)

	// User 登入
	userToken := login(t, ts, "user@example.com", "password123")

	// 查詢分析結果
	getJSON(t, ts, "/api/analysis/daily?trade_date=2025-12-01&limit=10", userToken, http.StatusOK)

	// 強勢股清單
	getJSON(t, ts, "/api/screener/strong-stocks?trade_date=2025-12-01&limit=10", userToken, http.StatusOK)
}

// TestAuthErrors 檢查未帶 token、錯誤密碼、權限不足的行為。
func TestAuthErrors(t *testing.T) {
	srv := newServer(config.Config{})
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	// 未帶 token 呼叫受保護 API
	resp := getJSON(t, ts, "/api/analysis/daily?trade_date=2025-12-01", "", http.StatusUnauthorized)
	if resp.ErrorCode != errCodeUnauthorized {
		t.Fatalf("expected error_code=%s got=%s", errCodeUnauthorized, resp.ErrorCode)
	}

	// 登入失敗
	fail := postJSON(t, ts, "/api/auth/login", "", map[string]string{
		"email":    "user@example.com",
		"password": "wrong",
	}, http.StatusUnauthorized)
	if fail.ErrorCode != errCodeInvalidCredentials {
		t.Fatalf("expected error_code=%s got=%s", errCodeInvalidCredentials, fail.ErrorCode)
	}

	// user 呼叫 admin API 應 Forbidden
	userToken := login(t, ts, "user@example.com", "password123")
	forbidden := postJSON(t, ts, "/api/admin/ingestion/daily", userToken, map[string]string{
		"trade_date": "2025-12-01",
	}, http.StatusForbidden)
	if forbidden.ErrorCode != errCodeForbidden {
		t.Fatalf("expected error_code=%s got=%s", errCodeForbidden, forbidden.ErrorCode)
	}
}

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
		// nothing to decode
		return
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}
