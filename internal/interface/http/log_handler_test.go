package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-auto-trade/internal/domain/auth"
	"ai-auto-trade/internal/infrastructure/config"
)

func TestLogHandler_Memory(t *testing.T) {
	cfg := config.Config{}
	server := NewServer(cfg, nil)

	// Admin token
	user, _ := server.authRepo.FindByEmail(context.Background(), "admin@example.com")
	pair, _ := server.tokenSvc.Issue(context.Background(), user, auth.TokenMeta{})
	token := pair.AccessToken

	t.Run("ListLogs", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/strategies/s1/logs", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}
