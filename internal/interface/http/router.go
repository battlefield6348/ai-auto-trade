package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/auth"
	"ai-auto-trade/internal/application/mvp"
	"ai-auto-trade/internal/application/trading"
	authDomain "ai-auto-trade/internal/domain/auth"
	"ai-auto-trade/internal/infra/memory"
	authinfra "ai-auto-trade/internal/infrastructure/auth"
	"ai-auto-trade/internal/infrastructure/config"
	"ai-auto-trade/internal/infrastructure/notify"
	"ai-auto-trade/internal/infrastructure/persistence/postgres"
)

const seedTimeout = 5 * time.Second

// Server 封裝 HTTP 路由與依賴。
type Server struct {
	mux           *http.ServeMux
	store         *memory.Store
	loginUC       *auth.LoginUseCase
	logoutUC      *auth.LogoutUseCase
	authz         *auth.Authorizer
	queryUC       *analysis.QueryUseCase
	screenerUC    *mvp.StrongScreener
	tokenTTL      time.Duration
	refreshTTL    time.Duration
	db            *sql.DB
	dataRepo      DataRepository
	authRepo      auth.UserRepository
	tokenSvc      *authinfra.JWTIssuer
	useSynthetic  bool
	tgClient      *notify.TelegramClient
	tgConfig      config.TelegramConfig
	autoInterval  time.Duration
	backfillStart string
	presetStore   backtestPresetStore
	tradingSvc    *trading.Service
}

// NewServer 建立 API 伺服器，預設使用記憶體資料存儲；若 db 未來可用，再注入對應 repository。
func NewServer(cfg config.Config, db *sql.DB) *Server {
	store := memory.NewStore()
	store.SeedUsers()

	var dataRepo DataRepository
	var authRepo auth.UserRepository
	var sessionStore authDomain.SessionStore
	var presetStore backtestPresetStore
	var tradingRepo trading.Repository
	if db != nil {
		dataRepo = postgres.NewRepo(db)
		repo := postgres.NewAuthRepo(db)
		authRepo = repo
		sessionStore = repo
		presetStore = pgPresetStore{repo: postgres.NewBacktestPresetStore(db)}
		tradingRepo = postgres.NewTradingRepo(db)
	} else {
		dataRepo = memoryRepoAdapter{store: store}
		authRepo = store
		sessionStore = store
		presetStore = memoryPresetStore{store: store}
		tradingRepo = memory.NewTradingRepo()
	}

	ttl := cfg.Auth.TokenTTL
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	refreshTTL := cfg.Auth.RefreshTTL
	if refreshTTL == 0 {
		refreshTTL = 30 * 24 * time.Hour
	}
	tokenSvc := authinfra.NewJWTIssuer(cfg.Auth.Secret, ttl, refreshTTL, sessionStore, authRepo)
	loginUC := auth.NewLoginUseCase(authRepo, authinfra.BcryptHasher{}, tokenSvc)
	logoutUC := auth.NewLogoutUseCase(tokenSvc)
	authz := auth.NewAuthorizer(authRepo, memory.OwnerChecker{})
	queryUC := analysis.NewQueryUseCase(dataRepo)
	screenerUC := mvp.NewStrongScreener(dataRepo)
	tradingSvc := trading.NewService(tradingRepo, dataRepo)
	var tgClient *notify.TelegramClient
	if cfg.Notifier.Telegram.Enabled && cfg.Notifier.Telegram.Token != "" && cfg.Notifier.Telegram.ChatID != 0 {
		tgClient = notify.NewTelegramClient(cfg.Notifier.Telegram.Token, cfg.Notifier.Telegram.ChatID)
	}

	s := &Server{
		mux:           http.NewServeMux(),
		store:         store,
		loginUC:       loginUC,
		logoutUC:      logoutUC,
		authz:         authz,
		queryUC:       queryUC,
		screenerUC:    screenerUC,
		tokenTTL:      ttl,
		refreshTTL:    refreshTTL,
		db:            db,
		dataRepo:      dataRepo,
		authRepo:      authRepo,
		tokenSvc:      tokenSvc,
		useSynthetic:  cfg.Ingestion.UseSynthetic,
		tgClient:      tgClient,
		tgConfig:      cfg.Notifier.Telegram,
		autoInterval:  cfg.Ingestion.AutoInterval,
		backfillStart: cfg.Ingestion.BackfillStartDate,
		presetStore:   presetStore,
		tradingSvc:    tradingSvc,
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
	if s.backfillStart != "" {
		go s.startConfigBackfill()
	}
	return s
}

// Handler 回傳路由處理器，供 HTTP server 掛載。
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		s.mux.ServeHTTP(w, r)
	})
}

