package reports

import (
	"context"
	"strings"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/strategy"
	alertDomain "ai-auto-trade/internal/domain/alert"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

type fakeAnalysisReader struct {
	byDate  []analysisDomain.DailyAnalysisResult
	history map[string][]analysisDomain.DailyAnalysisResult
}

func (f fakeAnalysisReader) QueryByDate(_ context.Context, _ analysis.QueryByDateInput) (analysis.QueryByDateOutput, error) {
	return analysis.QueryByDateOutput{Results: f.byDate, Total: len(f.byDate)}, nil
}

func (f fakeAnalysisReader) QueryHistory(_ context.Context, input analysis.QueryHistoryInput) ([]analysisDomain.DailyAnalysisResult, error) {
	return f.history[input.Symbol], nil
}

type fakeStrategyRunReader struct {
	runs []strategy.RunRecord
}

func (f fakeStrategyRunReader) ListRuns(_ context.Context, _ time.Time) ([]strategy.RunRecord, error) {
	return f.runs, nil
}

type fakeHealthReader struct {
	metrics []alertDomain.SystemMetric
}

func (f fakeHealthReader) Check(_ context.Context, _ time.Time) ([]alertDomain.SystemMetric, error) {
	return f.metrics, nil
}

func TestBuildMarketOverview(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	reader := fakeAnalysisReader{
		byDate: []analysisDomain.DailyAnalysisResult{
			{Symbol: "2330", Industry: "半導體", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 80, Tags: []analysisDomain.Tag{analysisDomain.TagShortTermStrong}},
			{Symbol: "2317", Industry: "電子", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 60, Tags: []analysisDomain.Tag{analysisDomain.TagVolumeSurge}},
		},
	}
	uc := NewUseCase(reader, nil, nil)
	out, err := uc.BuildMarketOverview(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.TotalCount != 2 {
		t.Fatalf("unexpected total: %d", out.TotalCount)
	}
	if out.TagCounters[analysisDomain.TagShortTermStrong] != 1 {
		t.Fatalf("expected tag counter")
	}
	if len(out.TopIndustries) == 0 {
		t.Fatalf("expected top industries")
	}
	if len(out.ScoreHistogram) == 0 {
		t.Fatalf("expected histogram")
	}
}

func TestBuildIndustryDashboard(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	reader := fakeAnalysisReader{
		byDate: []analysisDomain.DailyAnalysisResult{
			{Symbol: "2330", Industry: "半導體", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 80, Return5: ptr(0.05), Return20: ptr(0.1)},
			{Symbol: "2303", Industry: "半導體", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 70, Return5: ptr(0.02)},
		},
	}
	uc := NewUseCase(reader, nil, nil)
	out, err := uc.BuildIndustryDashboard(context.Background(), date, "半導體")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.TotalCount != 2 || out.AverageScore <= 0 {
		t.Fatalf("unexpected industry dashboard: %+v", out)
	}
	if len(out.TopStocks) == 0 {
		t.Fatalf("expected top stocks")
	}
}

func TestBuildStockDashboard(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	reader := fakeAnalysisReader{
		history: map[string][]analysisDomain.DailyAnalysisResult{
			"2330": {
				{Symbol: "2330", Market: dataingestion.MarketTWSE, Industry: "半導體", TradeDate: date.AddDate(0, 0, -1), Score: 70, Tags: []analysisDomain.Tag{analysisDomain.TagVolumeSurge}},
				{Symbol: "2330", Market: dataingestion.MarketTWSE, Industry: "半導體", TradeDate: date, Score: 80, Tags: []analysisDomain.Tag{analysisDomain.TagShortTermStrong}},
			},
		},
	}
	uc := NewUseCase(reader, nil, nil)
	out, err := uc.BuildStockDashboard(context.Background(), "2330", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.LastResult == nil || out.LastResult.Score != 80 {
		t.Fatalf("expected last result score 80")
	}
	if len(out.TagsTimeline) != 2 {
		t.Fatalf("expected tag timeline entries")
	}
}

func TestBuildStrategyPerformance(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	runReader := fakeStrategyRunReader{
		runs: []strategy.RunRecord{
			{StrategyID: "s1", Date: date, Triggered: true},
			{StrategyID: "s1", Date: date.AddDate(0, 0, -1), Triggered: true},
		},
	}
	uc := NewUseCase(fakeAnalysisReader{}, runReader, nil)
	out, err := uc.BuildStrategyPerformance(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Strategies) != 1 || out.Strategies[0].Triggered != 2 || !out.Strategies[0].TriggeredToday {
		t.Fatalf("unexpected strategy summary: %+v", out.Strategies)
	}
}

func TestBuildSystemHealth(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	health := fakeHealthReader{metrics: []alertDomain.SystemMetric{{Metric: "ingestion_fail", Value: 0.1}}}
	uc := NewUseCase(fakeAnalysisReader{}, nil, health)
	out, err := uc.BuildSystemHealth(context.Background(), date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Metrics) != 1 {
		t.Fatalf("expected metrics")
	}
}

func TestExportDailyMarketReport(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	reader := fakeAnalysisReader{
		byDate: []analysisDomain.DailyAnalysisResult{
			{Symbol: "2330", Market: dataingestion.MarketTWSE, Industry: "半導體", TradeDate: date, Score: 80, Close: 600, Return5: ptr(0.05)},
		},
	}
	uc := NewUseCase(reader, nil, nil)
	csvStr, err := uc.ExportDailyMarketReport(context.Background(), date, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(csvStr), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "2330") {
		t.Fatalf("unexpected csv content: %s", lines[1])
	}
}

func ptr(v float64) *float64 { return &v }
