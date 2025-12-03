package httpapi

import (
	"net/http"

	"ai-auto-trade/internal/interface/http/handler"
)

// NewRouter 建立 HTTP 路由並掛上對應 handler。
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/ping", handler.Ping())
	return mux
}
