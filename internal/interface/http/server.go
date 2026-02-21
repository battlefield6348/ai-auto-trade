package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/auth"
	appStrategy "ai-auto-trade/internal/application/strategy"
	"ai-auto-trade/internal/application/trading"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	"ai-auto-trade/internal/domain/strategy"
	tradingDomain "ai-auto-trade/internal/domain/trading"
	"ai-auto-trade/internal/infra/memory"

	"github.com/gin-gonic/gin"
)

const (
	errCodeBadRequest         = "BAD_REQUEST"
	errCodeInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	errCodeUnauthorized       = "AUTH_UNAUTHORIZED"
	errCodeForbidden          = "AUTH_FORBIDDEN"
	errCodeAnalysisNotReady   = "ANALYSIS_NOT_READY"
	errCodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
	errCodeNotFound           = "NOT_FOUND"
	errCodeInternal           = "INTERNAL_ERROR"
	refreshCookieName         = "refresh_token"
)

var errNoPrices = errors.New("ingestion data not ready")

// DataRepository 定義 ingestion/analysis 讀寫與查詢接口。
type DataRepository interface {
	analysis.AnalysisQueryRepository
	UpsertTradingPair(ctx context.Context, pair, name string, market dataDomain.Market, industry string) (string, error)
	InsertDailyPrice(ctx context.Context, stockID string, price dataDomain.DailyPrice) error
	PricesByDate(ctx context.Context, date time.Time) ([]dataDomain.DailyPrice, error)
	PricesByPair(ctx context.Context, pair string, timeframe string) ([]dataDomain.DailyPrice, error)
	FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error)
	Get(ctx context.Context, symbol string, date time.Time, timeframe string) (analysisDomain.DailyAnalysisResult, error)
	InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error
	HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error)
	LatestAnalysisDate(ctx context.Context) (time.Time, error)
}

// memoryRepoAdapter 讓 memory.Store 相容 DataRepository。
type memoryRepoAdapter struct {
	store *memory.Store
}

func (m memoryRepoAdapter) UpsertTradingPair(ctx context.Context, pair, name string, market dataDomain.Market, industry string) (string, error) {
	return m.store.UpsertTradingPair(pair, name, market, industry), nil
}

func (m memoryRepoAdapter) InsertDailyPrice(ctx context.Context, stockID string, price dataDomain.DailyPrice) error {
	m.store.InsertDailyPrice(price)
	return nil
}

func (m memoryRepoAdapter) PricesByDate(ctx context.Context, date time.Time) ([]dataDomain.DailyPrice, error) {
	return m.store.PricesByDate(date), nil
}

func (m memoryRepoAdapter) PricesByPair(ctx context.Context, pair string, timeframe string) ([]dataDomain.DailyPrice, error) {
	// memory store doesn't support timeframe yet, so we ignore it for now
	return m.store.PricesByPair(pair), nil
}

func (m memoryRepoAdapter) InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error {
	m.store.InsertAnalysisResult(res)
	return nil
}

func (m memoryRepoAdapter) HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error) {
	return m.store.HasAnalysisForDate(date), nil
}

func (m memoryRepoAdapter) LatestAnalysisDate(ctx context.Context) (time.Time, error) {
	d, ok := m.store.LatestAnalysisDate()
	if !ok {
		return time.Time{}, fmt.Errorf("no analysis data")
	}
	return d, nil
}

// Analysis query implementations
func (m memoryRepoAdapter) FindByDate(ctx context.Context, date time.Time, filter analysis.QueryFilter, sortOpt analysis.SortOption, pagination analysis.Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	return m.store.FindByDate(ctx, date, filter, sortOpt, pagination)
}

func (m memoryRepoAdapter) FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error) {
	// memory store doesn't support timeframe yet, so we ignore it for now or return daily
	return m.store.FindHistory(ctx, symbol, from, to, limit, onlySuccess)
}

func (m memoryRepoAdapter) Get(ctx context.Context, symbol string, date time.Time, timeframe string) (analysisDomain.DailyAnalysisResult, error) {
	return m.store.Get(ctx, symbol, date)
}

type analysisRunSummary struct {
	total   int
	success int
	failure int
}

type backfillFailure struct {
	TradeDate string `json:"trade_date"`
	Stage     string `json:"stage"`
	Reason    string `json:"reason"`
}

type jobRun struct {
	Kind          string
	TriggeredBy   string
	Start         time.Time
	End           time.Time
	IngestionOK   bool
	IngestionErr  string
	AnalysisOn    bool
	AnalysisOK    bool
	AnalysisTotal int
	AnalysisSucc  int
	AnalysisFail  int
	AnalysisErr   string
	Failures      []backfillFailure
	DataSource    string
}

type analysisBacktestRequest struct {
	Symbol    string             `json:"symbol"`
	StartDate string             `json:"start_date"`
	EndDate   string             `json:"end_date"`
	Entry     backtestSideParams `json:"entry"`
	Exit      backtestSideParams `json:"exit"`
	Horizons  []int              `json:"horizons"`
	Timeframe string             `json:"timeframe"`
}

type backtestSideParams struct {
	Weights    backtestWeights    `json:"weights"`
	Thresholds backtestThresholds `json:"thresholds"`
	Flags      backtestFlags      `json:"flags"`
	TotalMin   float64            `json:"total_min"`
}

type backtestWeights struct {
	Score       float64 `json:"score"`
	ChangeBonus float64 `json:"change_bonus"`
	VolumeBonus float64 `json:"volume_bonus"`
	ReturnBonus float64 `json:"return_bonus"`
	MaBonus     float64 `json:"ma_bonus"`
	AmpBonus    float64 `json:"amp_bonus"`
	RangeBonus  float64 `json:"range_bonus"`
}

// backtestSides removed

type backtestThresholds struct {
	ChangeMin      float64 `json:"change_min"`
	VolumeRatioMin float64 `json:"volume_ratio_min"`
	Return5Min     float64 `json:"return5_min"`
	MaGapMin       float64 `json:"ma_gap_min"`
	AmpMin         float64 `json:"amp_min"`
	RangeMin       float64 `json:"range_min"`
}

type backtestFlags struct {
	UseChange bool `json:"use_change"`
	UseVolume bool `json:"use_volume"`
	UseReturn bool `json:"use_return"`
	UseMa     bool `json:"use_ma"`
	UseAmp    bool `json:"use_amp"`
	UseRange  bool `json:"use_range"`
}

type analysisBacktestEvent struct {
	TradingPair    string             `json:"trading_pair"`
	TradeDate      string             `json:"trade_date"`
	ClosePrice     float64            `json:"close_price"`
	ChangePercent  float64            `json:"change_percent"`
	Return5d       *float64           `json:"return_5d,omitempty"`
	MaGap          *float64           `json:"ma_gap,omitempty"`
	VolumeRatio    *float64           `json:"volume_ratio,omitempty"`
	Score          float64            `json:"score"`
	TotalScore     float64            `json:"total_score"`
	EntryScore     float64            `json:"entry_score"`
	ExitScore      float64            `json:"exit_score"`
	IsTriggered    bool               `json:"is_triggered"`
	Components     map[string]float64 `json:"components,omitempty"`
	ExitComponents map[string]float64 `json:"exit_components,omitempty"`
	ForwardReturns map[string]float64 `json:"forward_returns,omitempty"`
}

type backtestReturnStat struct {
	AvgReturn float64 `json:"avg_return"`
	WinRate   float64 `json:"win_rate"`
}

type parsedBacktestInput struct {
	req       analysisBacktestRequest
	symbol    string
	startDate time.Time
	endDate   time.Time
	horizons  []int
	timeframe string
}

// --- Helpers ---

// --- Helpers ---

func (s *Server) getSymbol(c *gin.Context) string {
	return strings.ToUpper(strings.TrimSpace(c.DefaultQuery("symbol", "BTCUSDT")))
}

func (s *Server) getTimeframe(c *gin.Context, def string) string {
	return c.DefaultQuery("timeframe", def)
}

func (s *Server) parseDateRange(c *gin.Context) (time.Time, time.Time, error) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	if startStr == "" {
		// Default to last 30 days
		end := time.Now()
		start := end.AddDate(0, 0, -30)
		return start, end, nil
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_date")
	}

	var end time.Time
	if endStr == "" {
		end = time.Now()
	} else {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end_date")
		}
	}

	return start, end, nil
}

// --- Handlers ---

func (s *Server) handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "pong",
		"timestamp": time.Now().Unix(),
		"status":    "alive",
	})
}

func (s *Server) handleHealth(c *gin.Context) {
	dbStatus := "unavailable"
	if s.db == nil {
		dbStatus = "not_configured"
	} else {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if s.db != nil {
			if err := s.db.PingContext(ctx); err == nil {
				dbStatus = "ok"
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"db":             dbStatus,
		"use_synthetic":  s.useSynthetic,
		"active_env":     s.defaultEnv,
		"use_testnet":    s.defaultEnv == tradingDomain.EnvTest,
		"analysis_ready": s.tokenSvc != nil,
	})
}

func (s *Server) handleLogin(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	res, err := s.loginUC.Execute(c.Request.Context(), auth.LoginInput{
		Email:     body.Email,
		Password:  body.Password,
		UserAgent: c.Request.UserAgent(),
		IP:        c.ClientIP(),
	})
	if err != nil {
		log.Printf("login failed email=%s: %v", body.Email, err)
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid credentials", "error_code": errCodeInvalidCredentials})
		return
	}
	log.Printf("login success user_id=%s role=%s email=%s", res.User.ID, res.User.Role, res.User.Email)

	s.setRefreshCookie(c, res.Token.RefreshToken, res.Token.RefreshExpiry)
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"access_token":       res.Token.AccessToken,
		"token_type":         "Bearer",
		"expires_in":         int(s.tokenTTL.Seconds()),
		"refresh_expires_in": int(s.refreshTTL.Seconds()),
	})
}

