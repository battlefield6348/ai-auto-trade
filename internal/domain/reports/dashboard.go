package reports

import (
	"time"

	alertDomain "ai-auto-trade/internal/domain/alert"
	"ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

// MarketOverview 聚合市場當日摘要。
type MarketOverview struct {
	Date            time.Time
	TotalCount      int
	AverageScore    float64
	ScoreHistogram  map[string]int
	TagCounters     map[analysis.Tag]int
	TopIndustries   []IndustryStat
	StrongestStocks []StockBrief
}

// IndustryStat 描述產業表現。
type IndustryStat struct {
	Industry     string
	Count        int
	AverageScore float64
	AverageRet5  float64
	AverageRet20 float64
}

// StockBrief 提供簡短個股資料。
type StockBrief struct {
	Symbol string
	Score  float64
	Tags   []analysis.Tag
}

// IndustryDashboard 針對單一產業的摘要。
type IndustryDashboard struct {
	Date         time.Time
	Industry     string
	AverageScore float64
	AverageRet5  float64
	AverageRet20 float64
	TopStocks    []StockBrief
	TotalCount   int
}

// StockDashboard 個股歷史摘要。
type StockDashboard struct {
	Symbol       string
	Market       dataingestion.Market
	Industry     string
	History      []analysis.DailyAnalysisResult
	TagsTimeline []TagTimelineEntry
	LastResult   *analysis.DailyAnalysisResult
}

// TagTimelineEntry 記錄某日出現的標籤。
type TagTimelineEntry struct {
	Date time.Time
	Tags []analysis.Tag
}

// StrategyPerformance 策略績效摘要。
type StrategyPerformance struct {
	Date       time.Time
	Strategies []StrategySummary
}

// StrategySummary 為單一策略的統計。
type StrategySummary struct {
	ID             string
	Name           string
	Triggered      int
	TriggeredToday bool
}

// SystemHealthDashboard 系統健康摘要。
type SystemHealthDashboard struct {
	Date    time.Time
	Metrics []alertDomain.SystemMetric
}
