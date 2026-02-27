package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-auto-trade/internal/infrastructure/config"
)

func TestHealthHandler(t *testing.T) {
	cfg := config.Config{}
	// NewServer will apply defaults
	server := NewServer(cfg, nil)

	t.Run("Ping", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/ping", nil)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["message"] != "pong" {
			t.Errorf("expected pong, got %v", resp["message"])
		}
	})

	t.Run("Health", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/health", nil)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["health"] != "ok" {
			t.Errorf("expected ok, got %v", resp["health"])
		}
		if resp["db"] != "using_memory" {
			t.Errorf("expected using_memory, got %v", resp["db"])
		}
	})
}
