package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"database/sql"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/auth"
	"ai-auto-trade/internal/application/mvp"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	"ai-auto-trade/internal/infra/memory"
	"ai-auto-trade/internal/infrastructure/config"
)

type server struct {
	db         *sql.DB
	store      *memory.Store
	loginUC    *auth.LoginUseCase
	authz      *auth.Authorizer
	queryUC    *analysis.QueryUseCase
	screenerUC *mvp.StrongScreener
	tokenTTL   time.Duration
}

const (
	errCodeBadRequest         = "BAD_REQUEST"
	errCodeInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	errCodeUnauthorized       = "AUTH_UNAUTHORIZED"
	errCodeForbidden          = "AUTH_FORBIDDEN"
	errCodeAnalysisNotReady   = "ANALYSIS_NOT_READY"
	errCodeIngestionNotReady  = "INGESTION_NOT_READY"
	errCodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
	errCodeInternal           = "INTERNAL_ERROR"
)

func newServer(cfg config.Config, dbPool *sql.DB) *server {
	store := memory.NewStore()
	store.SeedUsers()

	ttl := cfg.Auth.TokenTTL
	if ttl == 0 {
		ttl = 30 * time.Minute
	}
	tokenIssuer := memory.NewMemoryTokenIssuer(store, ttl)
	loginUC := auth.NewLoginUseCase(store, memory.PlainHasher{}, tokenIssuer)
	authz := auth.NewAuthorizer(store, memory.OwnerChecker{})
	queryUC := analysis.NewQueryUseCase(store)
	screenerUC := mvp.NewStrongScreener(store)

	return &server{
		db:         dbPool,
		store:      store,
		loginUC:    loginUC,
		authz:      authz,
		queryUC:    queryUC,
		screenerUC: screenerUC,
		tokenTTL:   ttl,
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})
	// 前端靜態頁面：提供簡易操作介面。
	mux.Handle("/", http.FileServer(http.Dir("web")))

	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.Handle("/api/admin/ingestion/daily", s.withAuth(auth.PermIngestionTriggerDaily, s.handleIngestionDaily))
	mux.Handle("/api/admin/analysis/daily", s.withAuth(auth.PermAnalysisTriggerDaily, s.handleAnalysisDaily))
	mux.Handle("/api/analysis/daily", s.withAuth(auth.PermAnalysisQuery, s.handleAnalysisQuery))
	mux.Handle("/api/screener/strong-stocks", s.withAuth(auth.PermScreenerUse, s.handleStrongStocks))
	return mux
}

// --- Handlers ---

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid body")
		return
	}
	res, err := s.loginUC.Execute(r.Context(), auth.LoginInput{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		log.Printf("login failed email=%s: %v", body.Email, err)
		writeError(w, http.StatusUnauthorized, errCodeInvalidCredentials, "invalid credentials")
		return
	}
	log.Printf("login success user_id=%s role=%s email=%s", res.User.ID, res.User.Role, res.User.Email)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"access_token": res.Token.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   int(s.tokenTTL.Seconds()),
	})
}

