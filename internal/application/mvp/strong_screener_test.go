package mvp

import (
	"context"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

type fakeQueryRepoMVP struct {
	results []analysisDomain.DailyAnalysisResult
}

func (f fakeQueryRepoMVP) FindByDate(_ context.Context, date time.Time, _ analysis.QueryFilter, _ analysis.SortOption, _ analysis.Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	var filtered []analysisDomain.DailyAnalysisResult
	for _, r := range f.results {
		if sameDate(r.TradeDate, date) {
			filtered = append(filtered, r)
		}
	}
	return filtered, len(filtered), nil
}

func (f fakeQueryRepoMVP) FindHistory(context.Context, string, *time.Time, *time.Time, int, bool) ([]analysisDomain.DailyAnalysisResult, error) {
	return nil, nil
}

func (f fakeQueryRepoMVP) Get(context.Context, string, time.Time) (analysisDomain.DailyAnalysisResult, error) {
	return analysisDomain.DailyAnalysisResult{}, nil
}

func TestStrongScreener_Run(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	repo := fakeQueryRepoMVP{
		results: []analysisDomain.DailyAnalysisResult{
			{
				Symbol:         "2330",
				Market:         dataingestion.MarketTWSE,
				TradeDate:      date,
				Score:          80,
				Return5:        ptr(0.05),
				VolumeMultiple: ptr(1.6),
				ChangeRate:     0.01,
			},
			{
				Symbol:         "2317",
				Market:         dataingestion.MarketTWSE,
				TradeDate:      date,
				Score:          60,
				Return5:        ptr(0.02),
				VolumeMultiple: ptr(1.8),
				ChangeRate:     0.02,
			},
		},
	}

	screener := NewStrongScreener(repo)
	out, err := screener.Run(context.Background(), StrongScreenerInput{TradeDate: date})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.TotalCount != 1 {
		t.Fatalf("expected 1 strong stock, got %d", out.TotalCount)
	}
	if out.Items[0].Symbol != "2330" {
		t.Fatalf("expected 2330, got %s", out.Items[0].Symbol)
	}
}

func ptr(v float64) *float64 { return &v }
