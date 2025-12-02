package analysis

import (
	"context"
	"testing"
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

type fakeQueryRepoForScreener struct {
	results []analysisDomain.DailyAnalysisResult
}

func (f fakeQueryRepoForScreener) FindByDate(_ context.Context, date time.Time, _ QueryFilter, _ SortOption, _ Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	var out []analysisDomain.DailyAnalysisResult
	for _, r := range f.results {
		if sameDate(r.TradeDate, date) {
			out = append(out, r)
		}
	}
	return out, len(out), nil
}

func (f fakeQueryRepoForScreener) FindHistory(_ context.Context, _ string, _, _ *time.Time, _ int, _ bool) ([]analysisDomain.DailyAnalysisResult, error) {
	return nil, nil
}

func (f fakeQueryRepoForScreener) Get(_ context.Context, _ string, _ time.Time) (analysisDomain.DailyAnalysisResult, error) {
	return analysisDomain.DailyAnalysisResult{}, nil
}

func TestScreenerUseCase_AND(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	repo := fakeQueryRepoForScreener{
		results: []analysisDomain.DailyAnalysisResult{
			{Symbol: "2330", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 85, Return5: ptr(0.06), RangePos20: ptr(0.9), VolumeMultiple: ptr(1.5), Tags: []analysisDomain.Tag{analysisDomain.TagShortTermStrong, analysisDomain.TagVolumeSurge}, Success: true},
			{Symbol: "2317", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 55, Return5: ptr(0.01), RangePos20: ptr(0.3), VolumeMultiple: ptr(1.1), Tags: []analysisDomain.Tag{analysisDomain.TagLowVolatility}, Success: true},
		},
	}

	usecase := NewScreenerUseCase(repo)
	out, err := usecase.Run(context.Background(), ScreenerInput{
		Date:  date,
		Logic: LogicAND,
		Conditions: []Condition{
			numericCond(FieldReturn5, OpGTE, 0.05),
			numericCond(FieldRangePos20, OpGTE, 0.8),
			{Type: ConditionTags, Tags: &TagsCondition{IncludeAny: []analysisDomain.Tag{analysisDomain.TagShortTermStrong}}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Total != 1 || len(out.Results) != 1 {
		t.Fatalf("expected 1 match, got %+v", out)
	}
	if out.Results[0].Symbol != "2330" {
		t.Fatalf("unexpected symbol: %s", out.Results[0].Symbol)
	}
}

func TestScreenerUseCase_OR(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	repo := fakeQueryRepoForScreener{
		results: []analysisDomain.DailyAnalysisResult{
			{Symbol: "2330", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 85, Return5: ptr(0.06), Success: true},
			{Symbol: "2317", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 55, Return5: ptr(0.01), Tags: []analysisDomain.Tag{analysisDomain.TagVolumeSurge}, Success: true},
		},
	}

	usecase := NewScreenerUseCase(repo)
	out, err := usecase.Run(context.Background(), ScreenerInput{
		Date:  date,
		Logic: LogicOR,
		Conditions: []Condition{
			numericCond(FieldScore, OpGTE, 80),
			{Type: ConditionTags, Tags: &TagsCondition{IncludeAny: []analysisDomain.Tag{analysisDomain.TagVolumeSurge}}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Total != 2 {
		t.Fatalf("expected 2 matches, got %d", out.Total)
	}
}

func TestScreenerUseCase_SymbolExclude(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	repo := fakeQueryRepoForScreener{
		results: []analysisDomain.DailyAnalysisResult{
			{Symbol: "2330", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 85, Return5: ptr(0.06), Success: true},
			{Symbol: "2317", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 90, Return5: ptr(0.10), Success: true},
		},
	}

	usecase := NewScreenerUseCase(repo)
	out, err := usecase.Run(context.Background(), ScreenerInput{
		Date: date,
		Conditions: []Condition{
			{Type: ConditionSymbols, Symbols: &SymbolCondition{Exclude: []string{"2330"}}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 1 || out.Results[0].Symbol != "2317" {
		t.Fatalf("unexpected results: %+v", out)
	}
}

func TestPresetTemplates(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	templates := PresetTemplates(date)
	if len(templates) == 0 {
		t.Fatalf("expected preset templates")
	}
	found := false
	for _, tpl := range templates {
		if tpl.ID == "short_term_strong" {
			found = true
		}
		if tpl.Input.Date.IsZero() {
			t.Fatalf("template date should be set")
		}
	}
	if !found {
		t.Fatalf("missing short_term_strong template")
	}
}
