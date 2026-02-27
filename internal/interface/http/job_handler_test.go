package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-auto-trade/internal/domain/auth"
	"ai-auto-trade/internal/infrastructure/config"
)

func TestJobHandler_Memory(t *testing.T) {
	cfg := config.Config{}
	server := NewServer(cfg, nil)

	// Admin token
	user, _ := server.authRepo.FindByEmail(context.Background(), "admin@example.com")
	pair, _ := server.tokenSvc.Issue(context.Background(), user, auth.TokenMeta{})
	token := pair.AccessToken

	// Add mock history
	server.jobMu.Lock()
	server.jobHistory = append(server.jobHistory, jobRun{
		Kind:        "daily_job",
		TriggeredBy: "test",
		Start:       time.Now(),
		AnalysisOn:  true,
		AnalysisOK:  true,
	})
	server.jobMu.Unlock()

	t.Run("Status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/jobs/status", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != true {
			t.Error("expected success true")
		}
	})

	t.Run("History", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/admin/jobs/history", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["data"].([]interface{})
		if len(data) == 0 {
			t.Error("expected history data")
		}
	})
}
