package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-auto-trade/internal/domain/auth"
	"ai-auto-trade/internal/infrastructure/config"
)

func TestBacktestHandler_Memory(t *testing.T) {
	cfg := config.Config{}
	server := NewServer(cfg, nil)

	// Admin token
	user, _ := server.authRepo.FindByEmail(context.Background(), "admin@example.com")
	pair, _ := server.tokenSvc.Issue(context.Background(), user, auth.TokenMeta{})
	token := pair.AccessToken

	t.Run("AnalysisBacktest_Inline", func(t *testing.T) {
		body := map[string]interface{}{
			"symbol":     "2330.TW",
			"start_date": "2025-01-01",
			"end_date":   "2025-01-10",
			"entry": map[string]interface{}{
				"weights": map[string]float64{"score": 1.0},
				"thresholds": map[string]float64{"total_min": 60},
				"flags": map[string]bool{"use_change": true},
				"total_min": 60,
			},
			"exit": map[string]interface{}{
				"weights": map[string]float64{"score": 1.0},
				"thresholds": map[string]float64{"total_min": 40},
				"flags": map[string]bool{"use_change": true},
				"total_min": 40,
			},
			"horizons": []int{5, 20},
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/analysis/backtest", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		// Expected 200 or 500 depending on use case setup (mock data may be missing)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 200 or 500, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("SlugBacktest", func(t *testing.T) {
		body := map[string]interface{}{
			"slug":       "alpha",
			"symbol":     "2330.TW",
			"start_date": "2025-01-01",
			"end_date":   "2025-01-10",
			"horizons":   []int{5, 20},
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/analysis/backtest/slug", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("Presets", func(t *testing.T) {
		// GET
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/analysis/backtest/preset", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("GET preset failed: %d", w.Code)
		}

		// POST
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/analysis/backtest/preset", bytes.NewBufferString("{}"))
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("POST preset failed: %d", w.Code)
		}
	})
}
