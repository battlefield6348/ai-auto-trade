package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-auto-trade/internal/domain/auth"
	"ai-auto-trade/internal/infrastructure/config"

	"github.com/gin-gonic/gin"
)

func TestRequireAuthMiddleware_Memory(t *testing.T) {
	cfg := config.Config{}
	cfg.Auth.Secret = "test-secret"
	server := NewServer(cfg, nil) // Memory store

	// Seed user
	user, _ := server.authRepo.FindByEmail(context.Background(), "admin@example.com")
	pair, _ := server.tokenSvc.Issue(context.Background(), user, auth.TokenMeta{})
	token := pair.AccessToken

	router := gin.New()
	router.GET("/protected", server.requireAuth(""), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("Unauthorized_NoToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Authorized_ValidToken", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Authorized_ValidCookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("Forbidden_MissingPermission", func(t *testing.T) {
		// Analyst user doesn't have PermUserManage
		user, _ := server.authRepo.FindByEmail(context.Background(), "analyst@example.com")
		pair, _ := server.tokenSvc.Issue(context.Background(), user, auth.TokenMeta{})
		token2 := pair.AccessToken

		router2 := gin.New()
		// PermUserManage is string("user:manage")
		router2.GET("/admin-only", server.requireAuth("user:manage"), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin-only", nil)
		req.Header.Set("Authorization", "Bearer "+token2)
		router2.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d. body: %s", w.Code, w.Body.String())
		}
	})
}