// handleRegister removed (single-user mode)

func (s *Server) handleRefresh(c *gin.Context) {
	cookie, err := c.Cookie(refreshCookieName)
	if err != nil || cookie == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "missing refresh token", "error_code": errCodeUnauthorized})
		return
	}
	pair, err := s.tokenSvc.Refresh(c.Request.Context(), cookie)
	if err != nil {
		log.Printf("refresh token failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "refresh token expired or invalid", "error_code": errCodeUnauthorized})
		return
	}
	s.setRefreshCookie(c, pair.RefreshToken, pair.RefreshExpiry)
	c.JSON(http.StatusOK, gin.H{
		"success":            true,
		"access_token":       pair.AccessToken,
		"token_type":         "Bearer",
		"expires_in":         int(time.Until(pair.AccessExpiry).Seconds()),
		"refresh_expires_in": int(time.Until(pair.RefreshExpiry).Seconds()),
	})
}

func (s *Server) handleLogout(c *gin.Context) {
	cookie, err := c.Cookie(refreshCookieName)
	if err == nil && cookie != "" {
		if s.logoutUC != nil {
			if revokeErr := s.logoutUC.Execute(c.Request.Context(), cookie); revokeErr != nil {
				log.Printf("logout revoke refresh failed: %v", revokeErr)
			}
		}
	}
	s.setRefreshCookie(c, "", time.Now().Add(-time.Hour))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "logged out",
	})
}

func (s *Server) handleIngestionBackfill(c *gin.Context) {
	var body struct {
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		RunAnalysis *bool  `json:"run_analysis"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	if body.StartDate == "" || body.EndDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "start_date and end_date required", "error_code": errCodeBadRequest})
		return
	}
	startDate, err := time.Parse("2006-01-02", body.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid start_date", "error_code": errCodeBadRequest})
		return
	}
	endDate, err := time.Parse("2006-01-02", body.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid end_date", "error_code": errCodeBadRequest})
		return
	}
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "end_date must be after start_date", "error_code": errCodeBadRequest})
		return
	}
	runAnalysis := true
	if body.RunAnalysis != nil {
		runAnalysis = *body.RunAnalysis
	}

	totalDays := 0
	ingestionSuccessDays := 0
	analysisSuccessDays := 0
	failures := make([]backfillFailure, 0)
	start := time.Now()
	log.Printf("ingestion backfill start start_date=%s end_date=%s run_analysis=%t", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), runAnalysis)

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		totalDays++
		var ingestErr error
		if s.useSynthetic {
			ingestErr = s.generateDailyPrices(c.Request.Context(), d)
		} else {
			ingestErr = s.generateDailyPricesStrict(c.Request.Context(), d)
		}
		if ingestErr != nil {
			failures = append(failures, backfillFailure{
				TradeDate: d.Format("2006-01-02"),
				Stage:     "ingestion",
				Reason:    ingestErr.Error(),
			})
			continue
		}
		ingestionSuccessDays++
		if runAnalysis {
			if _, err := s.runAnalysisForDate(c.Request.Context(), d); err != nil {
				failures = append(failures, backfillFailure{
					TradeDate: d.Format("2006-01-02"),
					Stage:     "analysis",
					Reason:    err.Error(),
				})
				continue
			}
			analysisSuccessDays++
		}
	}

	log.Printf("ingestion backfill done days=%d ingestion_success=%d analysis_success=%d failures=%d duration=%s", totalDays, ingestionSuccessDays, analysisSuccessDays, len(failures), time.Since(start))
	s.recordJob(jobRun{
		Kind:          "backfill",
		TriggeredBy:   currentUserID(c),
		Start:         start,
		End:           time.Now(),
		IngestionOK:   len(failures) == 0,
		AnalysisOn:    runAnalysis,
		AnalysisOK:    runAnalysis && len(failures) == 0,
		AnalysisTotal: totalDays,
		AnalysisSucc:  analysisSuccessDays,
		AnalysisFail:  totalDays - analysisSuccessDays,
		Failures:      failures,
	})
	c.JSON(http.StatusOK, gin.H{
		"success":                true,
		"start_date":             startDate.Format("2006-01-02"),
		"end_date":               endDate.Format("2006-01-02"),
		"total_days":             totalDays,
		"ingestion_success_days": ingestionSuccessDays,
		"analysis_success_days":  analysisSuccessDays,
		"failure_days":           len(failures),
		"analysis_enabled":       runAnalysis,
		"failures":               failures,
	})
}

func (s *Server) handleIngestionDaily(c *gin.Context) {
	var body struct {
		TradeDate   string `json:"trade_date"`
		RunAnalysis *bool  `json:"run_analysis"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	if body.TradeDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "trade_date required", "error_code": errCodeBadRequest})
		return
	}
	tradeDate, err := time.Parse("2006-01-02", body.TradeDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid trade_date", "error_code": errCodeBadRequest})
		return
	}
	runAnalysis := true
	if body.RunAnalysis != nil {
		runAnalysis = *body.RunAnalysis
	}

	start := time.Now()
	var ingestionErr error
	if s.useSynthetic {
		ingestionErr = s.generateDailyPrices(c.Request.Context(), tradeDate)
	} else {
		ingestionErr = s.generateDailyPricesStrict(c.Request.Context(), tradeDate)
	}

	var stats analysisRunSummary
	var analysisErr error
	if ingestionErr == nil && runAnalysis {
		stats, analysisErr = s.runAnalysisForDate(c.Request.Context(), tradeDate)
	}

	s.recordJob(jobRun{
		Kind:          "ingestion_daily",
		TriggeredBy:   currentUserID(c),
		Start:         start,
		End:           time.Now(),
		IngestionOK:   ingestionErr == nil,
		IngestionErr:  errorText(ingestionErr),
		AnalysisOn:    runAnalysis,
		AnalysisOK:    runAnalysis && analysisErr == nil,
		AnalysisTotal: stats.total,
		AnalysisSucc:  stats.success,
		AnalysisFail:  stats.failure,
		AnalysisErr:   errorText(analysisErr),
	})

	c.JSON(http.StatusOK, gin.H{
		"success":          ingestionErr == nil && (!runAnalysis || analysisErr == nil),
		"trade_date":       tradeDate.Format("2006-01-02"),
		"duration_seconds": int(time.Since(start).Seconds()),
		"ingestion": map[string]interface{}{
			"success": ingestionErr == nil,
			"error":   errorString(ingestionErr),
		},
		"analysis": map[string]interface{}{
			"enabled":       runAnalysis,
			"success":       analysisErr == nil && runAnalysis,
			"total":         stats.total,
			"success_count": stats.success,
			"failure_count": stats.failure,
			"error":         errorString(analysisErr),
		},
	})
}

func (s *Server) runAnalysisForDate(ctx context.Context, tradeDate time.Time) (analysisRunSummary, error) {
	stats := analysisRunSummary{}
	prices, err := s.dataRepo.PricesByDate(ctx, tradeDate)
	if err != nil {
		return stats, err
	}
	if len(prices) == 0 {
		return stats, errNoPrices
	}
	stats.total = len(prices)
	for _, p := range prices {
		stockID, err := s.dataRepo.UpsertTradingPair(ctx, p.Symbol, p.Symbol, p.Market, "")
		if err != nil {
			log.Printf("upsert stock failed symbol=%s: %v", p.Symbol, err)
			continue
		}
		res := s.calculateAnalysis(ctx, p)
		if err := s.dataRepo.InsertAnalysisResult(ctx, stockID, res); err != nil {
			log.Printf("write analysis failed symbol=%s date=%s: %v", p.Symbol, tradeDate.Format("2006-01-02"), err)
			continue
		}
		stats.success++
	}
	stats.failure = stats.total - stats.success
	return stats, nil
}

