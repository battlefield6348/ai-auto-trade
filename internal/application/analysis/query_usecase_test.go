package analysis

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

type fakeQueryRepo struct {
	byDate    []analysisDomain.DailyAnalysisResult
	history   map[string][]analysisDomain.DailyAnalysisResult
	detailErr error
}

func (f fakeQueryRepo) FindByDate(_ context.Context, date time.Time, filter QueryFilter, _ SortOption, pagination Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	var filtered []analysisDomain.DailyAnalysisResult
	for _, r := range f.byDate {
		if !sameDate(r.TradeDate, date) {
			continue
		}
		if filter.OnlySuccess && !r.Success {
			continue
		}
		if len(filter.Markets) > 0 && !containsMarket(filter.Markets, r.Market) {
			continue
		}
		if len(filter.Symbols) > 0 && !containsString(filter.Symbols, r.Symbol) {
			continue
		}
		filtered = append(filtered, r)
	}

	total := len(filtered)
	start := pagination.Offset
	if start > total {
		start = total
	}
	end := start + pagination.Limit
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (f fakeQueryRepo) FindHistory(_ context.Context, symbol string, _ string, _ *time.Time, _ *time.Time, limit int, _ bool) ([]analysisDomain.DailyAnalysisResult, error) {
	data, ok := f.history[symbol]
	if !ok {
		return nil, errors.New("not found")
	}
	if len(data) > limit {
		data = data[:limit]
	}
	return data, nil
}

func (f fakeQueryRepo) Get(_ context.Context, symbol string, date time.Time, _ string) (analysisDomain.DailyAnalysisResult, error) {
	if f.detailErr != nil {
		return analysisDomain.DailyAnalysisResult{}, f.detailErr
	}
	for _, r := range f.byDate {
		if r.Symbol == symbol && sameDate(r.TradeDate, date) {
			return r, nil
		}
	}
	return analysisDomain.DailyAnalysisResult{}, errors.New("not found")
}

func TestQueryByDate_SuccessWithPagination(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	repo := fakeQueryRepo{
		byDate: []analysisDomain.DailyAnalysisResult{
			{Symbol: "2330", Market: dataingestion.MarketTWSE, TradeDate: date, Success: true, Score: 80},
			{Symbol: "2317", Market: dataingestion.MarketTWSE, TradeDate: date, Success: true, Score: 70},
			{Symbol: "1101", Market: dataingestion.MarketTWSE, TradeDate: date, Success: false, Score: 60},
		},
	}

	usecase := NewQueryUseCase(repo)
	out, err := usecase.QueryByDate(context.Background(), QueryByDateInput{
		Date: date,
		Filter: QueryFilter{
			OnlySuccess: true,
			Markets:     []dataingestion.Market{dataingestion.MarketTWSE},
		},
		Pagination: Pagination{Limit: 1, Offset: 0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Total != 2 || len(out.Results) != 1 || !out.HasMore {
		t.Fatalf("unexpected pagination result: %+v", out)
	}
	if out.Results[0].Symbol != "2330" {
		t.Fatalf("unexpected first result: %+v", out.Results[0])
	}
}

func TestQueryHistory_DefaultLimit(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	history := make([]analysisDomain.DailyAnalysisResult, 0, 5)
	for i := 0; i < 5; i++ {
		history = append(history, analysisDomain.DailyAnalysisResult{
			Symbol:    "2330",
			Market:    dataingestion.MarketTWSE,
			TradeDate: date.AddDate(0, 0, -i),
			Success:   true,
		})
	}

	repo := fakeQueryRepo{history: map[string][]analysisDomain.DailyAnalysisResult{
		"2330": history,
	}}
	usecase := NewQueryUseCase(repo)

	res, err := usecase.QueryHistory(context.Background(), QueryHistoryInput{
		Symbol: "2330",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != len(history) {
		t.Fatalf("expected %d results, got %d", len(history), len(res))
	}
}

func TestQueryDetail_NotFound(t *testing.T) {
	usecase := NewQueryUseCase(fakeQueryRepo{})
	_, err := usecase.QueryDetail(context.Background(), QueryDetailInput{
		Symbol: "9999",
		Date:   time.Now(),
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExportDailyStrong_CSV(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	repo := fakeQueryRepo{
		byDate: []analysisDomain.DailyAnalysisResult{
			{
				Symbol:         "2330",
				Market:         dataingestion.MarketTWSE,
				TradeDate:      date,
				Industry:       "半導體",
				Close:          600,
				ChangeRate:     0.02,
				Return5:        ptr(0.05),
				Return20:       ptr(0.1),
				Volume:         1000,
				VolumeMultiple: ptr(1.5),
				Score:          80,
				Tags:           []analysisDomain.Tag{analysisDomain.TagShortTermStrong, analysisDomain.TagVolumeSurge},
				Success:        true,
			},
		},
	}

	usecase := NewQueryUseCase(repo)
	csvStr, err := usecase.ExportDailyStrong(context.Background(), ExportDailyStrongInput{
		Date: date,
		Filter: QueryFilter{
			OnlySuccess: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(csvStr), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "2330") || !strings.Contains(lines[1], "80.0000") {
		t.Fatalf("unexpected csv content: %s", lines[1])
	}
}

func containsMarket(list []dataingestion.Market, m dataingestion.Market) bool {
	for _, v := range list {
		if v == m {
			return true
		}
	}
	return false
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