// Store 主要用於測試注入初始資料。
func (s *Server) Store() *memory.Store {
	return s.store
}

func (s *Server) registerRoutes() {
	s.mux.Handle("/api/ping", s.wrapGet(s.handlePing))
	s.mux.Handle("/api/health", s.wrapGet(s.handleHealth))
	s.mux.Handle("/api/auth/login", s.wrapPost(s.handleLogin))
	s.mux.Handle("/api/auth/refresh", s.wrapPost(s.handleRefresh))
	s.mux.Handle("/api/auth/logout", s.wrapPost(s.handleLogout))
	s.mux.Handle("/api/admin/ingestion/backfill", s.requireAuth(auth.PermIngestionTriggerBackfill, s.wrapPost(s.handleIngestionBackfill)))
	s.mux.Handle("/api/analysis/daily", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisQuery)))
	s.mux.Handle("/api/analysis/history", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisHistory)))
	s.mux.Handle("/api/analysis/backtest", s.requireAuth(auth.PermAnalysisQuery, s.wrapPost(s.handleAnalysisBacktest)))
	s.mux.Handle("/api/analysis/backtest/preset", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleGetBacktestPreset)))
	s.mux.Handle("/api/analysis/backtest/preset/save", s.requireAuth(auth.PermAnalysisQuery, s.wrapPost(s.handleSaveBacktestPreset)))
	s.mux.Handle("/api/analysis/backtest/presets", s.requireAuth(auth.PermAnalysisQuery, s.wrapMethods(map[string]http.HandlerFunc{
		http.MethodGet:  s.handleListBacktestPresets,
		http.MethodPost: s.handleCreateBacktestPreset,
	})))
	s.mux.Handle("/api/analysis/backtest/presets/", s.requireAuth(auth.PermAnalysisQuery, s.wrapDelete(s.handleDeleteBacktestPreset)))
	s.mux.Handle("/api/analysis/summary", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisSummary)))
	s.mux.Handle("/api/screener/strong-stocks", s.requireAuth(auth.PermScreenerUse, s.wrapGet(s.handleStrongStocks)))
	// 策略與交易
	s.mux.Handle("/api/admin/strategies", s.requireAuth(auth.PermStrategy, s.wrapMethods(map[string]http.HandlerFunc{
		http.MethodGet:  s.handleListStrategies,
		http.MethodPost: s.handleCreateStrategy,
	})))
	s.mux.Handle("/api/admin/strategies/backtest", s.requireAuth(auth.PermStrategy, s.wrapPost(s.handleInlineBacktest)))
	s.mux.Handle("/api/admin/strategies/", s.requireAuth(auth.PermStrategy, http.HandlerFunc(s.handleStrategyRoute)))
	s.mux.Handle("/api/admin/trades", s.requireAuth(auth.PermStrategy, s.wrapGet(s.handleListTrades)))
	s.mux.Handle("/api/admin/positions", s.requireAuth(auth.PermStrategy, s.wrapGet(s.handleListPositions)))
	// 前端操作介面
	s.mux.Handle("/", http.FileServer(http.Dir("web")))
}