func (s *Server) handleAnalysisDaily(c *gin.Context) {
	var body struct {
		TradeDate string `json:"trade_date"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	if body.TradeDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "trade_date required", "error_code": errCodeBadRequest})
		return
	}
	tradeDate, err := time.Parse("2006-01-02", body.TradeDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid trade_date", "error_code": errCodeBadRequest})
		return
	}

	start := time.Now()
	stats, runErr := s.runAnalysisForDate(c.Request.Context(), tradeDate)
	if errors.Is(runErr, errNoPrices) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "ingestion data not ready for trade_date", "error_code": errCodeAnalysisNotReady})
		return
	}
	if runErr != nil {
		log.Printf("analysis daily failed date=%s: %v", tradeDate.Format("2006-01-02"), runErr)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "analysis failed", "error_code": errCodeInternal})
		return
	}

	s.recordJob(jobRun{
		Kind:          "analysis_daily",
		TriggeredBy:   currentUserID(c),
		Start:         start,
		End:           time.Now(),
		IngestionOK:   true,
		AnalysisOn:    true,
		AnalysisOK:    runErr == nil,
		AnalysisTotal: stats.total,
		AnalysisSucc:  stats.success,
		AnalysisFail:  stats.failure,
		AnalysisErr:   errorText(runErr),
	})

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"trade_date":       tradeDate.Format("2006-01-02"),
		"total":            stats.total,
		"success_count":    stats.success,
		"failure_count":    stats.failure,
		"duration_seconds": int(time.Since(start).Seconds()),
	})
}

func (s *Server) handleAnalysisQuery(c *gin.Context) {
	tradeDateStr := c.Query("trade_date")
	if tradeDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "trade_date required", "error_code": errCodeBadRequest})
		return
	}
	tradeDate, err := time.Parse("2006-01-02", tradeDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid trade_date", "error_code": errCodeBadRequest})
		return
	}
	timeframe := s.getTimeframe(c, "1d")
	limit := parseIntDefault(c.Query("limit"), 100)
	if limit > 1000 {
		limit = 1000
	}
	offset := parseIntDefault(c.Query("offset"), 0)

	out, err := s.queryUC.QueryByDate(c.Request.Context(), analysis.QueryByDateInput{
		Date: tradeDate,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
			Timeframe:   timeframe,
		},
		Pagination: analysis.Pagination{
			Offset: offset,
			Limit:  limit,
		},
	})
	if err != nil {
		log.Printf("analysis query failed date=%s: %v", tradeDate.Format("2006-01-02"), err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal error", "error_code": errCodeInternal})
		return
	}
	if out.Total == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "analysis results not ready for trade_date", "error_code": errCodeAnalysisNotReady})
		return
	}

	type item struct {
		TradingPair   string   `json:"trading_pair"`
		MarketType    string   `json:"market_type"`
		ClosePrice    float64  `json:"close_price"`
		ChangePercent float64  `json:"change_percent"`
		Return5d      *float64 `json:"return_5d,omitempty"`
		Volume        int64    `json:"volume"`
		VolumeRatio   *float64 `json:"volume_ratio,omitempty"`
		Score         float64  `json:"score"`
	}
	items := make([]item, 0, len(out.Results))
	for _, r := range out.Results {
		items = append(items, item{
			TradingPair:   r.Symbol,
			MarketType:    string(r.Market),
			ClosePrice:    r.Close,
			ChangePercent: r.ChangeRate,
			Return5d:      r.Return5,
			Volume:        int64(r.Volume),
			VolumeRatio:   r.VolumeMultiple,
			Score:         r.Score,
		})
	}

	log.Printf("analysis query done date=%s total=%d limit=%d offset=%d", tradeDate.Format("2006-01-02"), out.Total, limit, offset)
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"trade_date":  tradeDate.Format("2006-01-02"),
		"total_count": out.Total,
		"items":       items,
	})
}

func (s *Server) handleAnalysisHistory(c *gin.Context) {
	symbol := s.getSymbol(c)
	startDate, endDate, err := s.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}

	limit := parseIntDefault(c.Query("limit"), 1000)
	onlySuccess := parseBoolDefault(c.Query("only_success"), true)
	timeframe := s.getTimeframe(c, "1d")

	out, err := s.queryUC.QueryHistory(c.Request.Context(), analysis.QueryHistoryInput{
		Symbol:      symbol,
		Timeframe:   timeframe,
		From:        &startDate,
		To:          &endDate,
		Limit:       limit,
		OnlySuccess: onlySuccess,
	})
	if err != nil {
		log.Printf("analysis history failed symbol=%s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal error", "error_code": errCodeInternal})
		return
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].TradeDate.Before(out[j].TradeDate)
	})

	type item struct {
		TradingPair   string   `json:"trading_pair"`
		MarketType    string   `json:"market_type"`
		TradeDate     string   `json:"trade_date"`
		ClosePrice    float64  `json:"close_price"`
		ChangePercent float64  `json:"change_percent"`
		Return5d      *float64 `json:"return_5d,omitempty"`
		Volume        int64    `json:"volume"`
		VolumeRatio   *float64 `json:"volume_ratio,omitempty"`
		Score         float64  `json:"score"`
		Success       bool     `json:"success"`
	}
	items := make([]item, 0, len(out))
	for _, r := range out {
		items = append(items, item{
			TradingPair:   r.Symbol,
			MarketType:    string(r.Market),
			TradeDate:     r.TradeDate.Format("2006-01-02"),
			ClosePrice:    r.Close,
			ChangePercent: r.ChangeRate,
			Return5d:      r.Return5,
			Volume:        int64(r.Volume),
			VolumeRatio:   r.VolumeMultiple,
			Score:         r.Score,
			Success:       r.Success,
		})
	}

	respStart := ""
	respEnd := ""
	if !startDate.IsZero() {
		respStart = startDate.Format("2006-01-02")
	}
	if !endDate.IsZero() {
		respEnd = endDate.Format("2006-01-02")
	}
	if respStart == "" && len(out) > 0 {
		respStart = out[0].TradeDate.Format("2006-01-02")
	}
	if respEnd == "" && len(out) > 0 {
		respEnd = out[len(out)-1].TradeDate.Format("2006-01-02")
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"symbol":      symbol,
		"start_date":  respStart,
		"end_date":    respEnd,
		"total_count": len(out),
		"items":       items,
	})
}

func (s *Server) handleAnalysisSummary(c *gin.Context) {
	latestDate, err := s.dataRepo.LatestAnalysisDate(c.Request.Context())
	if err != nil || latestDate.IsZero() {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "analysis results not ready", "error_code": errCodeAnalysisNotReady})
		return
	}
	out, err := s.queryUC.QueryByDate(c.Request.Context(), analysis.QueryByDateInput{
		Date: latestDate,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
		},
		Pagination: analysis.Pagination{
			Offset: 0,
			Limit:  100,
		},
	})
	if err != nil || len(out.Results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "analysis results not ready", "error_code": errCodeAnalysisNotReady})
		return
	}

	// 以 5 日收益率最高的交易對作為當前趨勢參考 (最高收益資料)
	best := out.Results[0]
	for _, r := range out.Results {
		if r.Return5 != nil && best.Return5 != nil {
			if *r.Return5 > *best.Return5 {
				best = r
			}
		} else if r.Score > best.Score {
			best = r
		}
	}

	trend := "neutral"
	advice := "保持觀察，等待更明確的趨勢。"
	if best.Score >= 80 {
		trend = "bullish"
		advice = "偏多：可分批佈局或續抱，留意風險控管。"
	} else if best.Score <= 40 {
		trend = "bearish"
		advice = "偏空：宜觀望或減碼，避免追價。"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"trade_date":   latestDate.Format("2006-01-02"),
		"trading_pair": best.Symbol,
		"trend":        trend,
		"advice":       advice,
		"metrics": map[string]interface{}{
			"close_price":    best.Close,
			"change_percent": best.ChangeRate,
			"return_5d":      best.Return5,
			"volume_ratio":   best.VolumeMultiple,
			"score":          best.Score,
		},
	})
}

func (s *Server) handleAnalysisBacktest(c *gin.Context) {
	var body analysisBacktestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	input, err := normalizeBacktestRequest(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}

	history, err := s.dataRepo.FindHistory(c.Request.Context(), input.symbol, input.timeframe, &input.startDate, &input.endDate, 5000, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "query history failed", "error_code": errCodeInternal})
		return
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].TradeDate.Before(history[j].TradeDate)
	})

	events := make([]analysisBacktestEvent, 0, len(history))
	retStats := make(map[int][]float64)

	// Simulation state
	type backtestTrade struct {
		EntryDate  string  `json:"entry_date"`
		EntryPrice float64 `json:"entry_price"`
		ExitDate   string  `json:"exit_date"`
		ExitPrice  float64 `json:"exit_price"`
		PnL        float64 `json:"pnl"`
		PnLPct     float64 `json:"pnl_pct"`
		Reason     string  `json:"reason"`
	}
	var trades []backtestTrade
	var currentPosition *backtestTrade
	totalReturn := 1.0

	for idx, res := range history {
		entryTotal, entryComps := calcBacktestScore(res, input.req.Entry)
		exitTotal, exitComps := calcBacktestScore(res, input.req.Exit)
		
		triggered := entryTotal >= input.req.Entry.TotalMin

		// Calculate forward returns for statistics (only for triggered events to maintain density)
		var forward map[string]float64
		if triggered {
			forward = calcForwardReturns(history, idx, input.horizons)
			for _, h := range input.horizons {
				key := fmt.Sprintf("d%d", h)
				if val, ok := forward[key]; ok {
					retStats[h] = append(retStats[h], val)
				}
			}
		}

		ev := analysisBacktestEvent{
			TradingPair:    res.Symbol,
			TradeDate:      res.TradeDate.Format("2006-01-02"),
			ClosePrice:     res.Close,
			ChangePercent:  res.ChangeRate,
			Return5d:       res.Return5,
			MaGap:          res.Deviation20,
			VolumeRatio:    res.VolumeMultiple,
			Score:          res.Score,
			TotalScore:     entryTotal,
			EntryScore:     entryTotal,
			ExitScore:      exitTotal,
			IsTriggered:    triggered,
			Components:     entryComps,
			ExitComponents: exitComps,
		}
		if len(forward) > 0 {
			ev.ForwardReturns = forward
		}
		events = append(events, ev)

		// Simulation Logic (Unchanged but ensuring consistency)
		if currentPosition == nil {
			if triggered {
				currentPosition = &backtestTrade{
					EntryDate:  res.TradeDate.Format("2006-01-02"),
					EntryPrice: res.Close,
				}
			}
		} else {
			// Check Exit
			exitTriggered := exitTotal < input.req.Exit.TotalMin
			
			// Also auto-exit if entry score drops below 50% of threshold (builtin protection)
			if !exitTriggered {
				if entryTotal < (input.req.Entry.TotalMin * 0.5) {
					exitTriggered = true
					currentPosition.Reason = fmt.Sprintf("AI信號轉弱 (%.1f < %.1f)", entryTotal, input.req.Entry.TotalMin*0.5)
				}
			}

			// User requirement: Exit if ExitScore < ExitThreshold
			if exitTriggered {
				currentPosition.ExitDate = res.TradeDate.Format("2006-01-02")
				currentPosition.ExitPrice = res.Close
				
				// Apply 0.1% slippage/fee on exit
				exitPriceWithFee := currentPosition.ExitPrice * 0.999
				
				currentPosition.PnL = exitPriceWithFee - currentPosition.EntryPrice
				currentPosition.PnLPct = (exitPriceWithFee / currentPosition.EntryPrice) - 1.0
				if currentPosition.Reason == "" { // Only set if not already set by auto-exit
					currentPosition.Reason = fmt.Sprintf("AI信號轉弱 (%.1f < %.1f)", exitTotal, input.req.Exit.TotalMin)
				}
				
				trades = append(trades, *currentPosition)
				totalReturn *= (1.0 + currentPosition.PnLPct)
				currentPosition = nil
			}
		}
	}
	
	// Force close last position if still open at end of data (apply fee)
	if currentPosition != nil && len(history) > 0 {
		last := history[len(history)-1]
		currentPosition.ExitDate = last.TradeDate.Format("2006-01-02")
		currentPosition.ExitPrice = last.Close * 0.999 // Fee
		currentPosition.PnL = currentPosition.ExitPrice - currentPosition.EntryPrice
		currentPosition.PnLPct = (currentPosition.ExitPrice / currentPosition.EntryPrice) - 1.0
		currentPosition.Reason = "回測結束前尚未出場 (Simulation End)"
		
		trades = append(trades, *currentPosition)
		totalReturn *= (1.0 + currentPosition.PnLPct)
		currentPosition = nil
	}

	summary := map[string]interface{}{
		"total_trades": len(trades),
		"total_return": (totalReturn - 1.0) * 100,
	}
	if len(trades) > 0 {
		wins := 0
		for _, t := range trades {
			if t.PnLPct > 0 {
				wins++
			}
		}
		summary["win_rate"] = float64(wins) / float64(len(trades)) * 100
	}

	stats := make(map[string]backtestReturnStat)
	for h, vals := range retStats {
		if len(vals) == 0 {
			continue
		}
		sum := 0.0
		wins := 0
		for _, v := range vals {
			sum += v
			if v > 0 {
				wins++
			}
		}
		stats[fmt.Sprintf("d%d", h)] = backtestReturnStat{
			AvgReturn: sum / float64(len(vals)),
			WinRate:   float64(wins) / float64(len(vals)),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"symbol":       input.symbol,
		"start_date":   input.startDate.Format("2006-01-02"),
		"end_date":     input.endDate.Format("2006-01-02"),
		"total_events": len(events),
		"config":       input.req,
		"events":       events,
		"trades":       trades,
		"summary":      summary,
		"stats": map[string]interface{}{
			"returns": stats,
		},
	})
}

type slugBacktestRequest struct {
	Slug      string `json:"slug"`
	Symbol    string `json:"symbol"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (s *Server) handleSlugBacktest(c *gin.Context) {
	var body slugBacktestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	start, err := time.Parse("2006-01-02", body.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid start_date", "error_code": errCodeBadRequest})
		return
	}
	end, err := time.Parse("2006-01-02", body.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid end_date", "error_code": errCodeBadRequest})
		return
	}
	symbol := strings.ToUpper(body.Symbol)
	if symbol == "" {
		symbol = "BTCUSDT"
	}

	if s.scoringBtUC == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "database storage not available", "error_code": errCodeNotFound})
		return
	}
	res, err := s.scoringBtUC.Execute(c.Request.Context(), body.Slug, symbol, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  res,
	})
}

