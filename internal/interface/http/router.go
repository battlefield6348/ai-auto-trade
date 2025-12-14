package httpapi

import (
	"net/http"
)

// Server 封裝 HTTP 路由與未來所需的依賴。
// 目前僅提供路由骨架，後續會注入實際的 usecase / repository。
type Server struct {
	mux *http.ServeMux
}

// NewServer 建立 API 伺服器骨架。
func NewServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// Handler 回傳路由處理器，供 HTTP server 掛載。
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.Handle("/api/ping", s.wrapGet(s.handlePing))
	s.mux.Handle("/api/auth/login", s.wrapPost(s.handleNotImplemented))
	s.mux.Handle("/api/admin/ingestion/daily", s.wrapProtected(s.handleNotImplemented))
	s.mux.Handle("/api/admin/analysis/daily", s.wrapProtected(s.handleNotImplemented))
	s.mux.Handle("/api/analysis/daily", s.wrapProtected(s.handleNotImplemented))
	s.mux.Handle("/api/screener/strong-stocks", s.wrapProtected(s.handleNotImplemented))
}

// --- handlers ---

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "pong",
	})
}

func (s *Server) handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "endpoint not implemented yet")
}

// --- routing helpers ---

func (s *Server) wrapGet(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) wrapPost(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) wrapProtected(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}
		// TODO: 之後加入 token 驗證與權限檢查
		next.ServeHTTP(w, r)
	})
}
