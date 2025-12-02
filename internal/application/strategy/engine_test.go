package strategy

import (
	"context"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	alertDomain "ai-auto-trade/internal/domain/alert"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
)

type fakeStrategyRepo struct {
	strategies []strategyDomain.Strategy
	runs       []RunRecord
	updates    []struct {
		ID        string
		LastRun   time.Time
		Triggered bool
	}
}

func (f *fakeStrategyRepo) ListActive(context.Context) ([]strategyDomain.Strategy, error) {
	return f.strategies, nil
}

func (f *fakeStrategyRepo) SaveRun(_ context.Context, record RunRecord) error {
	f.runs = append(f.runs, record)
	return nil
}

func (f *fakeStrategyRepo) UpdateState(_ context.Context, id string, lastRun time.Time, triggered bool) error {
	f.updates = append(f.updates, struct {
		ID        string
		LastRun   time.Time
		Triggered bool
	}{id, lastRun, triggered})
	return nil
}

type fakeScreenerExec struct {
	output analysis.ScreenerOutput
	err    error
}

func (f fakeScreenerExec) Run(context.Context, analysis.ScreenerInput) (analysis.ScreenerOutput, error) {
	return f.output, f.err
}

type fakeHistoryProvider struct {
	results map[string][]analysisDomain.DailyAnalysisResult
}

func (f fakeHistoryProvider) QueryHistory(_ context.Context, input analysis.QueryHistoryInput) ([]analysisDomain.DailyAnalysisResult, error) {
	return f.results[input.Symbol], nil
}

type fakeActionDispatcher struct {
	dispatched int
}

func (f *fakeActionDispatcher) Dispatch(context.Context, strategyDomain.StrategyAction, strategyDomain.Strategy, []analysisDomain.DailyAnalysisResult) error {
	f.dispatched++
	return nil
}

func TestEngine_TriggerSingleDay(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	repo := &fakeStrategyRepo{
		strategies: []strategyDomain.Strategy{{
			ID:            "st1",
			Name:          "強勢策略",
			Enabled:       true,
			FrequencyDays: 1,
			Condition: strategyDomain.StrategyCondition{
				Logic:      analysis.LogicAND,
				Conditions: []analysis.Condition{numericCondForTest(analysis.FieldScore, analysis.OpGTE, 50)},
			},
			Actions: []strategyDomain.StrategyAction{
				{Type: strategyDomain.ActionNotify, Channel: alertDomain.ChannelEmail},
			},
		}},
	}

	engine := NewEngine(
		repo,
		fakeScreenerExec{output: analysis.ScreenerOutput{
			Results: []analysisDomain.DailyAnalysisResult{{
				Symbol:    "2330",
				Market:    dataingestion.MarketTWSE,
				TradeDate: date,
				Score:     80,
				Success:   true,
			}},
			Total: 1,
		}},
		fakeHistoryProvider{},
		&fakeActionDispatcher{},
	)

	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.runs) != 1 || !repo.runs[0].Triggered {
		t.Fatalf("expected triggered run, got %+v", repo.runs)
	}
	if len(repo.updates) != 1 || !repo.updates[0].Triggered {
		t.Fatalf("expected state update")
	}
}