func (s *server) handleIngestionDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		TradeDate string `json:"trade_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid body")
		return
	}
	tradeDate, err := time.Parse("2006-01-02", body.TradeDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid trade_date")
		return
	}

	// Seed sample stocks and generate synthetic prices for MVP.
	start := time.Now()
	log.Printf("ingestion daily start trade_date=%s", tradeDate.Format("2006-01-02"))
	s.generateDailyPrices(tradeDate)
	log.Printf("ingestion daily done trade_date=%s duration=%s", tradeDate.Format("2006-01-02"), time.Since(start))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"trade_date":    tradeDate.Format("2006-01-02"),
		"total_stocks":  2,
		"success_count": 2,
		"failure_count": 0,
	})
}

func (s *server) handleAnalysisDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		TradeDate string `json:"trade_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid body")
		return
	}
	tradeDate, err := time.Parse("2006-01-02", body.TradeDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, errCodeBadRequest, "invalid trade_date")
		return
	}

	prices := s.store.PricesByDate(tradeDate)
	if len(prices) == 0 {
		writeError(w, http.StatusConflict, errCodeIngestionNotReady, "ingestion data not ready for trade_date")
		return
	}
	start := time.Now()
	log.Printf("analysis daily start trade_date=%s prices=%d", tradeDate.Format("2006-01-02"), len(prices))
	success := 0
	for _, p := range prices {
		res := s.calculateAnalysis(p)
		s.store.InsertAnalysisResult(res)
		success++
	}
	log.Printf("analysis daily done trade_date=%s success=%d duration=%s", tradeDate.Format("2006-01-02"), success, time.Since(start))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"trade_date":    tradeDate.Format("2006-01-02"),
		"total_stocks":  len(prices),
		"success_count": success,
		"failure_count": 0,
	})
}

func (s *server) handleAnalysisQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
		return
	}
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
		StockCode     string   `json:"stock_code"`
		StockName     string   `json:"stock_name"`
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
			StockCode:     r.Symbol,
			StockName:     r.Industry, // placeholder
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

func (s *server) handleStrongStocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, errCodeMethodNotAllowed, "method not allowed")
		return
	}
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

	if !s.store.HasAnalysisForDate(tradeDate) {
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
		StockCode     string   `json:"stock_code"`
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
			StockCode:     r.Symbol,
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

// --- Helpers ---

func (s *server) withAuth(perm auth.Permission, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := parseBearer(r.Header.Get("Authorization"))
		if token == "" {
			writeError(w, http.StatusUnauthorized, errCodeUnauthorized, "missing token")
			return
		}
		user, ok := s.store.ValidateToken(token)
		if !ok {
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
		// stash user in context? for simplicity, skip.
		next.ServeHTTP(w, r)
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

type errorResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	ErrorCode string `json:"error_code,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorResponse{
		Success:   false,
		Error:     msg,
		ErrorCode: code,
	})
}

func writeJSON(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// generateDailyPrices seeds two stocks and daily prices plus simple history for return calculation.
func (s *server) generateDailyPrices(tradeDate time.Time) {
	s1 := s.store.UpsertStock("2330", "TSMC", dataDomain.MarketTWSE, "半導體")
	s2 := s.store.UpsertStock("2317", "HonHai", dataDomain.MarketTWSE, "電子")
	_ = s1
	_ = s2

	s.insertPriceSeries("2330", dataDomain.MarketTWSE, tradeDate, 600, 610, 595, 608, 1_000_000)
	s.insertPriceSeries("2317", dataDomain.MarketTWSE, tradeDate, 100, 105, 99, 104, 2_000_000)
}

func (s *server) insertPriceSeries(code string, market dataDomain.Market, tradeDate time.Time, open, high, low, close float64, volume int64) {
	// generate last 5 days synthetic if missing
	for i := 5; i >= 1; i-- {
		d := tradeDate.AddDate(0, 0, -i)
		if len(s.store.PricesByDate(d)) == 0 {
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
			s.store.InsertDailyPrice(price)
		}
	}
	// today
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
	s.store.InsertDailyPrice(price)
}

func (s *server) calculateAnalysis(p dataDomain.DailyPrice) analysisDomain.DailyAnalysisResult {
	history := s.store.PricesBySymbol(p.Symbol)
	var return5 *float64
	var volumeRatio *float64
	var changeRate float64
	// find previous close
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].TradeDate.Equal(p.TradeDate) && i > 0 {
			prev := history[i-1]
			if prev.Close > 0 {
				changeRate = (p.Close - prev.Close) / prev.Close
			}
			break
		}
	}
	if len(history) >= 6 {
		// current + previous 5
		idx := len(history) - 1
		earlier := history[idx-5]
		if earlier.Close > 0 {
			val := (p.Close / earlier.Close) - 1
			return5 = &val
		}
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
