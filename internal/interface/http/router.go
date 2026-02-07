package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"sync"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/auth"
	appStrategy "ai-auto-trade/internal/application/strategy"
	"ai-auto-trade/internal/application/trading"
	authDomain "ai-auto-trade/internal/domain/auth"
	tradingDomain "ai-auto-trade/internal/domain/trading"
	"ai-auto-trade/internal/infra/memory"
	authinfra "ai-auto-trade/internal/infrastructure/auth"
	"ai-auto-trade/internal/infrastructure/config"
	"ai-auto-trade/internal/infrastructure/external/binance"
	"ai-auto-trade/internal/infrastructure/notify"
	"ai-auto-trade/internal/infrastructure/persistence/postgres"
)

type backtestPresetStore interface {
	Save(ctx context.Context, userID string, config []byte) error
	Load(ctx context.Context, userID string) ([]byte, error)
	NotFound(err error) bool
}

const seedTimeout = 5 * time.Second

// Server 封裝 HTTP 路由與依賴。
type Server struct {
	mux           *http.ServeMux
	store         *memory.Store
	loginUC       *auth.LoginUseCase
	registerUC    *auth.RegisterUseCase
	logoutUC      *auth.LogoutUseCase
	authz         *auth.Authorizer
	queryUC       *analysis.QueryUseCase
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
	tradingSvc    *trading.Service
	jobMu         sync.Mutex
	jobHistory    []jobRun
	lastAutoRun   time.Time
	dataSource    string
	presetStore   backtestPresetStore
	scoringBtUC   *appStrategy.BacktestUseCase
	saveScoringBtUC *appStrategy.SaveScoringStrategyUseCase
	binanceClient   *binance.Client
	defaultEnv      tradingDomain.Environment


	configMu       sync.Mutex
}


