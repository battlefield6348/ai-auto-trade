package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-auto-trade/internal/infrastructure/config"

	"github.com/gin-gonic/gin"
)

func TestServer_Misc(t *testing.T) {
	cfg := config.Config{}
	cfg.Auth.Secret = "test-secret"
	server := NewServer(cfg, nil)

	t.Run("Health", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/health", nil)
		server.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("Helpers", func(t *testing.T) {
		if errorString(nil) != nil {
			t.Error("expected nil")
		}
		if optionalString("") != nil {
			t.Error("expected nil")
		}
		if formatPercent(0.1234) != "12.34%" {
			t.Errorf("got %s", formatPercent(0.1234))
		}
		
		loc := taipeiLocation()
		if loc.String() != "Asia/Taipei" {
			t.Errorf("expected Asia/Taipei, got %s", loc.String())
		}
		
		if parseIntDefault("123", 0) != 123 {
			t.Error("parseIntDefault failed")
		}
		if parseIntDefault("abc", 9) != 9 {
			t.Error("parseIntDefault fallback failed")
		}
	})
}

func TestServer_DateRange(t *testing.T) {
	// Enable DebugMode for CreateTestContext if needed, but gin.New() is fine
	gin.SetMode(gin.TestMode)
	server := NewServer(config.Config{}, nil)
	
	t.Run("Default", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?foo=bar", nil)
		start, end, err := server.parseDateRange(c)
		if err != nil {
			t.Fatal(err)
		}
		if end.Sub(start) < time.Hour*24*29 {
			t.Error("expected at least 29 days range")
		}
	})

	t.Run("Valid", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?start_date=2025-01-01&end_date=2025-01-02", nil)
		start, end, err := server.parseDateRange(c)
		if err != nil {
			t.Fatal(err)
		}
		if start.Format("2006-01-02") != "2025-01-01" || end.Format("2006-01-02") != "2025-01-02" {
			t.Error("dates mismatch")
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/?start_date=bad", nil)
		_, _, err := server.parseDateRange(c)
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestMemoryRepoAdapter(t *testing.T) {
	server := NewServer(config.Config{}, nil)
	repo := server.dataRepo
	ctx := context.Background()

	t.Run("LatestAnalysisDate_Empty", func(t *testing.T) {
		_, err := repo.LatestAnalysisDate(ctx)
		if err == nil {
			t.Error("expected error for empty store")
		}
	})
}
