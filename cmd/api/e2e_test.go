package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	"ai-auto-trade/internal/infrastructure/config"
	httpapi "ai-auto-trade/internal/interface/http"
)

const (
	errUnauthorized     = "AUTH_UNAUTHORIZED"
	errForbidden        = "AUTH_FORBIDDEN"
	errInvalidCreds     = "AUTH_INVALID_CREDENTIALS"
	errAnalysisNotReady = "ANALYSIS_NOT_READY"
)

// TestMvpE2EFlow 覆蓋登入、回補（含分析）、查詢與強勢股清單。
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
	getJSON(t, ts, "/api/analysis/daily?trade_date=2025-12-01&limit=10", userToken, http.StatusOK)
	getJSON(t, ts, "/api/screener/strong-stocks?trade_date=2025-12-01&limit=10", userToken, http.StatusOK)

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

	resp := getJSON(t, ts, "/api/analysis/daily?trade_date=2025-12-01", "", http.StatusUnauthorized)
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

// TestAnalysisFlow 檢查分析未完成與完成後的查詢行為。
func TestAnalysisFlow(t *testing.T) {
	cfg := config.Config{Auth: config.AuthConfig{Secret: "test-secret"}, Ingestion: config.IngestionConfig{UseSynthetic: true}}
	srv := httpapi.NewServer(cfg, nil)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	admin := login(t, ts, "admin@example.com", "password123")
	userToken := login(t, ts, "user@example.com", "password123")

	notReady := getJSON(t, ts, "/api/analysis/daily?trade_date=2025-12-01", userToken, http.StatusNotFound)
	if notReady.ErrorCode != errAnalysisNotReady {
		t.Fatalf("expected %s got %s", errAnalysisNotReady, notReady.ErrorCode)
	}

	postJSON(t, ts, "/api/admin/ingestion/backfill", admin, map[string]interface{}{
		"start_date":   "2025-12-01",
		"end_date":     "2025-12-01",
		"run_analysis": true,
	}, http.StatusOK)

	queryResp := getJSON(t, ts, "/api/analysis/daily?trade_date=2025-12-01&limit=5", userToken, http.StatusOK)
	requireSuccess(t, queryResp)

	var body struct {
		Success    bool `json:"success"`
		TotalCount int  `json:"total_count"`
		Items      []struct {
			TradingPair string  `json:"trading_pair"`
			Close       float64 `json:"close_price"`
			Score       float64 `json:"score"`
		} `json:"items"`
	}
	parse(t, queryResp.RawBody, &body)
	if body.TotalCount == 0 || len(body.Items) == 0 {
		t.Fatalf("expected analysis items")
	}
	if body.Items[0].TradingPair == "" || body.Items[0].Close == 0 {
		t.Fatalf("missing fields in analysis item")
	}
}

// TestScreenerConditions 檢查篩選、排序與空結果。
func TestScreenerConditions(t *testing.T) {
	cfg := config.Config{Auth: config.AuthConfig{Secret: "test-secret"}, Ingestion: config.IngestionConfig{UseSynthetic: true}}
	srv := httpapi.NewServer(cfg, nil)
	tradeDate, _ := time.Parse("2006-01-02", "2025-12-01")
	srv.Store().InsertAnalysisResult(analysisDomain.DailyAnalysisResult{
		Symbol:         "AAA",
		Market:         dataDomain.MarketTWSE,
		TradeDate:      tradeDate,
		ChangeRate:     0.02,
		Return5:        floatPtr(0.05),
		VolumeMultiple: floatPtr(2.0),
		Score:          80,
		Success:        true,
	})
	srv.Store().InsertAnalysisResult(analysisDomain.DailyAnalysisResult{
		Symbol:         "BBB",
		Market:         dataDomain.MarketTWSE,
		TradeDate:      tradeDate,
		ChangeRate:     0.01,
		Return5:        floatPtr(0.03),
		VolumeMultiple: floatPtr(1.6),
		Score:          75,
		Success:        true,
	})
	srv.Store().InsertAnalysisResult(analysisDomain.DailyAnalysisResult{
		Symbol:         "CCC",
		Market:         dataDomain.MarketTWSE,
		TradeDate:      tradeDate,
		ChangeRate:     -0.01,
		Return5:        floatPtr(-0.02),
		VolumeMultiple: floatPtr(1.2),
		Score:          90,
		Success:        true,
	})

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := login(t, ts, "user@example.com", "password123")

	resp := getJSON(t, ts, "/api/screener/strong-stocks?trade_date=2025-12-01&limit=5", token, http.StatusOK)
	requireSuccess(t, resp)

	var body struct {
		Success    bool `json:"success"`
		TotalCount int  `json:"total_count"`
		Items      []struct {
			TradingPair string   `json:"trading_pair"`
			Return5     *float64 `json:"return_5d"`
			Score       float64  `json:"score"`
		} `json:"items"`
	}
	parse(t, resp.RawBody, &body)
	if body.TotalCount != 2 || len(body.Items) != 2 {
		t.Fatalf("expected 2 items after filter, got %d", len(body.Items))
	}
	if body.Items[0].Score < body.Items[1].Score {
		t.Fatalf("expected sorted by score desc")
	}

	emptyResp := getJSON(t, ts, "/api/screener/strong-stocks?trade_date=2025-12-01&score_min=200", token, http.StatusOK)
	requireSuccess(t, emptyResp)
	var emptyBody struct {
		Items []interface{} `json:"items"`
	}
	parse(t, emptyResp.RawBody, &emptyBody)
	if len(emptyBody.Items) != 0 {
		t.Fatalf("expected empty items")
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
