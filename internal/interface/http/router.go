package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/auth"
	"ai-auto-trade/internal/application/mvp"
	"ai-auto-trade/internal/infra/memory"
	authinfra "ai-auto-trade/internal/infrastructure/auth"
	"ai-auto-trade/internal/infrastructure/config"
	"ai-auto-trade/internal/infrastructure/notify"
	"ai-auto-trade/internal/infrastructure/persistence/postgres"
)

const seedTimeout = 5 * time.Second

// Server 封裝 HTTP 路由與依賴。
type Server struct {
	mux          *http.ServeMux
	store        *memory.Store
	loginUC      *auth.LoginUseCase
	authz        *auth.Authorizer
	queryUC      *analysis.QueryUseCase
	screenerUC   *mvp.StrongScreener
	tokenTTL     time.Duration
	db           *sql.DB
	dataRepo     DataRepository
	authRepo     auth.UserRepository
	tokenSvc     *authinfra.JWTIssuer
	useSynthetic bool
	tgClient     *notify.TelegramClient
	tgConfig     config.TelegramConfig
	autoInterval time.Duration
}

// NewServer 建立 API 伺服器，預設使用記憶體資料存儲；若 db 未來可用，再注入對應 repository。
func NewServer(cfg config.Config, db *sql.DB) *Server {
	store := memory.NewStore()
	store.SeedUsers()

	var dataRepo DataRepository
	var authRepo auth.UserRepository
	if db != nil {
		dataRepo = postgres.NewRepo(db)
		authRepo = postgres.NewAuthRepo(db)
	} else {
		dataRepo = memoryRepoAdapter{store: store}
		authRepo = store
	}

	ttl := cfg.Auth.TokenTTL
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	tokenSvc := authinfra.NewJWTIssuer(cfg.Auth.Secret, ttl)
	loginUC := auth.NewLoginUseCase(authRepo, authinfra.BcryptHasher{}, tokenSvc)
	authz := auth.NewAuthorizer(authRepo, memory.OwnerChecker{})
	queryUC := analysis.NewQueryUseCase(dataRepo)
	screenerUC := mvp.NewStrongScreener(dataRepo)
	var tgClient *notify.TelegramClient
	if cfg.Notifier.Telegram.Enabled && cfg.Notifier.Telegram.Token != "" && cfg.Notifier.Telegram.ChatID != 0 {
		tgClient = notify.NewTelegramClient(cfg.Notifier.Telegram.Token, cfg.Notifier.Telegram.ChatID)
	}

	s := &Server{
		mux:          http.NewServeMux(),
		store:        store,
		loginUC:      loginUC,
		authz:        authz,
		queryUC:      queryUC,
		screenerUC:   screenerUC,
		tokenTTL:     ttl,
		db:           db,
		dataRepo:     dataRepo,
		authRepo:     authRepo,
		tokenSvc:     tokenSvc,
		useSynthetic: cfg.Ingestion.UseSynthetic,
		tgClient:     tgClient,
		tgConfig:     cfg.Notifier.Telegram,
		autoInterval: cfg.Ingestion.AutoInterval,
	}
	if db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
		defer cancel()
		if err := seedAuth(ctx, authRepo); err != nil {
			println("warning: seed auth failed:", err.Error())
		}
	}
	s.registerRoutes()
	if s.tgClient != nil && s.tgConfig.Enabled {
		go s.startTelegramJob()
	}
	if s.autoInterval > 0 {
		go s.startAutoPipeline()
	}
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
	s.mux.Handle("/api/health", s.wrapGet(s.handleHealth))
	s.mux.Handle("/api/auth/login", s.wrapPost(s.handleLogin))
	s.mux.Handle("/api/admin/ingestion/daily", s.requireAuth(auth.PermIngestionTriggerDaily, s.wrapPost(s.handleIngestionDaily)))
	s.mux.Handle("/api/admin/ingestion/backfill", s.requireAuth(auth.PermIngestionTriggerBackfill, s.wrapPost(s.handleIngestionBackfill)))
	s.mux.Handle("/api/admin/analysis/daily", s.requireAuth(auth.PermAnalysisTriggerDaily, s.wrapPost(s.handleAnalysisDaily)))
	s.mux.Handle("/api/analysis/daily", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisQuery)))
	s.mux.Handle("/api/analysis/history", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisHistory)))
	s.mux.Handle("/api/analysis/summary", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisSummary)))
	s.mux.Handle("/api/screener/strong-stocks", s.requireAuth(auth.PermScreenerUse, s.wrapGet(s.handleStrongStocks)))
	// 前端操作介面
	s.mux.Handle("/", http.FileServer(http.Dir("web")))
}
