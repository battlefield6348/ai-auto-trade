package reports

import (
	"context"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/strategy"
	alertDomain "ai-auto-trade/internal/domain/alert"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	reportsDomain "ai-auto-trade/internal/domain/reports"
)

// AnalysisReader 提供分析結果查詢。
type AnalysisReader interface {
	QueryByDate(ctx context.Context, input analysis.QueryByDateInput) (analysis.QueryByDateOutput, error)
	QueryHistory(ctx context.Context, input analysis.QueryHistoryInput) ([]analysisDomain.DailyAnalysisResult, error)
}

// StrategyRunReader 讀取策略執行紀錄。
type StrategyRunReader interface {
	ListRuns(ctx context.Context, date time.Time) ([]strategy.RunRecord, error)
}

// HealthReader 讀取系統健康度資料。
type HealthReader interface {
	Check(ctx context.Context, date time.Time) ([]alertDomain.SystemMetric, error)
}

// UseCase 聚合儀表板與報表邏輯。
type UseCase struct {
	analysis AnalysisReader
	strategy StrategyRunReader
	health   HealthReader
	now      func() time.Time
}

// NewUseCase 建立報表與儀表板用例，匯總分析、策略與健康度資料。
func NewUseCase(analysis AnalysisReader, strategy StrategyRunReader, health HealthReader) *UseCase {
	return &UseCase{
		analysis: analysis,
		strategy: strategy,
		health:   health,
		now:      time.Now,
	}
}

// BuildMarketOverview 產出市場總覽。
func (u *UseCase) BuildMarketOverview(ctx context.Context, date time.Time) (reportsDomain.MarketOverview, error) {
	out := reportsDomain.MarketOverview{Date: date}
	resp, err := u.analysis.QueryByDate(ctx, analysis.QueryByDateInput{
		Date: date,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
		},
		Pagination: analysis.Pagination{Offset: 0, Limit: 10000},
	})
	if err != nil {
		return out, err
	}
	out.TotalCount = len(resp.Results)
	if out.TotalCount == 0 {
		return out, nil
	}

	tagCount := make(map[analysisDomain.Tag]int)
	scoreSum := 0.0
	for _, r := range resp.Results {
		scoreSum += r.Score
		for _, t := range r.Tags {
			tagCount[t]++
		}
	}
	out.AverageScore = scoreSum / float64(out.TotalCount)
	out.TagCounters = tagCount
	out.ScoreHistogram = buildScoreHistogram(resp.Results)
	out.TopIndustries = topIndustries(resp.Results, 3)
	out.StrongestStocks = topStocks(resp.Results, 5)

	return out, nil
}

// BuildIndustryDashboard 產出單一產業摘要。
func (u *UseCase) BuildIndustryDashboard(ctx context.Context, date time.Time, industry string) (reportsDomain.IndustryDashboard, error) {
	out := reportsDomain.IndustryDashboard{Date: date, Industry: industry}
	resp, err := u.analysis.QueryByDate(ctx, analysis.QueryByDateInput{
		Date: date,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
			Industries:  []string{industry},
		},
		Pagination: analysis.Pagination{Offset: 0, Limit: 5000},
	})
	if err != nil {
		return out, err
	}
	if len(resp.Results) == 0 {
		return out, nil
	}

	out.TotalCount = len(resp.Results)
	sumScore, sumRet5, sumRet20 := 0.0, 0.0, 0.0
	countRet5, countRet20 := 0, 0
	for _, r := range resp.Results {
		sumScore += r.Score
		if r.Return5 != nil {
			sumRet5 += *r.Return5
			countRet5++
		}
		if r.Return20 != nil {
			sumRet20 += *r.Return20
			countRet20++
		}
	}
	out.AverageScore = sumScore / float64(out.TotalCount)
	if countRet5 > 0 {
		out.AverageRet5 = sumRet5 / float64(countRet5)
	}
	if countRet20 > 0 {
		out.AverageRet20 = sumRet20 / float64(countRet20)
	}
	out.TopStocks = topStocks(resp.Results, 10)
	return out, nil
}

// BuildStockDashboard 產出個股摘要。
func (u *UseCase) BuildStockDashboard(ctx context.Context, symbol string, from, to *time.Time) (reportsDomain.StockDashboard, error) {
	out := reportsDomain.StockDashboard{Symbol: symbol}
	if symbol == "" {
		return out, fmt.Errorf("symbol is required")
	}
	resp, err := u.analysis.QueryHistory(ctx, analysis.QueryHistoryInput{
		Symbol: symbol,
		From:   from,
		To:     to,
		Limit:  365,
	})
	if err != nil {
		return out, err
	}
	if len(resp) == 0 {
		return out, nil
	}
	out.History = resp
	last := resp[len(resp)-1]
	out.LastResult = &last
	out.Market = last.Market
	out.Industry = last.Industry
	out.TagsTimeline = buildTagTimeline(resp)
	return out, nil
}