func (s *Server) handleStrategyExecute(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		// Fallback for non-param route if needed
		slug = strings.TrimPrefix(c.Request.URL.Path, "/api/admin/strategies/execute/")
	}
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "slug is required", "error_code": errCodeBadRequest})
		return
	}

	// 預設執行環境為 Test (Binance Testnet)
	env := tradingDomain.EnvTest
	if c.Query("env") == "prod" {
		env = tradingDomain.EnvProd
	}

	userID := currentUserID(c)
	if userID == "" {
		userID = "00000000-0000-0000-0000-000000000001" // Fallback admin
	}

	err := s.tradingSvc.ExecuteScoringAutoTrade(c.Request.Context(), slug, env, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Strategy check executed",
	})
}

func (s *Server) handleListScoringStrategies(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"strategies": []interface{}{},
		})
		return
	}
	userID := currentUserID(c)
	// 查詢使用者自己的策略，或者是系統預設策略 (由 admin@example.com 擁有)
	rows, err := s.db.QueryContext(c.Request.Context(), `
		SELECT id, name, slug, threshold, env, is_active, updated_at 
		FROM strategies 
		WHERE slug IS NOT NULL 
		AND (user_id = $1 OR user_id = (SELECT id FROM users WHERE email = 'admin@example.com' LIMIT 1))
		ORDER BY updated_at DESC
	`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to query strategies", "error_code": errCodeInternal})
		return
	}
	defer rows.Close()

	type item struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Slug      string    `json:"slug"`
		Threshold float64   `json:"threshold"`
		Env       string    `json:"env"`
		IsActive  bool      `json:"active"` // Map to 'active' for frontend compatibility
		UpdatedAt time.Time `json:"updated_at"`
	}
	var list []item
	for rows.Next() {
		var i item
		if err := rows.Scan(&i.ID, &i.Name, &i.Slug, &i.Threshold, &i.Env, &i.IsActive, &i.UpdatedAt); err != nil {
			log.Printf("scan strategy error: %v", err)
			continue
		}
		list = append(list, i)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"strategies": list,
	})
}

func (s *Server) handleSaveScoringStrategy(c *gin.Context) {
	var body appStrategy.SaveScoringStrategyInput
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	if body.Slug == "" || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "name and slug are required", "error_code": errCodeBadRequest})
		return
	}

	body.UserID = currentUserID(c)
	if s.saveScoringBtUC == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "database storage not available", "error_code": errCodeNotFound})
		return
	}
	if err := s.saveScoringBtUC.Execute(c.Request.Context(), body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

func (s *Server) handleGetScoringStrategy(c *gin.Context) {
	slug := c.Query("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "slug is required", "error_code": errCodeBadRequest})
		return
	}

	if s.db == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "database not available", "error_code": errCodeNotFound})
		return
	}
	strat, err := strategy.LoadScoringStrategyBySlug(c.Request.Context(), s.db, slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"strategy": strat,
	})
}


func (s *Server) handleGetBacktestPreset(c *gin.Context) {
	if s.presetStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "preset store not ready", "error_code": errCodeInternal})
		return
	}
	userID := currentUserID(c)
	raw, err := s.presetStore.Load(c.Request.Context(), userID)
	if err != nil {
		if s.presetStore.NotFound(err) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "尚無預設",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "load preset failed", "error_code": errCodeInternal})
		return
	}
	var preset analysisBacktestRequest
	if err := json.Unmarshal(raw, &preset); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "invalid preset data", "error_code": errCodeInternal})
		return
	}
	if len(preset.Horizons) == 0 {
		preset.Horizons = []int{3, 5, 10}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"preset":  preset,
	})
}

func (s *Server) handleSaveBacktestPreset(c *gin.Context) {
	if s.presetStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "preset store not ready", "error_code": errCodeInternal})
		return
	}
	var body analysisBacktestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	input, err := normalizeBacktestRequest(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}
	payload, err := json.Marshal(input.req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "encode preset failed", "error_code": errCodeInternal})
		return
	}
	if err := s.presetStore.Save(c.Request.Context(), currentUserID(c), payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "save preset failed", "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// handleStrongStocks removed.

func normalizeBacktestRequest(req analysisBacktestRequest) (parsedBacktestInput, error) {
	var out parsedBacktestInput

	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		symbol = "BTCUSDT"
	}

	start, err := time.Parse("2006-01-02", strings.TrimSpace(req.StartDate))
	if err != nil {
		return out, fmt.Errorf("invalid start_date")
	}
	end, err := time.Parse("2006-01-02", strings.TrimSpace(req.EndDate))
	if err != nil {
		return out, fmt.Errorf("invalid end_date")
	}
	if end.Before(start) {
		return out, fmt.Errorf("end_date must be after start_date")
	}

	// Thresholds are already normalized to ratios by the frontend.
	horizons := normalizeHorizons(req.Horizons)
	req.Symbol = symbol
	req.Horizons = horizons
	tf := req.Timeframe
	if tf == "" {
		tf = "1d"
	}

	out = parsedBacktestInput{
		req:       req,
		symbol:    symbol,
		startDate: start,
		endDate:   end,
		horizons:  horizons,
		timeframe: tf,
	}
	return out, nil
}

