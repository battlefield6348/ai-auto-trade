package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-auto-trade/internal/infrastructure/config"
)

func TestAuthHandler_Login(t *testing.T) {
	cfg := config.Config{}
	server := NewServer(cfg, nil)

	t.Run("LoginSuccess", func(t *testing.T) {
		body := map[string]string{
			"email":    "admin@example.com",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d. body: %s", w.Code, w.Body.String())
		}

	var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["success"] != true {
			t.Errorf("expected success true, got %v", resp["success"])
		}
		if resp["access_token"] == "" {
			t.Error("expected access_token, got empty")
		}

		// Verify cookie
		cookies := w.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == "refresh_token" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected refresh_token cookie")
		}
	})

	t.Run("LoginFailure", func(t *testing.T) {
		body := map[string]string{
			"email":    "admin@example.com",
			"password": "wrong-password",
		}
		jsonBody, _ := json.Marshal(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestAuthHandler_Refresh(t *testing.T) {
	cfg := config.Config{}
	server := NewServer(cfg, nil)

	// 1. Login to get cookie
	loginBody := map[string]string{"email": "admin@example.com", "password": "password123"}
	b, _ := json.Marshal(loginBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	server.Handler().ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			refreshCookie = c
		}
	}

	t.Run("RefreshSuccess", func(t *testing.T) {
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/auth/refresh", nil)
		req.AddCookie(refreshCookie)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d. body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("RefreshNoCookie", func(t *testing.T) {
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/auth/refresh", nil)
		server.Handler().ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})
}

func TestAuthHandler_Logout(t *testing.T) {
	cfg := config.Config{}
	server := NewServer(cfg, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth/logout", nil)
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify cookie cleared
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "refresh_token" && c.MaxAge != -1 {
			t.Error("expected refresh_token cookie to be cleared")
		}
	}
}