// BuildStrategyPerformance 根據策略執行紀錄產出摘要。
func (u *UseCase) BuildStrategyPerformance(ctx context.Context, date time.Time) (reportsDomain.StrategyPerformance, error) {
	out := reportsDomain.StrategyPerformance{Date: date}
	if u.strategy == nil {
		return out, nil
	}
	runs, err := u.strategy.ListRuns(ctx, date)
	if err != nil {
		return out, err
	}
	if len(runs) == 0 {
		return out, nil
	}

	m := make(map[string]*reportsDomain.StrategySummary)
	for _, r := range runs {
		sum := m[r.StrategyID]
		if sum == nil {
			sum = &reportsDomain.StrategySummary{ID: r.StrategyID}
			m[r.StrategyID] = sum
		}
		if r.Triggered {
			sum.Triggered++
			if sameDate(r.Date, date) {
				sum.TriggeredToday = true
			}
		}
	}

	for _, v := range m {
		out.Strategies = append(out.Strategies, *v)
	}
	sort.Slice(out.Strategies, func(i, j int) bool {
		return out.Strategies[i].Triggered > out.Strategies[j].Triggered
	})
	return out, nil
}

// BuildSystemHealth 產出系統健康摘要。
func (u *UseCase) BuildSystemHealth(ctx context.Context, date time.Time) (reportsDomain.SystemHealthDashboard, error) {
	out := reportsDomain.SystemHealthDashboard{Date: date}
	if u.health == nil {
		return out, nil
	}
	metrics, err := u.health.Check(ctx, date)
	if err != nil {
		return out, err
	}
	out.Metrics = metrics
	return out, nil
}

// ExportDailyMarketReport 匯出當日市場報告 CSV。
func (u *UseCase) ExportDailyMarketReport(ctx context.Context, date time.Time, limit int) (string, error) {
	if limit <= 0 {
		limit = 200
	}
	resp, err := u.analysis.QueryByDate(ctx, analysis.QueryByDateInput{
		Date: date,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
		},
		Sort:       analysis.SortOption{Field: analysis.SortScore, Desc: true},
		Pagination: analysis.Pagination{Offset: 0, Limit: limit},
	})
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	w := csv.NewWriter(&sb)
	header := []string{"date", "symbol", "market", "industry", "close", "score", "return5", "return20", "volume_multiple", "tags"}
	if err := w.Write(header); err != nil {
		return "", err
	}
	for _, r := range resp.Results {
		record := []string{
			r.TradeDate.Format("2006-01-02"),
			r.Symbol,
			string(r.Market),
			r.Industry,
			formatFloat(r.Close),
			formatFloat(r.Score),
			formatPtr(r.Return5),
			formatPtr(r.Return20),
			formatPtr(r.VolumeMultiple),
			strings.Join(tagStrings(r.Tags), "|"),
		}
		if err := w.Write(record); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// helpers
func buildScoreHistogram(results []analysisDomain.DailyAnalysisResult) map[string]int {
	buckets := map[string]int{
		"0-20":   0,
		"20-40":  0,
		"40-60":  0,
		"60-80":  0,
		"80-100": 0,
	}
	for _, r := range results {
		switch {
		case r.Score < 20:
			buckets["0-20"]++
		case r.Score < 40:
			buckets["20-40"]++
		case r.Score < 60:
			buckets["40-60"]++
		case r.Score < 80:
			buckets["60-80"]++
		default:
			buckets["80-100"]++
		}
	}
	return buckets
}

func topIndustries(results []analysisDomain.DailyAnalysisResult, n int) []reportsDomain.IndustryStat {
	type agg struct {
		count      int
		score      float64
		ret5       float64
		ret5Count  int
		ret20      float64
		ret20Count int
	}
	stats := make(map[string]*agg)
	for _, r := range results {
		key := r.Industry
		if key == "" {
			key = "UNKNOWN"
		}
		a := stats[key]
		if a == nil {
			a = &agg{}
			stats[key] = a
		}
		a.count++
		a.score += r.Score
		if r.Return5 != nil {
			a.ret5 += *r.Return5
			a.ret5Count++
		}
		if r.Return20 != nil {
			a.ret20 += *r.Return20
			a.ret20Count++
		}
	}

	var list []reportsDomain.IndustryStat
	for ind, a := range stats {
		stat := reportsDomain.IndustryStat{
			Industry:     ind,
			Count:        a.count,
			AverageScore: a.score / float64(a.count),
		}
		if a.ret5Count > 0 {
			stat.AverageRet5 = a.ret5 / float64(a.ret5Count)
		}
		if a.ret20Count > 0 {
			stat.AverageRet20 = a.ret20 / float64(a.ret20Count)
		}
		list = append(list, stat)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].AverageScore > list[j].AverageScore
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}

func topStocks(results []analysisDomain.DailyAnalysisResult, n int) []reportsDomain.StockBrief {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > n {
		results = results[:n]
	}
	out := make([]reportsDomain.StockBrief, 0, len(results))
	for _, r := range results {
		out = append(out, reportsDomain.StockBrief{
			Symbol: r.Symbol,
			Score:  r.Score,
			Tags:   r.Tags,
		})
	}
	return out
}

func buildTagTimeline(history []analysisDomain.DailyAnalysisResult) []reportsDomain.TagTimelineEntry {
	timeline := make([]reportsDomain.TagTimelineEntry, 0, len(history))
	for _, r := range history {
		if len(r.Tags) == 0 {
			continue
		}
		timeline = append(timeline, reportsDomain.TagTimelineEntry{
			Date: r.TradeDate,
			Tags: r.Tags,
		})
	}
	return timeline
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}

func formatPtr(v *float64) string {
	if v == nil {
		return ""
	}
	return formatFloat(*v)
}

func tagStrings(tags []analysisDomain.Tag) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = string(t)
	}
	sort.Strings(out)
	return out
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