func normalizeHorizons(values []int) []int {
	defaults := []int{3, 5, 10}
	seen := make(map[int]bool)
	out := make([]int, 0, len(values))
	for _, v := range values {
		if v <= 0 || v > 365 {
			continue
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	if len(out) == 0 {
		return defaults
	}
	sort.Ints(out)
	return out
}

func calcBacktestScore(res analysisDomain.DailyAnalysisResult, params backtestSideParams) (float64, map[string]float64) {
	total := 0.0
	totalWeight := 0.0
	components := make(map[string]float64)

	// AI Core Score
	totalWeight += params.Weights.Score
	total += params.Weights.Score * (res.Score / 100.0)
	if params.Weights.Score != 0 {
		components["score"] = params.Weights.Score * (res.Score / 100.0)
	}

	// Change Bonus
	if params.Flags.UseChange {
		totalWeight += params.Weights.ChangeBonus
		if res.ChangeRate >= params.Thresholds.ChangeMin {
			total += params.Weights.ChangeBonus
			if params.Weights.ChangeBonus != 0 {
				components["change"] = params.Weights.ChangeBonus
			}
		}
	}

	// Volume Bonus
	if params.Flags.UseVolume {
		totalWeight += params.Weights.VolumeBonus
		vol := 0.0
		if res.VolumeMultiple != nil {
			vol = *res.VolumeMultiple
		}
		if vol >= params.Thresholds.VolumeRatioMin {
			total += params.Weights.VolumeBonus
			if params.Weights.VolumeBonus != 0 {
				components["volume"] = params.Weights.VolumeBonus
			}
		}
	}

	// Return Bonus
	if params.Flags.UseReturn {
		totalWeight += params.Weights.ReturnBonus
		ret := 0.0
		if res.Return5 != nil {
			ret = *res.Return5
		}
		if ret >= params.Thresholds.Return5Min {
			total += params.Weights.ReturnBonus
			if params.Weights.ReturnBonus != 0 {
				components["return"] = params.Weights.ReturnBonus
			}
		}
	}

	// MA Bonus
	if params.Flags.UseMa {
		totalWeight += params.Weights.MaBonus
		gap := 0.0
		if res.Deviation20 != nil {
			gap = *res.Deviation20
		}
		if gap >= params.Thresholds.MaGapMin {
			total += params.Weights.MaBonus
			if params.Weights.MaBonus != 0 {
				components["ma"] = params.Weights.MaBonus
			}
		}
	}

	// Amplitude Bonus
	if params.Flags.UseAmp {
		totalWeight += params.Weights.AmpBonus
		amp := 0.0
		if res.Amplitude != nil {
			amp = *res.Amplitude
		}
		if amp >= params.Thresholds.AmpMin {
			total += params.Weights.AmpBonus
			if params.Weights.AmpBonus != 0 {
				components["amplitude"] = params.Weights.AmpBonus
			}
		}
	}

	// Range Bonus
	if params.Flags.UseRange {
		totalWeight += params.Weights.RangeBonus
		rangePos := 0.0
		if res.RangePos20 != nil {
			rangePos = *res.RangePos20
		}
		if rangePos >= (params.Thresholds.RangeMin / 100.0) {
			total += params.Weights.RangeBonus
			if params.Weights.RangeBonus != 0 {
				components["range"] = params.Weights.RangeBonus
			}
		}
	}

	if totalWeight > 0 {
		normalizedTotal := (total / totalWeight) * 100.0
		for k, v := range components {
			components[k] = (v / totalWeight) * 100.0
		}
		return normalizedTotal, components
	}

	return total, components
}

func calcForwardReturns(history []analysisDomain.DailyAnalysisResult, idx int, horizons []int) map[string]float64 {
	out := make(map[string]float64)
	if idx < 0 || idx >= len(history) {
		return out
	}
	base := history[idx]
	if base.Close <= 0 {
		return out
	}
	for _, h := range horizons {
		if h <= 0 {
			continue
		}
		target := idx + h
		if target >= len(history) {
			continue
		}
		next := history[target]
		if next.Close <= 0 {
			continue
		}
		out[fmt.Sprintf("d%d", h)] = (next.Close / base.Close) - 1
	}
	return out
}

func (s *Server) handleJobsStatus(c *gin.Context) {
	loc := taipeiLocation()
	s.jobMu.Lock()
	var last *jobRun
	if n := len(s.jobHistory); n > 0 {
		copy := s.jobHistory[n-1]
		last = &copy
	}
	base := s.lastAutoRun
	if base.IsZero() {
		base = time.Now()
	}
	nextRun := ""
	if s.autoInterval > 0 {
		nextRun = base.Add(s.autoInterval).In(loc).Format(time.RFC3339)
	}
	s.jobMu.Unlock()

	resp := gin.H{
		"success":               true,
		"next_run":              nextRun,
		"retry_strategy":        []string{"20:00", "20:30"},
		"timezone":              "Asia/Taipei",
		"use_synthetic":         s.useSynthetic,
		"auto_interval_seconds": int(s.autoInterval.Seconds()),
		"data_source":           s.dataSource,
	}
	if last != nil {
		resp["last_run"] = jobRunToMap(*last, loc)
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleJobsHistory(c *gin.Context) {
	limit := parseIntDefault(c.Query("limit"), 20)
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	loc := taipeiLocation()

	s.jobMu.Lock()
	n := len(s.jobHistory)
	start := 0
	if n > limit {
		start = n - limit
	}
	history := make([]jobRun, n-start)
	copy(history, s.jobHistory[start:])
	s.jobMu.Unlock()

	items := make([]map[string]interface{}, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		items = append(items, jobRunToMap(history[i], loc))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   len(items),
		"items":   items,
	})
}

// --- Strategies / Backtest / Trades ---

func (s *Server) handleCreateStrategy(c *gin.Context) {
	var body tradingDomain.Strategy
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	userID := currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "missing user", "error_code": errCodeUnauthorized})
		return
	}
	body.CreatedBy = userID
	body.UpdatedBy = userID
	strat, err := s.tradingSvc.CreateStrategy(c.Request.Context(), body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"strategy": strat,
	})
}

func (s *Server) handleListStrategies(c *gin.Context) {
	filter := trading.StrategyFilter{
		Name: c.Query("name"),
	}
	if status := c.Query("status"); status != "" {
		filter.Status = tradingDomain.Status(status)
	}
	if env := c.Query("env"); env != "" {
		filter.Env = tradingDomain.Environment(env)
	}
	list, err := s.tradingSvc.ListStrategies(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "list strategies failed", "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"strategies": list,
	})
}


func (s *Server) handleStrategyGetOrUpdate(c *gin.Context, id string) {
	switch c.Request.Method {
	case http.MethodGet:
		strat, err := s.tradingSvc.GetStrategy(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "strategy not found", "error_code": errCodeNotFound})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"strategy": strat,
		})
	case http.MethodDelete:
		if err := s.tradingSvc.DeleteStrategy(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
		})
	case http.MethodPut:
		var body tradingDomain.Strategy
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
			return
		}
		body.UpdatedBy = currentUserID(c)
		strat, err := s.tradingSvc.UpdateStrategy(c.Request.Context(), id, body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":  true,
			"strategy": strat,
		})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"success": false, "error": "method not allowed", "error_code": errCodeMethodNotAllowed})
	}
}

type strategyBacktestRequest struct {
	StartDate       string                  `json:"start_date"`
	EndDate         string                  `json:"end_date"`
	InitialEquity   float64                 `json:"initial_equity"`
	FeesPct         *float64                `json:"fees_pct"`
	SlippagePct     *float64                `json:"slippage_pct"`
	PriceMode       string                  `json:"price_mode"`
	StopLossPct     *float64                `json:"stop_loss_pct"`
	TakeProfitPct   *float64                `json:"take_profit_pct"`
	MaxDailyLossPct *float64                `json:"max_daily_loss_pct"`
	CoolDownDays    *int                    `json:"cool_down_days"`
	MinHoldDays     *int                    `json:"min_hold_days"`
	MaxPositions    *int                    `json:"max_positions"`
	Strategy        *tradingDomain.Strategy `json:"strategy,omitempty"`
}

func (s *Server) handleStrategyBacktest(c *gin.Context, strategyID string) {
	var body strategyBacktestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	input, err := buildBacktestInput(body, strategyID, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}
	input.Save = true
	input.CreatedBy = currentUserID(c)
	rec, err := s.tradingSvc.Backtest(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  rec,
	})
}

func (s *Server) handleInlineBacktest(c *gin.Context) {
	var body strategyBacktestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	if body.Strategy == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "strategy required", "error_code": errCodeBadRequest})
		return
	}
	input, err := buildBacktestInput(body, "", body.Strategy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}
	input.Save = false
	input.CreatedBy = currentUserID(c)
	rec, err := s.tradingSvc.Backtest(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  rec,
	})
}

func (s *Server) handleListStrategyBacktests(c *gin.Context, strategyID string) {
	list, err := s.tradingSvc.ListBacktests(c.Request.Context(), strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "list backtests failed", "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"backtests": list,
	})
}

func (s *Server) handleRunStrategy(c *gin.Context, strategyID string) {
	env := tradingDomain.EnvTest
	if e := c.Query("env"); e != "" {
		env = tradingDomain.Environment(e)
	}
	_, err := s.tradingSvc.RunOnce(c.Request.Context(), strategyID, env, currentUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Strategy executed manually",
	})
}

