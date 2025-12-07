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
)

// TestMvpE2EFlow 模擬文件中的 T01、T04、T05、T08、T10、T15 主要路徑。
func TestMvpE2EFlow(t *testing.T) {
	srv := newServer(config.Config{}, nil)
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
	srv := newServer(config.Config{}, nil)
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

// TestIngestionRoles 檢查 admin/analyst 可觸發，user 不可。
func TestIngestionRoles(t *testing.T) {
	srv := newServer(config.Config{}, nil)
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	admin := login(t, ts, "admin@example.com", "password123")
	analyst := login(t, ts, "analyst@example.com", "password123")

	requireSuccess(t, postJSON(t, ts, "/api/admin/ingestion/daily", admin, map[string]string{
		"trade_date": "2025-12-01",
	}, http.StatusOK))

	requireSuccess(t, postJSON(t, ts, "/api/admin/ingestion/daily", analyst, map[string]string{
		"trade_date": "2025-12-01",
	}, http.StatusOK))

	user := login(t, ts, "user@example.com", "password123")
	resp := postJSON(t, ts, "/api/admin/ingestion/daily", user, map[string]string{
		"trade_date": "2025-12-01",
	}, http.StatusForbidden)
	if resp.ErrorCode != errCodeForbidden {
		t.Fatalf("expected forbidden for user")
	}
}

// TestAnalysisFlow 檢查分析未完成與完成後的查詢行為。
func TestAnalysisFlow(t *testing.T) {
	srv := newServer(config.Config{}, nil)
	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	admin := login(t, ts, "admin@example.com", "password123")
	userToken := login(t, ts, "user@example.com", "password123")

	// 未完成分析
	notReady := getJSON(t, ts, "/api/analysis/daily?trade_date=2025-12-01", userToken, http.StatusNotFound)
	if notReady.ErrorCode != errCodeAnalysisNotReady {
		t.Fatalf("expected %s got %s", errCodeAnalysisNotReady, notReady.ErrorCode)
	}

	postJSON(t, ts, "/api/admin/ingestion/daily", admin, map[string]string{
		"trade_date": "2025-12-01",
	}, http.StatusOK)
	postJSON(t, ts, "/api/admin/analysis/daily", admin, map[string]string{
		"trade_date": "2025-12-01",
	}, http.StatusOK)

	queryResp := getJSON(t, ts, "/api/analysis/daily?trade_date=2025-12-01&limit=5", userToken, http.StatusOK)
	requireSuccess(t, queryResp)

	var body struct {
		Success    bool `json:"success"`
		TotalCount int  `json:"total_count"`
		Items      []struct {
			StockCode string  `json:"stock_code"`
			Close     float64 `json:"close_price"`
			Score     float64 `json:"score"`
		} `json:"items"`
	}
	parse(t, queryResp.RawBody, &body)
	if body.TotalCount == 0 || len(body.Items) == 0 {
		t.Fatalf("expected analysis items")
	}
	if body.Items[0].StockCode == "" || body.Items[0].Close == 0 {
		t.Fatalf("missing fields in analysis item")
	}
}

// TestScreenerConditions 檢查篩選、排序與空結果。
func TestScreenerConditions(t *testing.T) {
	srv := newServer(config.Config{}, nil)
	tradeDate, _ := time.Parse("2006-01-02", "2025-12-01")
	srv.store.InsertAnalysisResult(analysisDomain.DailyAnalysisResult{
		Symbol:         "AAA",
		Market:         dataDomain.MarketTWSE,
		TradeDate:      tradeDate,
		ChangeRate:     0.02,
		Return5:        floatPtr(0.05),
		VolumeMultiple: floatPtr(2.0),
		Score:          80,
		Success:        true,
	})
	srv.store.InsertAnalysisResult(analysisDomain.DailyAnalysisResult{
		Symbol:         "BBB",
		Market:         dataDomain.MarketTWSE,
		TradeDate:      tradeDate,
		ChangeRate:     0.01,
		Return5:        floatPtr(0.03),
		VolumeMultiple: floatPtr(1.6),
		Score:          75,
		Success:        true,
	})
	srv.store.InsertAnalysisResult(analysisDomain.DailyAnalysisResult{
		Symbol:         "CCC",
		Market:         dataDomain.MarketTWSE,
		TradeDate:      tradeDate,
		ChangeRate:     -0.01,
		Return5:        floatPtr(-0.02),
		VolumeMultiple: floatPtr(1.2),
		Score:          90,
		Success:        true,
	})

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()

	token := login(t, ts, "user@example.com", "password123")

	resp := getJSON(t, ts, "/api/screener/strong-stocks?trade_date=2025-12-01&limit=5", token, http.StatusOK)
	requireSuccess(t, resp)

	var body struct {
		Success    bool `json:"success"`
		TotalCount int  `json:"total_count"`
		Items      []struct {
			StockCode string   `json:"stock_code"`
			Return5   *float64 `json:"return_5d"`
			Score     float64  `json:"score"`
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

func parse(t *testing.T, raw []byte, out interface{}) {
	decode(t, raw, out)
}

func requireSuccess(t *testing.T, resp apiResponse) {
	if !resp.Success && resp.Status < 400 {
		t.Fatalf("expected success but got error_code=%s err=%s", resp.ErrorCode, resp.Error)
	}
}

func floatPtr(f float64) *float64 { return &f }
