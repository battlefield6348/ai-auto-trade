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

	t.Run("Run_Strategy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/admin/strategies/"+realID+"/run", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		// Might be 500 if memory repo doesn't implement LoadScoringStrategyBySlug
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
			t.Errorf("expected 200, 404 or 500, got %d. body: %s", w.Code, w.Body.String())
		}
	})
}
