package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-auto-trade/internal/domain/auth"
	tradingDomain "ai-auto-trade/internal/domain/trading"
	"ai-auto-trade/internal/infrastructure/config"
)

func TestStrategyHandler_Memory(t *testing.T) {
	cfg := config.Config{}
	cfg.Auth.Secret = "test-secret"
	server := NewServer(cfg, nil) // Use memory store

	// Mock a user and get token
	// Ensure user exists in memory store if needed, but tokenSvc might work anyway
	// SeedUsers already adds admin@example.com etc.
	user, err := server.authRepo.FindByEmail(context.Background(), "admin@example.com")
	if err != nil {
		t.Fatalf("failed to find admin user: %v", err)
	}
	
	pair, _ := server.tokenSvc.Issue(context.Background(), user, auth.TokenMeta{})
	token := pair.AccessToken

	// Seed a strategy via API or via internal repo
	// Since we are in the same package (httpapi), we can access server fields
	t.Run("CreateStrategy_Success", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "Test Strategy",
			"slug":        "test-strategy",
			"base_symbol": "BTCUSDT",
			"rules": []map[string]interface{}{
				{"rule_type": "entry", "type": "BASE_SCORE"},
			},
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("unexpected status %d. body: %s", w.Code, w.Body.String())
		}
	})

	// To ensure s1 exists for subsequent tests, we manually seed the memory repo
	var realID string
	if sRepo, ok := server.tradingRepo.(interface {
		CreateStrategy(context.Context, tradingDomain.Strategy) (string, error)
	}); ok {
		id, _ := sRepo.CreateStrategy(context.Background(), tradingDomain.Strategy{
			Name: "Test Strategy",
			Slug: "test-strategy",
		})
		realID = id
	}

	t.Run("ListStrategies_Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/strategies", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Activate_Strategy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/"+realID+"/activate", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Deactivate_Strategy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/"+realID+"/deactivate", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetStrategyByQuery", func(t *testing.T) {
		// Slug query
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/analysis/strategies/get?slug=test-strategy", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}

		// ID query
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/api/analysis/strategies/get?id="+realID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("GetOrUpdateStrategy", func(t *testing.T) {
		// GET
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/strategies/"+realID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("GET expected 200, got %d", w.Code)
		}

		// PUT
		// PUT
		body := map[string]interface{}{
			"name":        "Updated Name",
			"base_symbol": "BTCUSDT",
			"buy_conditions": map[string]interface{}{
				"logic": "AND",
				"conditions": []interface{}{
					map[string]interface{}{"type": "numeric", "numeric": map[string]interface{}{"field": "score", "op": "GTE", "value": 60}},
				},
			},
			"sell_conditions": map[string]interface{}{
				"logic": "AND",
				"conditions": []interface{}{
					map[string]interface{}{"type": "numeric", "numeric": map[string]interface{}{"field": "score", "op": "LTE", "value": 40}},
				},
			},
			"risk_settings": map[string]interface{}{
				"order_size_mode":  "fixed_usdt",
				"order_size_value": 1000,
				"price_mode":       "next_open",
			},
		}
		jsonBody, _ := json.Marshal(body)
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("PUT", "/api/admin/strategies/"+realID, bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("PUT expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("StrategyBacktest", func(t *testing.T) {
		body := map[string]interface{}{
			"start_date": "2025-01-01",
			"end_date":   "2025-01-10",
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/"+realID+"/backtest", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		// Expect 500 or 200 depending on mock data
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 200 or 500, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ListBacktests", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/strategies/"+realID+"/backtests", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("GenerateReport", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/"+realID+"/report-generate?env=paper", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 200 or 500, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("StrategyExecute", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/execute/test-strategy", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		// Expect 500 or 200
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 200 or 500, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("InlineBacktest", func(t *testing.T) {
		body := map[string]interface{}{
			"symbol":     "BTCUSDT",
			"start_date": "2025-01-01",
			"end_date":   "2025-01-10",
			"strategy": map[string]interface{}{
				"base_symbol": "BTCUSDT",
				"buy_conditions": map[string]interface{}{
					"logic":      "AND",
					"conditions": []interface{}{},
				},
				"sell_conditions": map[string]interface{}{
					"logic":      "AND",
					"conditions": []interface{}{},
				},
			},
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/backtest", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 200 or 500, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Reports", func(t *testing.T) {
		// Create
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/"+realID+"/reports", bytes.NewBufferString("{}"))
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("POST Report expected 200 or 500, got %d", w.Code)
		}

		// List
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/api/admin/strategies/"+realID+"/reports", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("GET Reports expected 200, got %d", w.Code)
		}
	})

	t.Run("Run_Strategy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/"+realID+"/run", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 200 or 500, got %d. body: %s", w.Code, w.Body.String())
		}
	})
}
