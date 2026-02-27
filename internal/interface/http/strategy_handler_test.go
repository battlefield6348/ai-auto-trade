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

	t.Run("ListStrategies_Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/strategies", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("CreateStrategy_NoDBError", func(t *testing.T) {
		// This should fail gracefully because db is nil in server
		body := map[string]interface{}{
			"name":        "Test Strategy",
			"slug":        "test-strategy",
			"base_symbol": "BTCUSDT",
			"rules": []map[string]interface{}{
				{"rule_type": "entry", "type": "BASE_SCORE"},
				{"rule_type": "exit", "type": "BASE_SCORE"},
			},
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies", bytes.NewBuffer(jsonBody))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		// Expect 500 because db is nil, but successfully caught by Execute check
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Activate_Strategy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/s1/activate", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Deactivate_Strategy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/s1/deactivate", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Run_Strategy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/s1/run", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		// Might be 200 if it just triggers a goroutine or 500 if error
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})
}
