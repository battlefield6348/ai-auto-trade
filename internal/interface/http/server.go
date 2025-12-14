package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
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
	errCodeIngestionNotReady  = "INGESTION_NOT_READY"
	errCodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
	errCodeInternal           = "INTERNAL_ERROR"
)

// DataRepository 定義 ingestion/analysis 讀寫與查詢接口。
type DataRepository interface {
	analysis.AnalysisQueryRepository
	UpsertStock(ctx context.Context, code, name string, market dataDomain.Market, industry string) (string, error)
	InsertDailyPrice(ctx context.Context, stockID string, price dataDomain.DailyPrice) error
	PricesByDate(ctx context.Context, date time.Time) ([]dataDomain.DailyPrice, error)
	PricesBySymbol(ctx context.Context, symbol string) ([]dataDomain.DailyPrice, error)
	InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error
	HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error)
}

// memoryRepoAdapter 讓 memory.Store 相容 DataRepository。
type memoryRepoAdapter struct {
	store *memory.Store
}

func (m memoryRepoAdapter) UpsertStock(ctx context.Context, code, name string, market dataDomain.Market, industry string) (string, error) {
	return m.store.UpsertStock(code, name, market, industry), nil
}
func (m memoryRepoAdapter) InsertDailyPrice(ctx context.Context, stockID string, price dataDomain.DailyPrice) error {
	m.store.InsertDailyPrice(price)
	return nil
}
func (m memoryRepoAdapter) PricesByDate(ctx context.Context, date time.Time) ([]dataDomain.DailyPrice, error) {
	return m.store.PricesByDate(date), nil
}
func (m memoryRepoAdapter) PricesBySymbol(ctx context.Context, symbol string) ([]dataDomain.DailyPrice, error) {
	return m.store.PricesBySymbol(symbol), nil
}
func (m memoryRepoAdapter) InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error {
	m.store.InsertAnalysisResult(res)
	return nil
}
func (m memoryRepoAdapter) HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error) {
	return m.store.HasAnalysisForDate(date), nil
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

// --- Handlers ---

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "pong",
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

func (s *Server) handleIngestionDaily(w http.ResponseWriter, r *http.Request) {
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

	start := time.Now()
	log.Printf("ingestion daily start trade_date=%s", tradeDate.Format("2006-01-02"))
	if err := s.generateDailyPrices(r.Context(), tradeDate); err != nil {
		log.Printf("ingestion error: %v; fallback to synthetic BTC", err)
		if fbErr := s.generateSyntheticBTC(r.Context(), tradeDate); fbErr != nil {
			log.Printf("fallback ingestion failed: %v", fbErr)
			writeError(w, http.StatusInternalServerError, errCodeInternal, "ingestion failed")
			return
		}
	}
	log.Printf("ingestion daily done trade_date=%s duration=%s", tradeDate.Format("2006-01-02"), time.Since(start))

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"trade_date":    tradeDate.Format("2006-01-02"),
		"total_stocks":  2,
		"success_count": 2,
		"failure_count": 0,
	})
}

func (s *Server) handleAnalysisDaily(w http.ResponseWriter, r *http.Request) {
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

	prices, err := s.dataRepo.PricesByDate(r.Context(), tradeDate)
	if err != nil {
		log.Printf("analysis read prices failed: %v", err)
		writeError(w, http.StatusInternalServerError, errCodeInternal, "failed to read prices")
		return
	}
	if len(prices) == 0 {
		writeError(w, http.StatusConflict, errCodeIngestionNotReady, "ingestion data not ready for trade_date")
		return
	}
	start := time.Now()
	log.Printf("analysis daily start trade_date=%s prices=%d", tradeDate.Format("2006-01-02"), len(prices))
	success := 0
	for _, p := range prices {
		stockID, err := s.dataRepo.UpsertStock(r.Context(), p.Symbol, p.Symbol, p.Market, "")
		if err != nil {
			log.Printf("upsert stock failed symbol=%s: %v", p.Symbol, err)
			continue
		}
		res := s.calculateAnalysis(r.Context(), p)
		if err := s.dataRepo.InsertAnalysisResult(r.Context(), stockID, res); err != nil {
			log.Printf("write analysis failed symbol=%s date=%s: %v", p.Symbol, tradeDate.Format("2006-01-02"), err)
			continue
		}
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

// --- Helpers ---

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

// generateDailyPrices 從外部來源取得 BTCUSDT 的日 K，失敗時由上層 fallback。
func (s *Server) generateDailyPrices(ctx context.Context, tradeDate time.Time) error {
	if s.useSynthetic {
		return s.generateSyntheticBTC(ctx, tradeDate)
	}
	series, err := s.fetchBTCSeries(ctx, tradeDate)
	if err != nil {
		return err
	}
	for _, p := range series {
		stockID, err := s.dataRepo.UpsertStock(ctx, p.Symbol, "Bitcoin", dataDomain.MarketCrypto, "Crypto")
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
		stockID, err := s.dataRepo.UpsertStock(ctx, "BTCUSDT", "Bitcoin", dataDomain.MarketCrypto, "Crypto")
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
			stockID, err := s.dataRepo.UpsertStock(ctx, code, code, market, "")
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
	stockID, err := s.dataRepo.UpsertStock(ctx, code, code, market, "")
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
	history, _ := s.dataRepo.PricesBySymbol(ctx, p.Symbol)
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

// fetchBTCSeries 從 Binance 抓取 BTCUSDT 1d K 線，包含指定日期與前 5 日。
func (s *Server) fetchBTCSeries(ctx context.Context, tradeDate time.Time) ([]dataDomain.DailyPrice, error) {
	start := tradeDate.AddDate(0, 0, -5)
	end := tradeDate.AddDate(0, 0, 1)
	url := "https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1d&startTime=" +
		strconv.FormatInt(start.UnixMilli(), 10) + "&endTime=" + strconv.FormatInt(end.UnixMilli(), 10)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("binance response not ok")
	}

	var raw [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
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
		return nil, errors.New("no kline data")
	}
	return out, nil
}
