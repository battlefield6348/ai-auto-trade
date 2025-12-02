package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPing_Get(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	Ping().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if string(body) != `{"message":"pong"}` {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestPing_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ping", nil)
	rec := httptest.NewRecorder()

	Ping().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
