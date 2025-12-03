package analysis

import (
	"context"
	"fmt"
	"math"
	"time"

	domain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

// PriceHistoryProvider 取得股票歷史日 K。
type PriceHistoryProvider interface {
	GetHistory(ctx context.Context, symbol string, endDate time.Time, lookback int) ([]dataingestion.DailyPrice, error)
}

// BasicInfoProvider 取得股票基本資料。
type BasicInfoProvider interface {
	ListBasicInfo(ctx context.Context, symbols []string, date time.Time) ([]BasicInfo, error)
}

// AnalysisRepository 儲存分析結果。
type AnalysisRepository interface {
	SaveDailyResult(ctx context.Context, result domain.DailyAnalysisResult) error
}

// BasicInfo 提供分析所需的最低限度基本資料。
type BasicInfo struct {
	Symbol   string
	Market   dataingestion.Market
	Industry string
}

type AnalyzeInput struct {
	TradeDate    time.Time
	Symbols      []string // 若為空則由 BasicInfoProvider 回傳預設清單
	LookbackDays int      // 不含當日，預設 120
	Replace      bool     // 目前保留，未使用；預留重跑覆蓋策略
	Version      string   // 分析版本，可追蹤算法
}

type Failure struct {
	Symbol string
	Reason string
}

type AnalyzeResult struct {
	SuccessCount int
	FailedCount  int
	Failures     []Failure
}

type AnalyzeUseCase struct {
	basicProvider   BasicInfoProvider
	historyProvider PriceHistoryProvider
	repo            AnalysisRepository
}

// NewAnalyzeUseCase 建立日批次分析用例，串接基本資料、歷史價格與儲存介面。
func NewAnalyzeUseCase(basicProvider BasicInfoProvider, historyProvider PriceHistoryProvider, repo AnalysisRepository) *AnalyzeUseCase {
	return &AnalyzeUseCase{
		basicProvider:   basicProvider,
		historyProvider: historyProvider,
		repo:            repo,
	}
}

func (u *AnalyzeUseCase) Execute(ctx context.Context, input AnalyzeInput) (AnalyzeResult, error) {
	var result AnalyzeResult

	if input.TradeDate.IsZero() {
		return result, fmt.Errorf("trade date is required")
	}
	if input.LookbackDays <= 0 {
		input.LookbackDays = 120
	}

	basicList, err := u.basicProvider.ListBasicInfo(ctx, input.Symbols, input.TradeDate)
	if err != nil {
		return result, fmt.Errorf("list basic info: %w", err)
	}

	for _, info := range basicList {
		if info.Symbol == "" {
			result.FailedCount++
			result.Failures = append(result.Failures, Failure{Reason: "missing symbol"})
			continue
		}

		history, err := u.historyProvider.GetHistory(ctx, info.Symbol, input.TradeDate, input.LookbackDays)
		if err != nil {
			result.FailedCount++
			result.Failures = append(result.Failures, Failure{Symbol: info.Symbol, Reason: fmt.Sprintf("history error: %v", err)})
			continue
		}

		analysisRes, err := analyzeOne(info, input.TradeDate, history, input.Version)
		if err != nil {
			result.FailedCount++
			result.Failures = append(result.Failures, Failure{Symbol: info.Symbol, Reason: err.Error()})
			continue
		}

		if err := u.repo.SaveDailyResult(ctx, analysisRes); err != nil {
			result.FailedCount++
			result.Failures = append(result.Failures, Failure{Symbol: info.Symbol, Reason: fmt.Sprintf("store failed: %v", err)})
			continue
		}

		result.SuccessCount++
	}

	return result, nil
}

func analyzeOne(info BasicInfo, tradeDate time.Time, history []dataingestion.DailyPrice, version string) (domain.DailyAnalysisResult, error) {
	var res domain.DailyAnalysisResult

	if len(history) == 0 {
		return res, fmt.Errorf("no history data")
	}

	latest := history[len(history)-1]
	if !sameDate(latest.TradeDate, tradeDate) {
		return res, fmt.Errorf("latest trade date mismatch")
	}

	res = domain.DailyAnalysisResult{
		Symbol:    info.Symbol,
		Market:    info.Market,
		Industry:  info.Industry,
		TradeDate: tradeDate,
		Version:   version,
		Close:     latest.Close,
		Volume:    latest.Volume,
		Success:   true,
	}

	if len(history) >= 2 {
		prev := history[len(history)-2]
		res.Change = latest.Close - prev.Close
		if prev.Close > 0 {
			res.ChangeRate = res.Change / prev.Close
		}
		res.Amplitude = ptr(amplitude(latest, prev.Close))
	}

	res.Return5 = pctReturn(history, 5)
	res.Return20 = pctReturn(history, 20)
	res.Return60 = pctReturn(history, 60)

	res.MA5 = movingAverage(history, 5)
	res.MA10 = movingAverage(history, 10)
	res.MA20 = movingAverage(history, 20)
	res.MA60 = movingAverage(history, 60)

	if res.MA20 != nil && *res.MA20 > 0 {
		res.Deviation20 = ptr((latest.Close - *res.MA20) / *res.MA20)
	}

	res.High20, res.Low20, res.RangePos20 = highLowRange(history, 20, latest.Close)

	res.AvgVolume5 = avgVolume(history, 5)
	res.AvgVolume20 = avgVolume(history, 20)
	if res.AvgVolume20 != nil && *res.AvgVolume20 > 0 {
		res.VolumeMultiple = ptr(float64(latest.Volume) / *res.AvgVolume20)
	}

	res.AvgAmplitude20 = avgAmplitude(history, 20)

	res.Tags = buildTags(res)
	res.Score = buildScore(res)

	if err := res.Validate(); err != nil {
		res.Success = false
		res.ErrorReason = err.Error()
		return res, err
	}

	return res, nil
}

func movingAverage(history []dataingestion.DailyPrice, window int) *float64 {
	if len(history) < window {
		return nil
	}
	sum := 0.0
	for i := len(history) - window; i < len(history); i++ {
		sum += history[i].Close
	}
	avg := sum / float64(window)
	return &avg
}

func pctReturn(history []dataingestion.DailyPrice, window int) *float64 {
	if len(history) < window+1 {
		return nil
	}
	current := history[len(history)-1]
	base := history[len(history)-1-window]
	if base.Close == 0 {
		return nil
	}
	r := (current.Close / base.Close) - 1
	return &r
}

func highLowRange(history []dataingestion.DailyPrice, window int, close float64) (*float64, *float64, *float64) {
	start := len(history) - window
	if start < 0 {
		start = 0
	}
	h := -math.MaxFloat64
	l := math.MaxFloat64
	for i := start; i < len(history); i++ {
		if history[i].High > h {
			h = history[i].High
		}
		if history[i].Low < l {
			l = history[i].Low
		}
	}
	if h == -math.MaxFloat64 || l == math.MaxFloat64 {
		return nil, nil, nil
	}
	rangePos := 0.0
	if h != l {
		rangePos = (close - l) / (h - l)
	}
	return &h, &l, &rangePos
}

func avgVolume(history []dataingestion.DailyPrice, window int) *float64 {
	if len(history) < window {
		return nil
	}
	var sum float64
	for i := len(history) - window; i < len(history); i++ {
		sum += float64(history[i].Volume)
	}
	avg := sum / float64(window)
	return &avg
}

func amplitude(day dataingestion.DailyPrice, prevClose float64) float64 {
	if prevClose <= 0 {
		return 0
	}
	return (day.High - day.Low) / prevClose
}

func avgAmplitude(history []dataingestion.DailyPrice, window int) *float64 {
	if len(history) < window+1 { // 需要前一天收盤價
		return nil
	}
	var sum float64
	for i := len(history) - window; i < len(history); i++ {
		prevClose := history[i-1].Close
		sum += amplitude(history[i], prevClose)
	}
	avg := sum / float64(window)
	return &avg
}

func buildTags(res domain.DailyAnalysisResult) []domain.Tag {
	var tags []domain.Tag

	if res.Return5 != nil && res.RangePos20 != nil && *res.Return5 >= 0.05 && *res.RangePos20 >= 0.8 {
		tags = append(tags, domain.TagShortTermStrong)
	}

	if res.VolumeMultiple != nil && *res.VolumeMultiple >= 1.3 {
		tags = append(tags, domain.TagVolumeSurge)
	}

	if res.RangePos20 != nil && *res.RangePos20 >= 0.8 {
		tags = append(tags, domain.TagNearHigh)
	}
	if res.RangePos20 != nil && *res.RangePos20 <= 0.2 {
		tags = append(tags, domain.TagNearLow)
	}

	if res.AvgAmplitude20 != nil && *res.AvgAmplitude20 >= 0.05 {
		tags = append(tags, domain.TagHighVolatility)
	}
	if res.AvgAmplitude20 != nil && *res.AvgAmplitude20 > 0 && *res.AvgAmplitude20 <= 0.02 {
		tags = append(tags, domain.TagLowVolatility)
	}

	return tags
}

func buildScore(res domain.DailyAnalysisResult) float64 {
	score := 50.0

	if res.Return5 != nil {
		score += clamp(*res.Return5*100, -20, 20) * 0.5
	}
	if res.Return20 != nil {
		score += clamp(*res.Return20*100, -40, 40) * 0.4
	}
	if res.VolumeMultiple != nil {
		score += clamp((*res.VolumeMultiple-1)*10, -10, 15)
	}
	if res.RangePos20 != nil {
		score += (*res.RangePos20 - 0.5) * 10
	}

	return score
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func ptr[T any](v T) *T { return &v }

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