func (s *Server) handleActivateStrategy(c *gin.Context, strategyID string) {
	var body struct {
		Env                string  `json:"env"`
		AutoStopMinBalance float64 `json:"auto_stop_min_balance"`
	}
	_ = c.ShouldBindJSON(&body)

	if body.AutoStopMinBalance > 0 {
		// 嘗試獲取現有策略以更新風控設定
		rows, err := s.db.QueryContext(c.Request.Context(), "SELECT risk_settings FROM strategies WHERE id = $1", strategyID)
		if err == nil && rows.Next() {
			var riskRaw []byte
			if err := rows.Scan(&riskRaw); err == nil {
				var risk tradingDomain.RiskSettings
				_ = json.Unmarshal(riskRaw, &risk)
				risk.AutoStopMinBalance = body.AutoStopMinBalance
				_ = s.tradingSvc.UpdateRiskSettings(c.Request.Context(), strategyID, risk)
			}
			rows.Close()
		}
	}

	env := tradingDomain.Environment(body.Env)
	if env == "" {
		env = s.defaultEnv
	}
	if err := s.tradingSvc.SetStatus(c.Request.Context(), strategyID, tradingDomain.StatusActive, env); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"env":     env,
		"status":  tradingDomain.StatusActive,
	})
}

func (s *Server) handleDeactivateStrategy(c *gin.Context, strategyID string) {
	if err := s.tradingSvc.SetStatus(c.Request.Context(), strategyID, tradingDomain.StatusDraft, ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  tradingDomain.StatusDraft,
	})
}

