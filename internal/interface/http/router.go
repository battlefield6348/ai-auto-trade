package httpapi

import (
	"net/http"

	"ai-auto-trade/internal/interface/http/handler"
)

// NewRouter wires HTTP routes to handlers.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/ping", handler.Ping())
	return mux
}
