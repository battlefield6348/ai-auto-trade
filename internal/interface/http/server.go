package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/auth"
	"ai-auto-trade/internal/application/mvp"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	"ai-auto-trade/internal/infra/memory"
)

const (
	errCodeBadRequest         = "BAD_REQUEST"
	errCodeInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	errCodeUnauthorized       = "AUTH_UNAUTHORIZED"
	errCodeForbidden          = "AUTH_FORBIDDEN"
	errCodeAnalysisNotReady   = "ANALYSIS_NOT_READY"
	errCodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
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
	PricesByPair(ctx context.Context, pair string) ([]dataDomain.DailyPrice, error)
	InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error
	HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error)
	LatestAnalysisDate(ctx context.Context) (time.Time, error)
}

// memoryRepoAdapter 讓 memory.Store 相容 DataRepository。
type memoryRepoAdapter struct {
	store *memory.Store
}

type memoryPresetStore struct {
	store *memory.Store
}

func (m memoryPresetStore) Save(ctx context.Context, userID string, config []byte) error {
	return m.store.Save(ctx, config, userID)
}

func (m memoryPresetStore) Load(ctx context.Context, userID string) ([]byte, error) {
	cfg, err := m.store.Load(ctx, userID)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (m memoryPresetStore) NotFound(err error) bool {
	return memory.IsPresetNotFound(err)
}

func (m memoryPresetStore) SaveNamed(ctx context.Context, userID, name string, config []byte) (string, error) {
	return m.store.SaveNamed(ctx, userID, name, config)
}

func (m memoryPresetStore) List(ctx context.Context, userID string) ([]presetRecord, error) {
	rows, err := m.store.ListPresets(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]presetRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, presetRecord{
			ID:        r.ID,
			Name:      r.Name,
			Config:    r.Config,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

func (m memoryPresetStore) Delete(ctx context.Context, userID, id string) error {
	return m.store.DeletePreset(ctx, userID, id)
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
func (m memoryRepoAdapter) PricesByPair(ctx context.Context, pair string) ([]dataDomain.DailyPrice, error) {
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

func (m memoryRepoAdapter) FindByDate(ctx context.Context, date time.Time, filter analysis.QueryFilter, sort analysis.SortOption, pagination analysis.Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	return m.store.FindByDate(ctx, date, filter, sort, pagination)
}
func (m memoryRepoAdapter) FindHistory(ctx context.Context, symbol string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error) {
	return m.store.FindHistory(ctx, symbol, from, to, limit, onlySuccess)
}
func (m memoryRepoAdapter) Get(ctx context.Context, symbol string, date time.Time) (analysisDomain.DailyAnalysisResult, error) {
	return m.store.Get(ctx, symbol, date)
}

type analysisRunSummary struct {
	total   int
	success int
	failure int
}

type backtestConfig struct {
	Symbol  string `json:"symbol"`
	Start   string `json:"start_date"`
	End     string `json:"end_date"`
	Weights struct {
		Score       float64 `json:"score"`
		ChangeBonus float64 `json:"change_bonus"`
		VolumeBonus float64 `json:"volume_bonus"`
		ReturnBonus float64 `json:"return_bonus"`
		MABonus     float64 `json:"ma_bonus"`
	} `json:"weights"`
	Thresholds struct {
		TotalMin       float64 `json:"total_min"`
		ChangeMin      float64 `json:"change_min"`
		VolumeRatioMin float64 `json:"volume_ratio_min"`
		Return5Min     float64 `json:"return5_min"`
		MAGapMin       float64 `json:"ma_gap_min"`
	} `json:"thresholds"`
	Flags struct {
		UseChange *bool `json:"use_change"`
		UseVolume *bool `json:"use_volume"`
		UseReturn *bool `json:"use_return"`
		UseMA     *bool `json:"use_ma"`
	} `json:"flags"`
	Horizons []int `json:"horizons"`
}

type backfillFailure struct {
	TradeDate string `json:"trade_date"`
	Stage     string `json:"stage"`
	Reason    string `json:"reason"`
}

type backtestPresetStore interface {
	Save(ctx context.Context, userID string, config []byte) error
	Load(ctx context.Context, userID string) ([]byte, error)
	SaveNamed(ctx context.Context, userID, name string, config []byte) (string, error)
	List(ctx context.Context, userID string) ([]presetRecord, error)
	Delete(ctx context.Context, userID, id string) error
	NotFound(err error) bool
}

type presetRecord struct {
	ID        string
	Name      string
	Config    []byte
	CreatedAt time.Time
	UpdatedAt time.Time
}

// --- Handlers ---

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "pong",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "unavailable"
	if s.db == nil {
		dbStatus = "not_configured"
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.db.PingContext(ctx); err == nil {
			dbStatus = "ok"
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"db":             dbStatus,
		"use_synthetic":  s.useSynthetic,
		"analysis_ready": s.tokenSvc != nil,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid body")
		return
	}
	res, err := s.loginUC.Execute(r.Context(), auth.LoginInput{
		Email:     body.Email,
		Password:  body.Password,
		UserAgent: r.UserAgent(),
		IP:        clientIP(r),
	})
	if err != nil {
		log.Printf("login failed email=%s: %v", body.Email, err)
		writeError(w, http.StatusUnauthorized, errCodeInvalidCredentials, "invalid credentials")
		return
	}
	log.Printf("login success user_id=%s role=%s email=%s", res.User.ID, res.User.Role, res.User.Email)

	s.setRefreshCookie(w, r, res.Token.RefreshToken, res.Token.RefreshExpiry)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":            true,
		"access_token":       res.Token.AccessToken,
		"token_type":         "Bearer",
		"expires_in":         int(s.tokenTTL.Seconds()),
		"refresh_expires_in": int(s.refreshTTL.Seconds()),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, errCodeUnauthorized, "missing refresh token")
		return
	}
	pair, err := s.tokenSvc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		log.Printf("refresh token failed: %v", err)
		writeError(w, http.StatusUnauthorized, errCodeUnauthorized, "refresh token expired or invalid")
		return
	}
	s.setRefreshCookie(w, r, pair.RefreshToken, pair.RefreshExpiry)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":            true,
		"access_token":       pair.AccessToken,
		"token_type":         "Bearer",
		"expires_in":         int(time.Until(pair.AccessExpiry).Seconds()),
		"refresh_expires_in": int(time.Until(pair.RefreshExpiry).Seconds()),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		if s.logoutUC != nil {
			if revokeErr := s.logoutUC.Execute(r.Context(), cookie.Value); revokeErr != nil {
				log.Printf("logout revoke refresh failed: %v", revokeErr)
			}
		}
	}
	s.setRefreshCookie(w, r, "", time.Now().Add(-time.Hour))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "logged out",
	})
}

