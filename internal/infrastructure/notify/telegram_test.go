package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegramClient_SendMessage(t *testing.T) {
	t.Run("nil_client", func(t *testing.T) {
		var c *TelegramClient
		err := c.SendMessage(context.Background(), "msg")
		if err == nil || err.Error() != "telegram client is nil" {
			t.Errorf("expected nil client error, got %v", err)
		}
	})

	t.Run("missing_config", func(t *testing.T) {
		c := NewTelegramClient("", 0, "")
		err := c.SendMessage(context.Background(), "msg")
		if err == nil || err.Error() != "telegram token or chat_id missing" {
			t.Error("expected missing config error")
		}
	})

	t.Run("success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer ts.Close()

		c := NewTelegramClient("tok", 123, "PROD")
		c.baseURL = ts.URL
		err := c.SendMessage(context.Background(), "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("server_error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"bad"}`))
		}))
		defer ts.Close()

		c := NewTelegramClient("tok", 123, "")
		c.baseURL = ts.URL
		err := c.SendMessage(context.Background(), "hello")
		if err == nil {
			t.Error("expected error for 400 status")
		}
	})
}
