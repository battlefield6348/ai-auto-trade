package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
)

func (s *Server) runAnalysisForDate(ctx context.Context, tradeDate time.Time) (analysisRunSummary, error) {
	prices, err := s.dataRepo.PricesByDate(ctx, tradeDate)
	if err != nil {
		return analysisRunSummary{}, err
	}
	if len(prices) == 0 {
		return analysisRunSummary{}, errNoPrices
	}

	summary := analysisRunSummary{total: len(prices)}
	for _, p := range prices {
		res := s.calculateAnalysis(ctx, p)
		if err := s.dataRepo.InsertAnalysisResult(ctx, p.Symbol, res); err != nil {
			summary.failure++
		} else {
			summary.success++
		}
	}
	return summary, nil
}

// startAutoPipeline 每隔 autoInterval 自動跑當日 ingestion + analysis。
func (s *Server) startAutoPipeline() {
	ticker := time.NewTicker(s.autoInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.runPipelineOnce()
	}
}

// startConfigBackfill 依組態設定的起始日，啟動一次性補資料與分析（僅補尚未分析的日期）。
func (s *Server) startConfigBackfill() {
	start, err := time.Parse("2006-01-02", s.backfillStart)
	if err != nil {
		log.Printf("[Backfill] Invalid backfill start date: %s", s.backfillStart)
		return
	}
	end := time.Now()

	log.Printf("[Backfill] Scanning from %s to %s", s.backfillStart, end.Format("2006-01-02"))

	curr := start
	for !curr.After(end) {
		exists, _ := s.dataRepo.HasAnalysisForDate(context.Background(), curr)
		if !exists {
			log.Printf("[Backfill] Auto-filling for %s", curr.Format("2006-01-02"))
			// Use strict version for backfill to ensure historical data integrity
			if err := s.generateDailyPricesStrict(context.Background(), curr); err == nil {
				_, _ = s.runAnalysisForDate(context.Background(), curr)
			}
		}
		curr = curr.AddDate(0, 0, 1)
	}
}

func (s *Server) runPipelineOnce() {
	now := time.Now()
	job := jobRun{
		Kind:        "auto",
		TriggeredBy: "system",
		Start:       now,
	}

	if err := s.generateDailyPrices(context.Background(), now); err != nil {
		job.IngestionOK = false
		job.IngestionErr = err.Error()
	} else {
		job.IngestionOK = true
		summary, err := s.runAnalysisForDate(context.Background(), now)
		if err != nil {
			job.AnalysisOK = false
			job.AnalysisErr = err.Error()
		} else {
			job.AnalysisOK = true
			job.AnalysisTotal = summary.total
			job.AnalysisSucc = summary.success
			job.AnalysisFail = summary.failure
		}
	}

	job.End = time.Now()
	s.recordJob(job)
}

// --- Calculation Internal Helpers ---

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
	code := "BTCUSDT"
	market := dataDomain.MarketCrypto
	open, high, low, close := 50000.0, 51000.0, 49000.0, 50500.0
	volume := int64(1000)

	// history (5 days)
	for i := 4; i >= 0; i-- {
		d := tradeDate.AddDate(0, 0, -(i + 1))
		stockID, err := s.dataRepo.UpsertTradingPair(ctx, code, code, market, "")
		if err == nil {
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