func (s *Server) handleListTrades(c *gin.Context) {
	var startPtr, endPtr *time.Time
	if v := c.Query("start_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid start_date", "error_code": errCodeBadRequest})
			return
		}
		startPtr = &t
	}
	if v := c.Query("end_date"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid end_date", "error_code": errCodeBadRequest})
			return
		}
		endPtr = &t
	}
	filter := tradingDomain.TradeFilter{
		StrategyID: c.Query("strategy_id"),
		Env:        tradingDomain.Environment(c.Query("env")),
		StartDate:  startPtr,
		EndDate:    endPtr,
	}
	trades, err := s.tradingSvc.ListTrades(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "list trades failed", "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"trades":  trades,
	})
}
func (s *Server) handleManualBuy(c *gin.Context) {
	var body struct {
		Symbol string  `json:"symbol"`
		Amount float64 `json:"amount"`
		Env    string  `json:"env"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	if body.Symbol == "" || body.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "symbol and amount are required", "error_code": errCodeBadRequest})
		return
	}
	env := tradingDomain.Environment(body.Env)
	if env == "" {
		env = tradingDomain.EnvTest
	}
	userID := currentUserID(c)

	if err := s.tradingSvc.ExecuteManualBuy(c.Request.Context(), body.Symbol, body.Amount, env, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}


func (s *Server) handleListPositions(c *gin.Context) {
	positions, err := s.tradingSvc.ListPositions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "list positions failed", "error_code": errCodeInternal})
		return
	}
	envFilter := tradingDomain.Environment(c.Query("env"))
	if envFilter != "" {
		filtered := make([]tradingDomain.Position, 0, len(positions))
		for _, p := range positions {
			if p.Env == envFilter {
				filtered = append(filtered, p)
			}
		}
		positions = filtered
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"positions": positions,
	})
}

type reportRequest struct {
	Env         string      `json:"env"`
	PeriodStart string      `json:"period_start"`
	PeriodEnd   string      `json:"period_end"`
	Summary     interface{} `json:"summary"`
	TradesRef   interface{} `json:"trades_ref"`
}

func (s *Server) handleCreateReport(c *gin.Context, strategyID string) {
	var body reportRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	start, err := time.Parse("2006-01-02", body.PeriodStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid period_start", "error_code": errCodeBadRequest})
		return
	}
	end, err := time.Parse("2006-01-02", body.PeriodEnd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid period_end", "error_code": errCodeBadRequest})
		return
	}
	rep := tradingDomain.Report{
		StrategyID:  strategyID,
		Env:         tradingDomain.Environment(body.Env),
		PeriodStart: start,
		PeriodEnd:   end,
		Summary:     body.Summary,
		TradesRef:   body.TradesRef,
		CreatedBy:   currentUserID(c),
		CreatedAt:   time.Now(),
	}
	id, err := s.tradingSvc.SaveReport(c.Request.Context(), rep)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "save report failed", "error_code": errCodeInternal})
		return
	}
	rep.ID = id
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"report":  rep,
	})
}

func (s *Server) handleListReports(c *gin.Context, strategyID string) {
	reps, err := s.tradingSvc.ListReports(c.Request.Context(), strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "list reports failed", "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"reports": reps,
	})
}

func (s *Server) handleListLogs(c *gin.Context, strategyID string) {
	env := tradingDomain.Environment(c.Query("env"))
	limit := parseIntDefault(c.Query("limit"), 50)
	logs, err := s.tradingSvc.ListLogs(c.Request.Context(), tradingDomain.LogFilter{
		StrategyID: strategyID,
		Env:        env,
		Limit:      limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "list logs failed", "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"logs":    logs,
	})
}

// --- Notifier (Telegram) ---

func (s *Server) startTelegramJob() {
	interval := s.tgConfig.Interval
	if interval <= 0 {
		interval = time.Hour
	}
	go func() {
		// small delay to avoid competing with bootstrapping
		time.Sleep(5 * time.Second)
		s.pushTelegramSummary(context.Background())
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.pushTelegramSummary(context.Background())
		}
	}()
}

func (s *Server) pushTelegramSummary(ctx context.Context) {
	if s.tgClient == nil {
		return
	}
	latestDate, err := s.dataRepo.LatestAnalysisDate(ctx)
	if err != nil || latestDate.IsZero() {
		log.Printf("telegram push skipped: no analysis data yet: %v", err)
		return
	}

	limit := s.tgConfig.StrongLimit
	if limit <= 0 {
		limit = 5
	}
	scoreMin := s.tgConfig.ScoreMin
	if scoreMin <= 0 {
		scoreMin = 70
	}
	volMin := s.tgConfig.VolumeRatioMin
	if volMin <= 0 {
		volMin = 1.5
	}

	out, err := s.queryUC.QueryByDate(ctx, analysis.QueryByDateInput{
		Date: latestDate,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
		},
		Pagination: analysis.Pagination{Offset: 0, Limit: 500},
	})
	if err != nil || len(out.Results) == 0 {
		log.Printf("telegram push skipped: no analysis results to report")
		return
	}

	best := out.Results[0]
	for _, r := range out.Results {
		if r.Score > best.Score {
			best = r
		}
	}

	candidates := make([]analysisDomain.DailyAnalysisResult, 0, len(out.Results))
	for _, r := range out.Results {
		vol := 0.0
		if r.VolumeMultiple != nil {
			vol = *r.VolumeMultiple
		}
		if r.Score >= scoreMin && vol >= volMin {
			candidates = append(candidates, r)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("【BTC/USDT 強勢摘要】\n日期: %s\n", latestDate.Format("2006-01-02")))
	builder.WriteString(fmt.Sprintf("最高分: %s | 分數 %.1f | 收盤 %.2f | 日漲跌 %s | 近5日 %s | 量能 %s\n",
		best.Symbol,
		best.Score,
		best.Close,
		formatPercent(best.ChangeRate),
		formatOptionalPercent(best.Return5),
		formatOptionalTimes(best.VolumeMultiple),
	))

	if len(candidates) > 0 {
		builder.WriteString("Top 強勢交易對:\n")
		for i, item := range candidates {
			builder.WriteString(fmt.Sprintf("%d) %s | 分數 %.1f | 收盤 %.2f | 日漲跌 %s | 近5日 %s | 量能 %s\n",
				i+1,
				item.Symbol,
				item.Score,
				item.Close,
				formatPercent(item.ChangeRate),
				formatOptionalPercent(item.Return5),
				formatOptionalTimes(item.VolumeMultiple),
			))
		}
	} else {
		builder.WriteString("Top 強勢交易對: 目前無符合條件的結果\n")
	}

	if err := s.tgClient.SendMessage(ctx, builder.String()); err != nil {
		log.Printf("telegram push failed: %v", err)
		return
	}
	log.Printf("telegram push sent trade_date=%s items=%d", latestDate.Format("2006-01-02"), len(candidates))
}

// startAutoPipeline 每隔 autoInterval 自動跑當日 ingestion + analysis。
func (s *Server) startAutoPipeline() {
	interval := s.autoInterval
	if interval <= 0 {
		return
	}
	go func() {
		time.Sleep(3 * time.Second)
		s.runPipelineOnce()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.runPipelineOnce()
		}
	}()
}

// startConfigBackfill 依組態設定的起始日，啟動一次性補資料與分析（僅補尚未分析的日期）。
func (s *Server) startConfigBackfill() {
	startDateStr := s.backfillStart
	if startDateStr == "" {
		return
	}
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		log.Printf("config backfill skipped: invalid date %s", startDateStr)
		return
	}
	go func() {
		endDate := time.Now().UTC().Truncate(24 * time.Hour)
		if startDate.After(endDate) {
			log.Printf("config backfill skipped: start_date %s after today", startDateStr)
			return
		}
		ctx := context.Background()
		log.Printf("config backfill start from=%s to=%s synthetic=%t", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), s.useSynthetic)
		days := 0
		completed := 0
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			days++
			has, err := s.dataRepo.HasAnalysisForDate(ctx, d)
			if err != nil {
				log.Printf("config backfill check failed date=%s: %v", d.Format("2006-01-02"), err)
				continue
			}
			if has {
				continue
			}
			var ingestErr error
			if s.useSynthetic {
				ingestErr = s.generateDailyPrices(ctx, d)
			} else {
				ingestErr = s.generateDailyPricesStrict(ctx, d)
			}
			if ingestErr != nil {
				log.Printf("config backfill ingestion failed date=%s: %v", d.Format("2006-01-02"), ingestErr)
				continue
			}
			if _, err := s.runAnalysisForDate(ctx, d); err != nil {
				log.Printf("config backfill analysis failed date=%s: %v", d.Format("2006-01-02"), err)
				continue
			}
			completed++
		}
		log.Printf("config backfill done total_days=%d completed=%d", days, completed)
	}()
}

func (s *Server) runPipelineOnce() {
	tradeDate := time.Now().UTC().Truncate(24 * time.Hour)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("auto pipeline start trade_date=%s", tradeDate.Format("2006-01-02"))
	ingestionOK := true
	ingestionErr := ""
	dataSource := s.dataSource
	if err := s.generateDailyPrices(ctx, tradeDate); err != nil {
		ingestionErr = err.Error()
		log.Printf("auto ingestion error: %v; fallback synthetic", err)
		if fbErr := s.generateSyntheticBTC(ctx, tradeDate); fbErr != nil {
			log.Printf("auto ingestion fallback failed: %v", fbErr)
			ingestionOK = false
		} else {
			dataSource = "synthetic (fallback)"
		}
	}

	prices, err := s.dataRepo.PricesByDate(ctx, tradeDate)
	if err != nil {
		log.Printf("auto analysis: read prices failed: %v", err)
		s.recordJob(jobRun{
			Kind:         "auto",
			Start:        start,
			End:          time.Now(),
			IngestionOK:  ingestionOK,
			IngestionErr: ingestionErr,
			AnalysisOn:   true,
			AnalysisErr:  err.Error(),
		})
		return
	}
	if len(prices) == 0 {
		log.Printf("auto analysis: no prices for trade_date=%s", tradeDate.Format("2006-01-02"))
		s.recordJob(jobRun{
			Kind:         "auto",
			Start:        start,
			End:          time.Now(),
			IngestionOK:  ingestionOK,
			IngestionErr: ingestionErr,
			AnalysisOn:   true,
			AnalysisErr:  "no prices for trade_date",
		})
		return
	}

	success := 0
	for _, p := range prices {
		stockID, err := s.dataRepo.UpsertTradingPair(ctx, p.Symbol, p.Symbol, p.Market, "")
		if err != nil {
			log.Printf("auto pipeline upsert pair failed symbol=%s: %v", p.Symbol, err)
			continue
		}
		res := s.calculateAnalysis(ctx, p)
		if err := s.dataRepo.InsertAnalysisResult(ctx, stockID, res); err != nil {
			log.Printf("auto pipeline write analysis failed symbol=%s: %v", p.Symbol, err)
			continue
		}
		success++
	}
	s.recordJob(jobRun{
		Kind:          "auto",
		TriggeredBy:   "system",
		Start:         start,
		End:           time.Now(),
		IngestionOK:   ingestionOK,
		IngestionErr:  ingestionErr,
		AnalysisOn:    true,
		AnalysisOK:    success == len(prices),
		AnalysisTotal: len(prices),
		AnalysisSucc:  success,
		AnalysisFail:  len(prices) - success,
		DataSource:    dataSource,
	})
	log.Printf("auto pipeline done trade_date=%s success=%d total=%d", tradeDate.Format("2006-01-02"), success, len(prices))
}

// --- Helpers ---

func (s *Server) setRefreshCookie(c *gin.Context, token string, expiry time.Time) {
	// 為了透過 ngrok/https 跨網域攜帶 cookie，強制使用 SameSite=None 且 Secure=true。
	if token == "" {
		c.SetSameSite(http.SameSiteNoneMode)
		c.SetCookie(refreshCookieName, "", -1, "/", "", true, true)
		return
	}
	seconds := int(time.Until(expiry).Seconds())
	if seconds < 0 {
		seconds = 0
	}
	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie(refreshCookieName, token, seconds, "/", "", true, true)
}

func errorString(err error) interface{} {
	if err == nil {
		return nil
	}
	return err.Error()
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func optionalString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func taipeiLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		return time.FixedZone("Asia/Taipei", 8*3600)
	}
	return loc
}

func jobRunToMap(j jobRun, loc *time.Location) map[string]interface{} {
	start := j.Start.In(loc)
	end := j.End.In(loc)
	duration := int(end.Sub(start).Seconds())
	return map[string]interface{}{
		"kind":             j.Kind,
		"triggered_by":     optionalString(j.TriggeredBy),
		"start":            start.Format(time.RFC3339),
		"end":              end.Format(time.RFC3339),
		"duration_seconds": duration,
		"data_source":      optionalString(j.DataSource),
		"ingestion": map[string]interface{}{
			"success": j.IngestionOK,
			"error":   optionalString(j.IngestionErr),
		},
		"analysis": map[string]interface{}{
			"enabled":       j.AnalysisOn,
			"success":       j.AnalysisOK,
			"total":         j.AnalysisTotal,
			"success_count": j.AnalysisSucc,
			"failure_count": j.AnalysisFail,
			"error":         optionalString(j.AnalysisErr),
		},
		"failures": j.Failures,
	}
}

func (s *Server) recordJob(j jobRun) {
	s.jobMu.Lock()
	defer s.jobMu.Unlock()
	if j.DataSource == "" {
		j.DataSource = s.dataSource
	}
	s.jobHistory = append(s.jobHistory, j)
	if len(s.jobHistory) > 50 {
		s.jobHistory = s.jobHistory[len(s.jobHistory)-50:]
	}
	if j.Kind == "auto" {
		s.lastAutoRun = j.End
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (s *Server) requireAuth(perm auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := parseBearer(c.GetHeader("Authorization"))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized", "error_code": errCodeUnauthorized})
			c.Abort()
			return
		}
		claims, err := s.tokenSvc.ParseAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid token", "error_code": errCodeUnauthorized})
			c.Abort()
			return
		}
		user, err := s.authRepo.FindByID(c.Request.Context(), claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid token", "error_code": errCodeUnauthorized})
			c.Abort()
			return
		}
		res, err := s.authz.Authorize(c.Request.Context(), auth.AuthorizeInput{
			UserID:   user.ID,
			Required: []auth.Permission{perm},
		})
		if err != nil {
			log.Printf("auth check failed user_id=%s: %v", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal error", "error_code": errCodeInternal})
			c.Abort()
			return
		}
		if !res.Allowed {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden", "error_code": errCodeForbidden})
			c.Abort()
			return
		}
		c.Set("userID", user.ID)
		c.Header("X-User-Role", string(user.Role))
		c.Next()
	}
}



func parseBearer(h string) string {
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func currentUserID(c *gin.Context) string {
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func parseStrategyPath(path string) (string, string) {
	const prefix = "/api/admin/strategies/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", ""
	}
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func buildBacktestInput(body strategyBacktestRequest, strategyID string, inline *tradingDomain.Strategy) (trading.BacktestInput, error) {
	var input trading.BacktestInput
	if body.StartDate == "" || body.EndDate == "" {
		return input, fmt.Errorf("start_date and end_date required")
	}
	start, err := time.Parse("2006-01-02", body.StartDate)
	if err != nil {
		return input, fmt.Errorf("invalid start_date")
	}
	end, err := time.Parse("2006-01-02", body.EndDate)
	if err != nil {
		return input, fmt.Errorf("invalid end_date")
	}
	pm := tradingDomain.PriceMode(body.PriceMode)
	if pm == "" {
		pm = tradingDomain.PriceNextOpen
	}
	input = trading.BacktestInput{
		StrategyID:      strategyID,
		Inline:          inline,
		StartDate:       start,
		EndDate:         end,
		InitialEquity:   body.InitialEquity,
		FeesPct:         body.FeesPct,
		SlippagePct:     body.SlippagePct,
		PriceMode:       &pm,
		StopLossPct:     body.StopLossPct,
		TakeProfitPct:   body.TakeProfitPct,
		MaxDailyLossPct: body.MaxDailyLossPct,
		CoolDownDays:    body.CoolDownDays,
		MinHoldDays:     body.MinHoldDays,
		MaxPositions:    body.MaxPositions,
	}
	return input, nil
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}

func parseFloatDefault(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func parseBoolDefault(s string, def bool) bool {
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}

func formatPercent(v float64) string {
	return fmt.Sprintf("%.2f%%", v*100)
}

func formatOptionalPercent(v *float64) string {
	if v == nil {
		return "-"
	}
	return formatPercent(*v)
}

func formatOptionalTimes(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.2fx", *v)
}

// generateDailyPrices 從外部來源取得 BTCUSDT 的日 K，失敗時由上層 fallback。
func (s *Server) generateDailyPrices(ctx context.Context, tradeDate time.Time) error {
	if s.useSynthetic {
		return s.generateSyntheticBTC(ctx, tradeDate)
	}
	series, err := s.fetchBTCSeries(ctx, tradeDate)
	if err != nil {
		return err
	}
	return s.storeBTCSeries(ctx, series)
}

func (s *Server) generateDailyPricesStrict(ctx context.Context, tradeDate time.Time) error {
	series, err := s.fetchBTCSeries(ctx, tradeDate)
	if err != nil {
		return err
	}
	return s.storeBTCSeries(ctx, series)
}

func (s *Server) storeBTCSeries(ctx context.Context, series []dataDomain.DailyPrice) error {
	for _, p := range series {
		stockID, err := s.dataRepo.UpsertTradingPair(ctx, p.Symbol, "Bitcoin", dataDomain.MarketCrypto, "Crypto")
		if err != nil {
			return err
		}
		if err := s.dataRepo.InsertDailyPrice(ctx, stockID, p); err != nil {
			return err
		}
	}
	return nil
}

// generateSyntheticBTC 為無法取數時的預設資料（含近 5 日）。
func (s *Server) generateSyntheticBTC(ctx context.Context, tradeDate time.Time) error {
	series := []struct {
		offset int
		open   float64
		high   float64
		low    float64
		close  float64
		volume int64
	}{
		{5, 38000, 38500, 37500, 38200, 150000},
		{4, 38200, 38800, 38000, 38700, 152000},
		{3, 38700, 39500, 38500, 39400, 160000},
		{2, 39400, 40000, 39000, 39800, 170000},
		{1, 39800, 40400, 39600, 40200, 175000},
		{0, 40200, 41000, 40000, 40800, 190000},
	}
	for _, srs := range series {
		d := tradeDate.AddDate(0, 0, -srs.offset)
		stockID, err := s.dataRepo.UpsertTradingPair(ctx, "BTCUSDT", "Bitcoin", dataDomain.MarketCrypto, "Crypto")
		if err != nil {
			return err
		}
		price := dataDomain.DailyPrice{
			Symbol:    "BTCUSDT",
			Market:    dataDomain.MarketCrypto,
			TradeDate: d,
			Open:      srs.open,
			High:      srs.high,
			Low:       srs.low,
			Close:     srs.close,
			Volume:    srs.volume,
			Timeframe: "1d",
		}
		if err := s.dataRepo.InsertDailyPrice(ctx, stockID, price); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) insertPriceSeries(ctx context.Context, code string, market dataDomain.Market, tradeDate time.Time, open, high, low, close float64, volume int64) error {
	// generate last 5 days synthetic if missing
	for i := 5; i >= 1; i-- {
		d := tradeDate.AddDate(0, 0, -i)
		existing, err := s.dataRepo.PricesByDate(ctx, d)
		if err != nil {
			return err
		}
		if len(existing) == 0 {
			stockID, err := s.dataRepo.UpsertTradingPair(ctx, code, code, market, "")
			if err != nil {
				return err
			}
			price := dataDomain.DailyPrice{
				Symbol:    code,
				Market:    market,
				TradeDate: d,
				Open:      open - float64(5+i),
				High:      high - float64(5+i),
				Low:       low - float64(5+i),
				Close:     close - float64(5+i),
				Volume:    volume / 2,
			}
			if err := s.dataRepo.InsertDailyPrice(ctx, stockID, price); err != nil {
				return err
			}
		}
	}
	// today
	stockID, err := s.dataRepo.UpsertTradingPair(ctx, code, code, market, "")
	if err != nil {
		return err
	}
	price := dataDomain.DailyPrice{
		Symbol:    code,
		Market:    market,
		TradeDate: tradeDate,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}
	if err := s.dataRepo.InsertDailyPrice(ctx, stockID, price); err != nil {
		return err
	}
	return nil
}

func (s *Server) calculateAnalysis(ctx context.Context, p dataDomain.DailyPrice) analysisDomain.DailyAnalysisResult {
	history, _ := s.dataRepo.PricesByPair(ctx, p.Symbol, p.Timeframe)
	var return5 *float64
	var volumeRatio *float64
	var changeRate float64
	idx := -1
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].TradeDate.Equal(p.TradeDate) {
			idx = i
			break
		}
	}
	if idx > 0 {
		prev := history[idx-1]
		if prev.Close > 0 {
			changeRate = (p.Close - prev.Close) / prev.Close
		}
	}
	if idx >= 5 {
		earlier := history[idx-5]
		if earlier.Close > 0 {
			val := (p.Close / earlier.Close) - 1
			return5 = &val
		}
	}
	if idx >= 5 {
		var sumVol float64
		for i := idx - 4; i <= idx; i++ {
			sumVol += float64(history[i].Volume)
		}
		avg := sumVol / 5
		if avg > 0 {
			vr := float64(p.Volume) / avg
			volumeRatio = &vr
		}
	}
	score := simpleScore(return5, changeRate, volumeRatio)

	return analysisDomain.DailyAnalysisResult{
		Symbol:         p.Symbol,
		Market:         p.Market,
		Timeframe:      p.Timeframe,
		Industry:       "",
		TradeDate:      p.TradeDate,
		Version:        "v1-mvp",
		Close:          p.Close,
		ChangeRate:     changeRate,
		Return5:        return5,
		Volume:         p.Volume,
		VolumeMultiple: volumeRatio,
		Score:          score,
		Success:        true,
	}
}

func simpleScore(ret5 *float64, changeRate float64, volumeRatio *float64) float64 {
	score := 50.0
	if ret5 != nil {
		score += *ret5 * 100
	}
	score += changeRate * 100
	if volumeRatio != nil {
		score += (*volumeRatio - 1) * 10
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// fetchBTCSeries 從 Binance 抓取 BTCUSDT 1d K 線，包含指定日期與前 5 日。
func (s *Server) fetchBTCSeries(ctx context.Context, tradeDate time.Time) ([]dataDomain.DailyPrice, error) {
	start := tradeDate.AddDate(0, 0, -5)
	end := tradeDate.AddDate(0, 0, 1)
	url := "https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1d&startTime=" +
		strconv.FormatInt(start.UnixMilli(), 10) + "&endTime=" + strconv.FormatInt(end.UnixMilli(), 10)

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)

		if err != nil {
			lastErr = err
		} else {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				lastErr = fmt.Errorf("binance response not ok: status %d, body: %s", resp.StatusCode, string(body))
			} else {
				var raw [][]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
					lastErr = err
				} else {
					var out []dataDomain.DailyPrice
					for _, row := range raw {
						if len(row) < 6 {
							continue
						}
						openTime, ok := row[0].(float64)
						if !ok {
							continue
						}
						open, _ := strconv.ParseFloat(row[1].(string), 64)
						high, _ := strconv.ParseFloat(row[2].(string), 64)
						low, _ := strconv.ParseFloat(row[3].(string), 64)
						closeP, _ := strconv.ParseFloat(row[4].(string), 64)
						vol, _ := strconv.ParseFloat(row[5].(string), 64)

						day := time.UnixMilli(int64(openTime)).UTC()
						out = append(out, dataDomain.DailyPrice{
							Symbol:    "BTCUSDT",
							Market:    dataDomain.MarketCrypto,
							Timeframe: "1d",
							TradeDate: day,
							Open:      open,
							High:      high,
							Low:       low,
							Close:     closeP,
							Volume:    int64(vol),
						})
					}
					if len(out) == 0 {
						lastErr = errors.New("no kline data")
					} else {
						return out, nil
					}
				}
			}
		}

		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, lastErr
}

func (s *Server) handleBinanceAccount(c *gin.Context) {
	if s.binanceClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "binance client not initialized", "error_code": errCodeInternal})
		return
	}
	info, err := s.binanceClient.GetAccountInfo()
	if err != nil {
		// If we are in Paper mode, don't return an error even if key is invalid.
		// Return a mock balance instead.
		if s.defaultEnv == tradingDomain.EnvPaper {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"is_mock": true,
				"account": gin.H{
					"accountType": "SPOT",
					"balances": []gin.H{
						{"asset": "USDT", "free": "0.00", "locked": "0.00"},
						{"asset": "BTC", "free": "0.000000", "locked": "0.000000"},
					},
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"account": info,
	})
}

func (s *Server) handleBinancePrice(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	price, err := s.tradingSvc.GetExchangePrice(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"symbol":  symbol,
		"price":   price,
	})
}

func (s *Server) handlePositionClose(c *gin.Context, id string) {
	if err := s.tradingSvc.ClosePositionManually(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

func (s *Server) handleGetBinanceConfig(c *gin.Context) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"active_env": s.defaultEnv,
	})
}

func (s *Server) handleUpdateBinanceConfig(c *gin.Context) {
	var body struct {
		ActiveEnv string `json:"active_env"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	newEnv := tradingDomain.Environment(body.ActiveEnv)
	// Simple validation
	switch newEnv {
	case tradingDomain.EnvProd, tradingDomain.EnvPaper, tradingDomain.EnvTest:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "unsupported environment", "error_code": errCodeBadRequest})
		return
	}

	s.configMu.Lock()
	defer s.configMu.Unlock()

	if s.binanceClient != nil {
		s.binanceClient.SetBaseURL(newEnv == tradingDomain.EnvTest)
	}
	s.defaultEnv = newEnv

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("System environment switched to %s", newEnv),
	})
}