func (s *Server) handleIngestionBackfill(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		RunAnalysis *bool  `json:"run_analysis"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid body")
		return
	}
	if body.StartDate == "" || body.EndDate == "" {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "start_date and end_date required")
		return
	}
	startDate, err := time.Parse("2006-01-02", body.StartDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid start_date")
		return
	}
	endDate, err := time.Parse("2006-01-02", body.EndDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid end_date")
		return
	}
	if endDate.Before(startDate) {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "end_date must be after start_date")
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
			ingestErr = s.generateDailyPrices(r.Context(), d)
		} else {
			ingestErr = s.generateDailyPricesStrict(r.Context(), d)
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
			if _, err := s.runAnalysisForDate(r.Context(), d); err != nil {
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
	writeJSON(w, http.StatusOK, map[string]interface{}{
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

func (s *Server) handleAnalysisQuery(w http.ResponseWriter, r *http.Request) {
	tradeDateStr := r.URL.Query().Get("trade_date")
	if tradeDateStr == "" {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "trade_date required")
		return
	}
	tradeDate, err := time.Parse("2006-01-02", tradeDateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid trade_date")
		return
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 100)
	if limit > 1000 {
		limit = 1000
	}
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	out, err := s.queryUC.QueryByDate(r.Context(), analysis.QueryByDateInput{
		Date: tradeDate,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
		},
		Pagination: analysis.Pagination{
			Offset: offset,
			Limit:  limit,
		},
	})
	if err != nil {
		log.Printf("analysis query failed date=%s: %v", tradeDate.Format("2006-01-02"), err)
		writeError(w, http.StatusInternalServerError, errCodeInternal, "internal error")
		return
	}
	if out.Total == 0 {
		writeError(w, http.StatusNotFound, errCodeAnalysisNotReady, "analysis results not ready for trade_date")
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
			Volume:        r.Volume,
			VolumeRatio:   r.VolumeMultiple,
			Score:         r.Score,
		})
	}

	log.Printf("analysis query done date=%s total=%d limit=%d offset=%d", tradeDate.Format("2006-01-02"), out.Total, limit, offset)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"trade_date":  tradeDate.Format("2006-01-02"),
		"total_count": out.Total,
		"items":       items,
	})
}

func (s *Server) handleAnalysisHistory(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	if symbol == "" {
		symbol = "BTCUSDT"
	}

	var startDate *time.Time
	startDateStr := r.URL.Query().Get("start_date")
	if startDateStr != "" {
		val, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid start_date")
			return
		}
		startDate = &val
	}
	var endDate *time.Time
	endDateStr := r.URL.Query().Get("end_date")
	if endDateStr != "" {
		val, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid end_date")
			return
		}
		endDate = &val
	}
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "end_date must be after start_date")
		return
	}

	limit := parseIntDefault(r.URL.Query().Get("limit"), 1000)
	onlySuccess := parseBoolDefault(r.URL.Query().Get("only_success"), true)

	out, err := s.queryUC.QueryHistory(r.Context(), analysis.QueryHistoryInput{
		Symbol:      symbol,
		From:        startDate,
		To:          endDate,
		Limit:       limit,
		OnlySuccess: onlySuccess,
	})
	if err != nil {
		log.Printf("analysis history failed symbol=%s: %v", symbol, err)
		writeError(w, http.StatusInternalServerError, errCodeInternal, "internal error")
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
			Volume:        r.Volume,
			VolumeRatio:   r.VolumeMultiple,
			Score:         r.Score,
			Success:       r.Success,
		})
	}

	respStart := ""
	respEnd := ""
	if startDate != nil {
		respStart = startDate.Format("2006-01-02")
	}
	if endDate != nil {
		respEnd = endDate.Format("2006-01-02")
	}
	if respStart == "" && len(out) > 0 {
		respStart = out[0].TradeDate.Format("2006-01-02")
	}
	if respEnd == "" && len(out) > 0 {
		respEnd = out[len(out)-1].TradeDate.Format("2006-01-02")
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"symbol":      symbol,
		"start_date":  respStart,
		"end_date":    respEnd,
		"total_count": len(out),
		"items":       items,
	})
}

func (s *Server) handleAnalysisSummary(w http.ResponseWriter, r *http.Request) {
	latestDate, err := s.dataRepo.LatestAnalysisDate(r.Context())
	if err != nil || latestDate.IsZero() {
		writeError(w, http.StatusNotFound, errCodeAnalysisNotReady, "analysis results not ready")
		return
	}
	out, err := s.queryUC.QueryByDate(r.Context(), analysis.QueryByDateInput{
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
		writeError(w, http.StatusNotFound, errCodeAnalysisNotReady, "analysis results not ready")
		return
	}

	// 以最高分數的交易對作為當前趨勢參考
	best := out.Results[0]
	for _, r := range out.Results {
		if r.Score > best.Score {
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

func (s *Server) handleAnalysisBacktest(w http.ResponseWriter, r *http.Request) {
	var body backtestConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid body")
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(body.Symbol))
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	if body.Start == "" || body.End == "" {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "start_date and end_date required")
		return
	}
	startDate, err := time.Parse("2006-01-02", body.Start)
	if err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid start_date")
		return
	}
	endDate, err := time.Parse("2006-01-02", body.End)
	if err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid end_date")
		return
	}
	if endDate.Before(startDate) {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "end_date must be after start_date")
		return
	}

	weights := body.Weights
	if weights.Score == 0 {
		weights.Score = 1
	}
	if weights.ChangeBonus == 0 {
		weights.ChangeBonus = 10
	}
	if weights.VolumeBonus == 0 {
		weights.VolumeBonus = 10
	}
	if weights.ReturnBonus == 0 {
		weights.ReturnBonus = 8
	}
	if weights.MABonus == 0 {
		weights.MABonus = 5
	}
	thresholds := body.Thresholds
	if thresholds.TotalMin == 0 {
		thresholds.TotalMin = 60
	}
	if thresholds.ChangeMin == 0 {
		thresholds.ChangeMin = 0.0
	}
	if thresholds.VolumeRatioMin == 0 {
		thresholds.VolumeRatioMin = 1.0
	}
	if thresholds.Return5Min == 0 {
		thresholds.Return5Min = 0.0
	}
	if thresholds.MAGapMin == 0 {
		thresholds.MAGapMin = 0.0
	}
	horizons := body.Horizons
	if len(horizons) == 0 {
		horizons = []int{3, 5, 10}
	}
	useChange := true
	useVolume := true
	useReturn := false
	useMA := false
	if body.Flags.UseChange != nil {
		useChange = *body.Flags.UseChange
	}
	if body.Flags.UseVolume != nil {
		useVolume = *body.Flags.UseVolume
	}
	if body.Flags.UseReturn != nil {
		useReturn = *body.Flags.UseReturn
	}
	if body.Flags.UseMA != nil {
		useMA = *body.Flags.UseMA
	}

	out, err := s.queryUC.QueryHistory(r.Context(), analysis.QueryHistoryInput{
		Symbol:      symbol,
		From:        &startDate,
		To:          &endDate,
		Limit:       2000,
		OnlySuccess: true,
	})
	if err != nil {
		log.Printf("analysis backtest failed symbol=%s: %v", symbol, err)
		writeError(w, http.StatusInternalServerError, errCodeInternal, "internal error")
		return
	}
	if len(out) == 0 {
		writeError(w, http.StatusNotFound, errCodeAnalysisNotReady, "analysis results not ready")
		return
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TradeDate.Before(out[j].TradeDate)
	})

	type event struct {
		TradingPair   string              `json:"trading_pair"`
		TradeDate     string              `json:"trade_date"`
		ClosePrice    float64             `json:"close_price"`
		ChangePercent float64             `json:"change_percent"`
		Return5d      *float64            `json:"return_5d,omitempty"`
		VolumeRatio   *float64            `json:"volume_ratio,omitempty"`
		Score         float64             `json:"score"`
		TotalScore    float64             `json:"total_score"`
		Components    map[string]float64  `json:"components"`
		Forward       map[string]*float64 `json:"forward_returns"`
		MAGap         *float64            `json:"ma_gap,omitempty"`
	}

	events := make([]event, 0)
	retSums := make(map[int]float64)
	retWins := make(map[int]int)
	retCounts := make(map[int]int)

	for idx, row := range out {
		if row.Close == 0 {
			continue
		}
		total := row.Score * weights.Score
		components := map[string]float64{
			"score": total,
		}
		if useChange && row.ChangeRate >= thresholds.ChangeMin {
			total += weights.ChangeBonus
			components["change_bonus"] = weights.ChangeBonus
		}
		if useVolume && row.VolumeMultiple != nil && *row.VolumeMultiple >= thresholds.VolumeRatioMin {
			total += weights.VolumeBonus
			components["volume_bonus"] = weights.VolumeBonus
		}
		if useReturn && row.Return5 != nil && *row.Return5 >= thresholds.Return5Min {
			total += weights.ReturnBonus
			components["return_bonus"] = weights.ReturnBonus
		}
		if useMA && row.MA20 != nil && row.Close > 0 {
			gap := (row.Close / *row.MA20) - 1
			if gap >= thresholds.MAGapMin {
				total += weights.MABonus
				components["ma_bonus"] = weights.MABonus
			}
		}
		if total < thresholds.TotalMin {
			continue
		}
		fwd := make(map[string]*float64)
		for _, h := range horizons {
			targetIdx := idx + h
			if targetIdx >= len(out) {
				fwd[fmt.Sprintf("d%d", h)] = nil
				continue
			}
			base := row.Close
			next := out[targetIdx].Close
			if base == 0 {
				fwd[fmt.Sprintf("d%d", h)] = nil
				continue
			}
			ret := (next / base) - 1
			retVal := ret
			fwd[fmt.Sprintf("d%d", h)] = &retVal
			retSums[h] += ret
			retCounts[h]++
			if ret > 0 {
				retWins[h]++
			}
		}
		events = append(events, event{
			TradingPair:   row.Symbol,
			TradeDate:     row.TradeDate.Format("2006-01-02"),
			ClosePrice:    row.Close,
			ChangePercent: row.ChangeRate,
			Return5d:      row.Return5,
			VolumeRatio:   row.VolumeMultiple,
			Score:         row.Score,
			TotalScore:    total,
			Components:    components,
			Forward:       fwd,
			MAGap: func() *float64 {
				if row.MA20 == nil || row.Close == 0 {
					return nil
				}
				g := (row.Close / *row.MA20) - 1
				return &g
			}(),
		})
	}

	statsRet := make(map[string]map[string]float64)
	for _, h := range horizons {
		avg := 0.0
		winRate := 0.0
		if retCounts[h] > 0 {
			avg = retSums[h] / float64(retCounts[h])
			winRate = float64(retWins[h]) / float64(retCounts[h])
		}
		statsRet[fmt.Sprintf("d%d", h)] = map[string]float64{
			"avg_return": avg,
			"win_rate":   winRate,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"symbol":       symbol,
		"start_date":   startDate.Format("2006-01-02"),
		"end_date":     endDate.Format("2006-01-02"),
		"total_events": len(events),
		"config": map[string]interface{}{
			"weights":    weights,
			"thresholds": thresholds,
			"flags": map[string]bool{
				"use_change": useChange,
				"use_volume": useVolume,
				"use_return": useReturn,
				"use_ma":     useMA,
			},
			"horizons": horizons,
		},
		"events": events,
		"stats": map[string]interface{}{
			"returns": statsRet,
		},
	})
}

func (s *Server) handleSaveBacktestPreset(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	if userID == "" || s.presetStore == nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "preset store not available")
		return
	}
	var body backtestConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid body")
		return
	}
	raw, err := json.Marshal(body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "marshal preset failed")
		return
	}
	if err := s.presetStore.Save(r.Context(), userID, raw); err != nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "save preset failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleGetBacktestPreset(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	if userID == "" || s.presetStore == nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "preset store not available")
		return
	}
	cfgRaw, err := s.presetStore.Load(r.Context(), userID)
	if err != nil {
		if s.presetStore.NotFound(err) {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"message": "preset not found",
			})
			return
		}
		writeError(w, http.StatusInternalServerError, errCodeInternal, "load preset failed")
		return
	}
	var cfg backtestConfig
	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "decode preset failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"preset":  cfg,
	})
}

func (s *Server) handleCreateBacktestPreset(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	if userID == "" || s.presetStore == nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "preset store not available")
		return
	}
	var body struct {
		Name   string         `json:"name"`
		Config backtestConfig `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid body")
		return
	}
	raw, err := json.Marshal(body.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "marshal preset failed")
		return
	}
	id, err := s.presetStore.SaveNamed(r.Context(), userID, body.Name, raw)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "save preset failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"id":      id,
	})
}

