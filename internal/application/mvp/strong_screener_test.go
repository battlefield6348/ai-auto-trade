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
		if r.TradeDate.Equal(date) {
			filtered = append(filtered, r)
		}
	}
	return filtered, len(filtered), nil
}

func (f fakeQueryRepoMVP) FindHistory(context.Context, string, string, *time.Time, *time.Time, int, bool) ([]analysisDomain.DailyAnalysisResult, error) {
	return nil, nil
}

func (f fakeQueryRepoMVP) Get(context.Context, string, time.Time, string) (analysisDomain.DailyAnalysisResult, error) {
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

	t.Run("Success", func(t *testing.T) {
		out, err := screener.Run(context.Background(), StrongScreenerInput{TradeDate: date})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.TotalCount != 1 {
			t.Errorf("expected 1 strong stock, got %d", out.TotalCount)
		}
	})

	t.Run("Empty Results", func(t *testing.T) {
		wrongDate := date.AddDate(0, 0, 1)
		out, _ := screener.Run(context.Background(), StrongScreenerInput{TradeDate: wrongDate})
		if out.TotalCount != 0 {
			t.Errorf("expected 0, got %d", out.TotalCount)
		}
	})

	t.Run("Zero Date", func(t *testing.T) {
		out, _ := screener.Run(context.Background(), StrongScreenerInput{})
		if out.TotalCount != 0 {
			t.Errorf("expected 0 for zero date")
		}
	})

	t.Run("Limit and Sort", func(t *testing.T) {
		repoLarge := fakeQueryRepoMVP{
			results: []analysisDomain.DailyAnalysisResult{
				{Symbol: "S1", TradeDate: date, Score: 80, Return5: ptr(0.05), VolumeMultiple: ptr(2.0)},
				{Symbol: "S2", TradeDate: date, Score: 80, Return5: ptr(0.10), VolumeMultiple: ptr(2.0)},
				{Symbol: "S3", TradeDate: date, Score: 90, Return5: ptr(0.01), VolumeMultiple: ptr(2.0)},
			},
		}
		screener2 := NewStrongScreener(repoLarge)
		out, _ := screener2.Run(context.Background(), StrongScreenerInput{TradeDate: date, Limit: 2})
		
		if len(out.Items) != 2 {
			t.Errorf("expected limit 2, got %d", len(out.Items))
		}
		// S3 has highest score (90), then S2 (80, return 0.10), then S1 (80, return 0.05)
		if out.Items[0].Symbol != "S3" || out.Items[1].Symbol != "S2" {
			t.Errorf("Sort order failed: %+v", out.Items)
		}
	})
}

func ptr(v float64) *float64 { return &v }
