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

func TestTradingHandlers_Memory(t *testing.T) {
	cfg := config.Config{}
	cfg.Auth.Secret = "test-secret"
	server := NewServer(cfg, nil) // Use memory store

	user, _ := server.authRepo.FindByEmail(context.Background(), "admin@example.com")
	pair, _ := server.tokenSvc.Issue(context.Background(), user, auth.TokenMeta{})
	token := pair.AccessToken

	t.Run("ListPositions", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/positions", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("ListTrades", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/trades", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("ManualBuy_BadRequest", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/trades/manual-buy", bytes.NewBufferString("invalid json"))
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("ManualBuy_Success", func(t *testing.T) {
		body := map[string]interface{}{
			"symbol": "BTCUSDT",
			"amount": 1000,
			"env":    "paper",
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/trades/manual-buy", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ListTrades_Filter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/trades?env=paper&strategy_id=s1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("PositionClose_NotFound", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/positions/none/close", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 404 or 500, got %d", w.Code)
		}
	})
}
