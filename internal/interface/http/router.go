package httpapi

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
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

	"github.com/gin-gonic/gin"
)

type backtestPresetStore interface {
	Save(ctx context.Context, userID string, config []byte) error
	Load(ctx context.Context, userID string) ([]byte, error)
	NotFound(err error) bool
}

const seedTimeout = 5 * time.Second

// Server 封裝 HTTP 路由與依賴。
type Server struct {
	engine        *gin.Engine
	store         *memory.Store
	loginUC       *auth.LoginUseCase
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

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	s := &Server{
		engine:        engine,
		store:         store,
		loginUC:       loginUC,
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
	return s.engine
}

// Store 主要用於測試注入初始資料。
func (s *Server) Store() *memory.Store {
	return s.store
}

func (s *Server) registerRoutes() {
	s.engine.Use(corsMiddleware())
	s.engine.Use(s.ginLogger())

	api := s.engine.Group("/api")
	{
		api.GET("/ping", s.handlePing)
		api.GET("/health", s.handleHealth)

		authG := api.Group("/auth")
		{
			authG.POST("/login", s.handleLogin)
			authG.POST("/refresh", s.handleRefresh)
			authG.POST("/logout", s.handleLogout)
		}

		admin := api.Group("/admin")
		{
			ingest := admin.Group("/ingestion")
			ingest.Use(s.requireAuth(auth.PermIngestionTriggerDaily))
			{
				ingest.POST("/daily", s.handleIngestionDaily)
				ingest.POST("/backfill", s.handleIngestionBackfill)
			}

			analysisG := admin.Group("/analysis")
			analysisG.Use(s.requireAuth(auth.PermAnalysisTriggerDaily))
			{
				analysisG.POST("/daily", s.handleAnalysisDaily)
			}

			jobs := admin.Group("/jobs")
			jobs.Use(s.requireAuth(auth.PermSystemHealth))
			{
				jobs.GET("/status", s.handleJobsStatus)
				jobs.GET("/history", s.handleJobsHistory)
			}

			strategies := admin.Group("/strategies")
			strategies.Use(s.requireAuth(auth.PermStrategy))
			{
				strategies.GET("", s.handleListStrategies)
				strategies.POST("", s.handleCreateStrategy)
				strategies.POST("/backtest", s.handleInlineBacktest)
				strategies.Any("/execute/:slug", s.handleStrategyExecute)

				instance := strategies.Group("/:id")
				{
					instance.GET("", func(c *gin.Context) { s.handleStrategyGetOrUpdate(c, c.Param("id")) })
					instance.PUT("", func(c *gin.Context) { s.handleStrategyGetOrUpdate(c, c.Param("id")) })
					instance.POST("/backtest", func(c *gin.Context) { s.handleStrategyBacktest(c, c.Param("id")) })
					instance.GET("/backtests", func(c *gin.Context) { s.handleListStrategyBacktests(c, c.Param("id")) })
					instance.POST("/run", func(c *gin.Context) { s.handleRunStrategy(c, c.Param("id")) })
					instance.POST("/activate", func(c *gin.Context) { s.handleActivateStrategy(c, c.Param("id")) })
					instance.POST("/deactivate", func(c *gin.Context) { s.handleDeactivateStrategy(c, c.Param("id")) })
					instance.GET("/reports", func(c *gin.Context) { s.handleListReports(c, c.Param("id")) })
					instance.POST("/reports", func(c *gin.Context) { s.handleCreateReport(c, c.Param("id")) })
					instance.GET("/logs", func(c *gin.Context) { s.handleListLogs(c, c.Param("id")) })
				}
			}

			trades := admin.Group("/trades")
			trades.Use(s.requireAuth(auth.PermStrategy))
			{
				trades.GET("", s.handleListTrades)
				trades.POST("/manual-buy", s.handleManualBuy)
			}

			pos := admin.Group("/positions")
			pos.Use(s.requireAuth(auth.PermStrategy))
			{
				pos.GET("", s.handleListPositions)
				pos.POST("/:id/close", func(c *gin.Context) { s.handlePositionClose(c, c.Param("id")) })
			}

			binanceG := admin.Group("/binance")
			binanceG.Use(s.requireAuth(auth.PermStrategy))
			{
				binanceG.GET("/account", s.handleBinanceAccount)
				binanceG.GET("/price", s.handleBinancePrice)
				binanceG.GET("/config", s.handleGetBinanceConfig)
				binanceG.POST("/config", s.handleUpdateBinanceConfig)
			}
		}

		// Analysis public-ish (requires analysis query perm)
		analysisQuery := api.Group("/analysis")
		analysisQuery.Use(s.requireAuth(auth.PermAnalysisQuery))
		{
			analysisQuery.GET("/daily", s.handleAnalysisQuery)
			analysisQuery.GET("/strategies", s.handleListStrategies)
			analysisQuery.GET("/strategies/get", func(c *gin.Context) { s.handleStrategyGetOrUpdate(c, c.Query("id")) })
			analysisQuery.POST("/strategies/save-scoring", s.handleCreateStrategy)
			analysisQuery.GET("/history", s.handleAnalysisHistory)
			analysisQuery.GET("/summary", s.handleAnalysisSummary)
			analysisQuery.POST("/backtest", s.handleAnalysisBacktest)
			analysisQuery.POST("/backtest/slug", s.handleSlugBacktest)
			analysisQuery.GET("/backtest/preset", s.handleGetBacktestPreset)
			analysisQuery.POST("/backtest/preset", s.handleSaveBacktestPreset)
		}
	}

	// 前端操作介面
	s.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "api not found"})
			return
		}
		http.FileServer(http.Dir("web")).ServeHTTP(c.Writer, c.Request)
	})
}


type telegramNotifierAdapter struct {
	client *notify.TelegramClient
}

func (a *telegramNotifierAdapter) Notify(msg string) error {
	return a.client.SendMessage(context.Background(), msg)
}
