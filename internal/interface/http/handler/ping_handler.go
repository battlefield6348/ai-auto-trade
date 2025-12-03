package handler

import (
	"net/http"
)

// Ping 回傳簡單的健康檢查 handler，使用 GET 時會回應 pong。
func Ping() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"pong"}`))
	})
}
