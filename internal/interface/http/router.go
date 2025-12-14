package httpapi

import (
	"database/sql"
	"net/http"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/auth"
	"ai-auto-trade/internal/application/mvp"
	"ai-auto-trade/internal/infra/memory"
	"ai-auto-trade/internal/infrastructure/config"
	pgrepo "ai-auto-trade/internal/infrastructure/persistence/postgres"
)

// Server 封裝 HTTP 路由與依賴。
type Server struct {
	mux        *http.ServeMux
	store      *memory.Store
	loginUC    *auth.LoginUseCase
	authz      *auth.Authorizer
	queryUC    *analysis.QueryUseCase
	screenerUC *mvp.StrongScreener
	tokenTTL   time.Duration
	db         *sql.DB
	dataRepo   DataRepository
}

// NewServer 建立 API 伺服器，預設使用記憶體資料存儲；若 db 未來可用，再注入對應 repository。
func NewServer(cfg config.Config, db *sql.DB) *Server {
	store := memory.NewStore()
	store.SeedUsers()

	var dataRepo DataRepository
	if db != nil {
		dataRepo = pgrepo.NewRepo(db)
	} else {
		dataRepo = memoryRepoAdapter{store: store}
	}

	ttl := cfg.Auth.TokenTTL
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	tokenIssuer := memory.NewMemoryTokenIssuer(store, ttl)
	loginUC := auth.NewLoginUseCase(store, memory.PlainHasher{}, tokenIssuer)
	authz := auth.NewAuthorizer(store, memory.OwnerChecker{})
	queryUC := analysis.NewQueryUseCase(dataRepo)
	screenerUC := mvp.NewStrongScreener(dataRepo)

	s := &Server{
		mux:        http.NewServeMux(),
		store:      store,
		loginUC:    loginUC,
		authz:      authz,
		queryUC:    queryUC,
		screenerUC: screenerUC,
		tokenTTL:   ttl,
		db:         db,
		dataRepo:   dataRepo,
	}
	s.registerRoutes()
	return s
}

// Handler 回傳路由處理器，供 HTTP server 掛載。
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Store 主要用於測試注入初始資料。
func (s *Server) Store() *memory.Store {
	return s.store
}

func (s *Server) registerRoutes() {
	s.mux.Handle("/api/ping", s.wrapGet(s.handlePing))
	s.mux.Handle("/api/auth/login", s.wrapPost(s.handleLogin))
	s.mux.Handle("/api/admin/ingestion/daily", s.requireAuth(auth.PermIngestionTriggerDaily, s.wrapPost(s.handleIngestionDaily)))
	s.mux.Handle("/api/admin/analysis/daily", s.requireAuth(auth.PermAnalysisTriggerDaily, s.wrapPost(s.handleAnalysisDaily)))
	s.mux.Handle("/api/analysis/daily", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisQuery)))
	s.mux.Handle("/api/screener/strong-stocks", s.requireAuth(auth.PermScreenerUse, s.wrapGet(s.handleStrongStocks)))
}