func TestEngine_MultiDayCondition(t *testing.T) {
	date := time.Date(2024, 12, 3, 0, 0, 0, 0, time.UTC)
	history := map[string][]analysisDomain.DailyAnalysisResult{
		"2330": {
			{Symbol: "2330", TradeDate: date.AddDate(0, 0, -2), Score: 80, Return5: ptr(0.05), Success: true},
			{Symbol: "2330", TradeDate: date.AddDate(0, 0, -1), Score: 80, Return5: ptr(0.05), Success: true},
			{Symbol: "2330", TradeDate: date, Score: 80, Return5: ptr(0.05), Success: true},
		},
	}
	repo := &fakeStrategyRepo{
		strategies: []strategyDomain.Strategy{{
			ID:            "st2",
			Name:          "連續強勢",
			Enabled:       true,
			FrequencyDays: 1,
			Condition: strategyDomain.StrategyCondition{
				Logic:      analysis.LogicAND,
				Conditions: []analysis.Condition{},
				MultiDay: &strategyDomain.MultiDayCondition{
					Days:      3,
					Condition: numericCondForTest(analysis.FieldReturn5, analysis.OpGTE, 0.01),
				},
			},
			Actions: []strategyDomain.StrategyAction{{Type: strategyDomain.ActionNotify, Channel: alertDomain.ChannelEmail}},
		}},
	}

	engine := NewEngine(
		repo,
		fakeScreenerExec{output: analysis.ScreenerOutput{
			Results: []analysisDomain.DailyAnalysisResult{{
				Symbol:    "2330",
				Market:    dataingestion.MarketTWSE,
				TradeDate: date,
				Score:     80,
				Return5:   ptr(0.05),
				Success:   true,
			}},
			Total: 1,
		}},
		fakeHistoryProvider{results: history},
		&fakeActionDispatcher{},
	)

	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.runs) == 0 || !repo.runs[0].Triggered {
		t.Fatalf("expected triggered for multi-day, got %+v", repo.runs)
	}
}

func TestEngine_MultiDayNotEnoughHistory(t *testing.T) {
	date := time.Date(2024, 12, 3, 0, 0, 0, 0, time.UTC)
	repo := &fakeStrategyRepo{
		strategies: []strategyDomain.Strategy{{
			ID:            "st3",
			Name:          "連續強勢",
			Enabled:       true,
			FrequencyDays: 1,
			Condition: strategyDomain.StrategyCondition{
				Logic: analysis.LogicAND,
				MultiDay: &strategyDomain.MultiDayCondition{
					Days:      3,
					Condition: numericCondForTest(analysis.FieldReturn5, analysis.OpGTE, 0.01),
				},
			},
			Actions: []strategyDomain.StrategyAction{{Type: strategyDomain.ActionNotify, Channel: alertDomain.ChannelEmail}},
		}},
	}

	engine := NewEngine(
		repo,
		fakeScreenerExec{output: analysis.ScreenerOutput{
			Results: []analysisDomain.DailyAnalysisResult{{
				Symbol:    "2330",
				Market:    dataingestion.MarketTWSE,
				TradeDate: date,
				Score:     80,
				Return5:   ptr(0.05),
				Success:   true,
			}},
			Total: 1,
		}},
		fakeHistoryProvider{results: map[string][]analysisDomain.DailyAnalysisResult{}},
		&fakeActionDispatcher{},
	)

	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.runs) > 0 && repo.runs[0].Triggered {
		t.Fatalf("expected not triggered due to insufficient history")
	}
}

func TestEngine_SkipByFrequency(t *testing.T) {
	date := time.Date(2024, 12, 3, 0, 0, 0, 0, time.UTC)
	lastRun := date
	repo := &fakeStrategyRepo{
		strategies: []strategyDomain.Strategy{{
			ID:            "st4",
			Name:          "頻率測試",
			Enabled:       true,
			FrequencyDays: 2,
			LastRun:       &lastRun,
			Condition:     strategyDomain.StrategyCondition{},
			Actions:       []strategyDomain.StrategyAction{{Type: strategyDomain.ActionNotify, Channel: alertDomain.ChannelEmail}},
		}},
	}
	engine := NewEngine(repo, fakeScreenerExec{}, fakeHistoryProvider{}, &fakeActionDispatcher{})

	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.runs) != 0 {
		t.Fatalf("expected skip due to frequency")
	}
}

// helper to build numeric condition as analysis.Condition
func numericCondForTest(field analysis.NumericField, op analysis.NumericOp, value float64) analysis.Condition {
	return analysis.Condition{
		Type: analysis.ConditionNumeric,
		Numeric: &analysis.NumericCondition{
			Field: field,
			Op:    op,
			Value: value,
		},
	}
}

func ptr(v float64) *float64 { return &v }