// NewServer 建立 API 伺服器，預設使用記憶體資料存儲；若 db 未來可用，再注入對應 repository。
func NewServer(cfg config.Config, db *sql.DB) *Server {
	store := memory.NewStore()
	store.SeedUsers()

	var dataRepo DataRepository
	var authRepo auth.UserRepository
	var sessionStore authDomain.SessionStore
	var tradingRepo trading.Repository
	var presetStore backtestPresetStore
	if db != nil {
		dataRepo = postgres.NewRepo(db)
		repo := postgres.NewAuthRepo(db)
		authRepo = repo
		sessionStore = repo
		tradingRepo = postgres.NewTradingRepo(db)
		presetStore = postgres.NewBacktestPresetStore(db)
	} else {
		dataRepo = memoryRepoAdapter{store: store}
		authRepo = store
		sessionStore = store
		tradingRepo = memory.NewTradingRepo()
		presetStore = store
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
	registerUC := auth.NewRegisterUseCase(authRepo, authinfra.BcryptHasher{})
	logoutUC := auth.NewLogoutUseCase(tokenSvc)
	authz := auth.NewAuthorizer(authRepo, memory.OwnerChecker{})
	queryUC := analysis.NewQueryUseCase(dataRepo)
	
	binanceClient := binance.NewClient(cfg.Binance.APIKey, cfg.Binance.APISecret, cfg.Binance.UseTestnet)


	binanceAdapter := binance.NewExchangeAdapter(binanceClient)
	
	var tgClient *notify.TelegramClient
	if cfg.Notifier.Telegram.Enabled && cfg.Notifier.Telegram.Token != "" && cfg.Notifier.Telegram.ChatID != 0 {
		tgClient = notify.NewTelegramClient(cfg.Notifier.Telegram.Token, cfg.Notifier.Telegram.ChatID, cfg.Notifier.Telegram.AppTag)
	}

	var tgNotifier trading.Notifier
	if tgClient != nil {
		tgNotifier = &telegramNotifierAdapter{client: tgClient}
	}

	tradingSvc := trading.NewService(tradingRepo, dataRepo, binanceAdapter, tgNotifier)

	source := "binance"
	if cfg.Ingestion.UseSynthetic {
		source = "synthetic"
	}

	s := &Server{
		mux:           http.NewServeMux(),
		store:         store,
		loginUC:       loginUC,
		registerUC:    registerUC,
		logoutUC:      logoutUC,
		authz:         authz,
		queryUC:       queryUC,
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
		tradingSvc:    tradingSvc,
		dataSource:    source,
		presetStore:   presetStore,
	}


	if db != nil {
		s.scoringBtUC = appStrategy.NewBacktestUseCase(db, dataRepo)
		s.saveScoringBtUC = appStrategy.NewSaveScoringStrategyUseCase(db)
	}
	s.binanceClient = binanceClient
	s.defaultEnv = tradingDomain.EnvTest
	if !cfg.Binance.UseTestnet {
		s.defaultEnv = tradingDomain.EnvProd
	}

	if db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
		defer cancel()
		if err := seedAuth(ctx, authRepo); err != nil {
			println("warning: seed auth failed:", err.Error())
		}
		if err := seedScoringStrategies(ctx, db); err != nil {
			println("warning: seed strategies failed:", err.Error())
		}
	}
	s.registerRoutes()
	if s.tgClient != nil && s.tgConfig.Enabled {
	// go s.startTelegramJob() // 移除每小時摘要報告，只保留進出場通知
	}
	if s.autoInterval > 0 {
		go s.startAutoPipeline()
	}
	if s.backfillStart != "" {
		go s.startConfigBackfill()
	}
	if cfg.AutoTrade.Interval > 0 {
		worker := trading.NewBackgroundWorker(tradingSvc, cfg.AutoTrade.Interval)
		worker.Start()
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
	s.mux.Handle("/api/auth/register", s.wrapPost(s.handleRegister))
	s.mux.Handle("/api/auth/refresh", s.wrapPost(s.handleRefresh))
	s.mux.Handle("/api/auth/logout", s.wrapPost(s.handleLogout))
	s.mux.Handle("/api/admin/ingestion/daily", s.requireAuth(auth.PermIngestionTriggerDaily, s.wrapPost(s.handleIngestionDaily)))
	s.mux.Handle("/api/admin/ingestion/backfill", s.requireAuth(auth.PermIngestionTriggerBackfill, s.wrapPost(s.handleIngestionBackfill)))
	s.mux.Handle("/api/admin/analysis/daily", s.requireAuth(auth.PermAnalysisTriggerDaily, s.wrapPost(s.handleAnalysisDaily)))
	s.mux.Handle("/api/analysis/daily", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisQuery)))
	s.mux.Handle("/api/analysis/strategies", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleListScoringStrategies)))
	s.mux.Handle("/api/analysis/strategies/get", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleGetScoringStrategy)))
	s.mux.Handle("/api/analysis/strategies/save-scoring", s.requireAuth(auth.PermAnalysisQuery, s.wrapPost(s.handleSaveScoringStrategy)))
	s.mux.Handle("/api/analysis/history", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisHistory)))
	s.mux.Handle("/api/analysis/summary", s.requireAuth(auth.PermAnalysisQuery, s.wrapGet(s.handleAnalysisSummary)))
	s.mux.Handle("/api/analysis/backtest", s.requireAuth(auth.PermAnalysisQuery, s.wrapPost(s.handleAnalysisBacktest)))
	s.mux.Handle("/api/analysis/backtest/slug", s.requireAuth(auth.PermAnalysisQuery, s.wrapPost(s.handleSlugBacktest)))
	s.mux.Handle("/api/analysis/backtest/preset", s.requireAuth(auth.PermAnalysisQuery, s.wrapMethods(map[string]http.HandlerFunc{
		http.MethodGet:  s.handleGetBacktestPreset,
		http.MethodPost: s.handleSaveBacktestPreset,
	})))
	s.mux.Handle("/api/admin/jobs/status", s.requireAuth(auth.PermSystemHealth, s.wrapGet(s.handleJobsStatus)))
	s.mux.Handle("/api/admin/jobs/history", s.requireAuth(auth.PermSystemHealth, s.wrapGet(s.handleJobsHistory)))
	// 策略與交易
	s.mux.Handle("/api/admin/strategies", s.requireAuth(auth.PermStrategy, s.wrapMethods(map[string]http.HandlerFunc{
		http.MethodGet:  s.handleListStrategies,
		http.MethodPost: s.handleCreateStrategy,
	})))
	s.mux.Handle("/api/admin/strategies/backtest", s.requireAuth(auth.PermStrategy, s.wrapPost(s.handleInlineBacktest)))
	s.mux.Handle("/api/admin/strategies/execute/", s.requireAuth(auth.PermStrategy, http.HandlerFunc(s.handleStrategyExecute)))
	s.mux.Handle("/api/admin/strategies/", s.requireAuth(auth.PermStrategy, http.HandlerFunc(s.handleStrategyRoute)))
	s.mux.Handle("/api/admin/trades", s.requireAuth(auth.PermStrategy, s.wrapGet(s.handleListTrades)))
	s.mux.Handle("/api/admin/positions", s.requireAuth(auth.PermStrategy, s.wrapGet(s.handleListPositions)))
	s.mux.Handle("/api/admin/positions/", s.requireAuth(auth.PermStrategy, http.HandlerFunc(s.handlePositionRoute)))
	s.mux.Handle("/api/admin/binance/account", s.requireAuth(auth.PermStrategy, s.wrapGet(s.handleBinanceAccount)))
	s.mux.Handle("/api/admin/binance/price", s.requireAuth(auth.PermStrategy, s.wrapGet(s.handleBinancePrice)))
	s.mux.Handle("/api/admin/binance/config", s.requireAuth(auth.PermStrategy, s.wrapMethods(map[string]http.HandlerFunc{
		http.MethodGet:  s.handleGetBinanceConfig,
		http.MethodPost: s.handleUpdateBinanceConfig,
	})))
	// 前端操作介面
	s.mux.Handle("/", http.FileServer(http.Dir("web")))
}

type telegramNotifierAdapter struct {
	client *notify.TelegramClient
}

func (a *telegramNotifierAdapter) Notify(msg string) error {
	return a.client.SendMessage(context.Background(), msg)
}