func (s *Server) handleListBacktestPresets(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	if userID == "" || s.presetStore == nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "preset store not available")
		return
	}
	rows, err := s.presetStore.List(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "load presets failed")
		return
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		var cfg backtestConfig
		if err := json.Unmarshal(row.Config, &cfg); err != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"id":         row.ID,
			"name":       row.Name,
			"config":     cfg,
			"created_at": row.CreatedAt,
			"updated_at": row.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"items":   items,
	})
}

func (s *Server) handleDeleteBacktestPreset(w http.ResponseWriter, r *http.Request) {
	userID := currentUserID(r)
	if userID == "" || s.presetStore == nil {
		writeError(w, http.StatusInternalServerError, errCodeInternal, "preset store not available")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/analysis/backtest/presets/")
	if id == "" {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "preset id required")
		return
	}
	if err := s.presetStore.Delete(r.Context(), userID, id); err != nil {
		if s.presetStore.NotFound(err) {
			writeError(w, http.StatusNotFound, errCodeNotFound, "preset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, errCodeInternal, "delete preset failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (s *Server) handleStrongStocks(w http.ResponseWriter, r *http.Request) {
	tradeDateStr := r.URL.Query().Get("trade_date")
	if tradeDateStr == "" {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "trade_date required")
		return
	}
	tradeDate, err := time.Parse("2006-01-02", tradeDateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid trade_date")
		return
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	if limit > 200 {
		limit = 200
	}
	scoreMin := parseFloatDefault(r.URL.Query().Get("score_min"), 70)
	volMin := parseFloatDefault(r.URL.Query().Get("volume_ratio_min"), 1.5)

	hasAnalysis, err := s.dataRepo.HasAnalysisForDate(r.Context(), tradeDate)
	if err != nil {
		log.Printf("check analysis ready failed: %v", err)
		writeError(w, http.StatusInternalServerError, errCodeInternal, "internal error")
		return
	}
	if !hasAnalysis {
		writeError(w, http.StatusNotFound, errCodeAnalysisNotReady, "analysis results not ready for trade_date")
		return
	}

	res, err := s.screenerUC.Run(r.Context(), mvp.StrongScreenerInput{
		TradeDate:      tradeDate,
		Limit:          limit,
		ScoreMin:       scoreMin,
		VolumeRatioMin: volMin,
	})
	if err != nil {
		log.Printf("strong stocks screener failed date=%s: %v", tradeDate.Format("2006-01-02"), err)
		writeError(w, http.StatusInternalServerError, errCodeInternal, "internal error")
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
	items := make([]item, 0, len(res.Items))
	for _, r := range res.Items {
		items = append(items, item{
			TradingPair:   r.Symbol,
			MarketType:    string(r.Market),
			ClosePrice:    r.Close,
			ChangePercent: r.ChangeRate,
			Return5d:      r.Return5,
			Volume:        r.Volume,
			VolumeRatio:   r.VolumeMultiple,
			Score:         r.Score,
		})
	}

	log.Printf("strong stocks query done date=%s total=%d returned=%d", tradeDate.Format("2006-01-02"), res.TotalCount, len(items))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"trade_date": tradeDate.Format("2006-01-02"),
		"params": map[string]interface{}{
			"score_min":        scoreMin,
			"volume_ratio_min": volMin,
			"limit":            limit,
		},
		"total_count": res.TotalCount,
		"items":       items,
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

	res, err := s.screenerUC.Run(ctx, mvp.StrongScreenerInput{
		TradeDate:      latestDate,
		Limit:          limit,
		ScoreMin:       scoreMin,
		VolumeRatioMin: volMin,
	})
	if err != nil {
		log.Printf("telegram push: screener error: %v", err)
		return
	}

	var best analysisDomain.DailyAnalysisResult
	if len(res.Items) > 0 {
		best = res.Items[0]
	} else {
		out, err := s.queryUC.QueryByDate(ctx, analysis.QueryByDateInput{
			Date: latestDate,
			Filter: analysis.QueryFilter{
				OnlySuccess: true,
			},
			Pagination: analysis.Pagination{Offset: 0, Limit: 100},
		})
		if err != nil || len(out.Results) == 0 {
			log.Printf("telegram push skipped: no analysis results to report")
			return
		}
		best = out.Results[0]
		for _, r := range out.Results {
			if r.Score > best.Score {
				best = r
			}
		}
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

	if len(res.Items) > 0 {
		builder.WriteString("Top 強勢交易對:\n")
		for i, item := range res.Items {
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
	log.Printf("telegram push sent trade_date=%s items=%d", latestDate.Format("2006-01-02"), len(res.Items))
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("auto pipeline start trade_date=%s", tradeDate.Format("2006-01-02"))
	if err := s.generateDailyPrices(ctx, tradeDate); err != nil {
		log.Printf("auto ingestion error: %v; fallback synthetic", err)
		if fbErr := s.generateSyntheticBTC(ctx, tradeDate); fbErr != nil {
			log.Printf("auto ingestion fallback failed: %v", fbErr)
			return
		}
	}

	prices, err := s.dataRepo.PricesByDate(ctx, tradeDate)
	if err != nil {
		log.Printf("auto analysis: read prices failed: %v", err)
		return
	}
	if len(prices) == 0 {
		log.Printf("auto analysis: no prices for trade_date=%s", tradeDate.Format("2006-01-02"))
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
	log.Printf("auto pipeline done trade_date=%s success=%d total=%d", tradeDate.Format("2006-01-02"), success, len(prices))
}

// --- Helpers ---

func (s *Server) setRefreshCookie(w http.ResponseWriter, r *http.Request, token string, expiry time.Time) {
	// 為了透過 ngrok/https 跨網域攜帶 cookie，強制使用 SameSite=None 且 Secure=true。
	sameSite := http.SameSiteNoneMode
	useHTTPS := true
	if token == "" {
		http.SetCookie(w, &http.Cookie{
			Name:     refreshCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: sameSite,
			Secure:   useHTTPS,
		})
		return
	}
	seconds := int(time.Until(expiry).Seconds())
	if seconds <= 0 {
		seconds = 0
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiry,
		MaxAge:   seconds,
		HttpOnly: true,
		SameSite: sameSite,
		Secure:   useHTTPS,
	})
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

func (s *Server) requireAuth(perm auth.Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := parseBearer(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, http.StatusUnauthorized, errCodeUnauthorized, "missing token")
			return
		}
		claims, err := s.tokenSvc.ParseAccessToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, errCodeUnauthorized, "invalid token")
			return
		}
		user, err := s.authRepo.FindByID(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, errCodeUnauthorized, "invalid token")
			return
		}
		res, err := s.authz.Authorize(r.Context(), auth.AuthorizeInput{
			UserID:   user.ID,
			Required: []auth.Permission{perm},
		})
		if err != nil {
			log.Printf("auth check failed user_id=%s: %v", user.ID, err)
			writeError(w, http.StatusInternalServerError, errCodeInternal, "internal error")
			return
		}
		if !res.Allowed {
			writeError(w, http.StatusForbidden, errCodeForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUserID{}, user.ID)))
	})
}

type ctxKeyUserID struct{}

func (s *Server) wrapGet(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) wrapPost(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) wrapDelete(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) wrapMethods(handlers map[string]http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, ok := handlers[r.Method]
		if !ok {
			writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
			return
		}
		h.ServeHTTP(w, r)
	})
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

func currentUserID(r *http.Request) string {
	if v := r.Context().Value(ctxKeyUserID{}); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
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
	history, _ := s.dataRepo.PricesByPair(ctx, p.Symbol)
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
				lastErr = errors.New("binance response not ok")
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
