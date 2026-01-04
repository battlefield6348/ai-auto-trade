package trading

import (
	"context"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	tradingDomain "ai-auto-trade/internal/domain/trading"
)

func TestBacktestEngine_BuyThenSellNextDay(t *testing.T) {
	day1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := day1.AddDate(0, 0, 1)

	history := []analysisDomain.DailyAnalysisResult{
		{TradeDate: day1, Close: 100, Score: 1},
		{TradeDate: day2, Close: 110, Score: 3},
	}
	prices := []dataDomain.DailyPrice{
		{TradeDate: day1, Open: 100, Close: 100},
		{TradeDate: day2, Open: 110, Close: 110},
	}

	engine := backtestEngine{
		params: tradingDomain.BacktestParams{
			StartDate:     day1,
			EndDate:       day2,
			InitialEquity: 10000,
			PriceMode:     tradingDomain.PriceCurrentClose,
			FeesPct:       0,
			SlippagePct:   0,
			Strategy: tradingDomain.Strategy{
				Buy: tradingDomain.ConditionSet{
					Logic: analysis.LogicAND,
					Conditions: []analysis.Condition{
						{
							Type: analysis.ConditionNumeric,
							Numeric: &analysis.NumericCondition{
								Field: analysis.FieldScore,
								Op:    analysis.OpGTE,
								Value: 1,
							},
						},
					},
				},
				Sell: tradingDomain.ConditionSet{
					Logic: analysis.LogicAND,
					Conditions: []analysis.Condition{
						{
							Type: analysis.ConditionNumeric,
							Numeric: &analysis.NumericCondition{
								Field: analysis.FieldScore,
								Op:    analysis.OpGTE,
								Value: 2,
							},
						},
					},
				},
				Risk: tradingDomain.RiskSettings{
					OrderSizeMode:  tradingDomain.OrderFixedUSDT,
					OrderSizeValue: 1000,
				},
			},
		},
		history: history,
		prices:  prices,
	}

	result := engine.Run()
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	tr := result.Trades[0]
	if !tr.EntryDate.Equal(day1) || !tr.ExitDate.Equal(day2) {
		t.Fatalf("unexpected trade dates: entry %v exit %v", tr.EntryDate, tr.ExitDate)
	}
	if tr.PNL <= 0 {
		t.Fatalf("expected positive pnl, got %f", tr.PNL)
	}
	if tr.PNLPct <= 0 {
		t.Fatalf("expected positive pnl pct, got %f", tr.PNLPct)
	}
	if result.Stats.TotalReturn <= 0 {
		t.Fatalf("expected positive total return, got %f", result.Stats.TotalReturn)
	}
	if result.Stats.TradeCount != 1 {
		t.Fatalf("expected trade count 1, got %d", result.Stats.TradeCount)
	}
	if result.Stats.WinRate != 1 {
		t.Fatalf("expected win rate 1, got %f", result.Stats.WinRate)
	}
}

func TestValidateStrategyContent_SingleConditionLimit(t *testing.T) {
	s := tradingDomain.Strategy{
		Buy: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 60}},
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 70}},
			},
		},
		Sell: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpLTE, Value: 40}},
			},
		},
		Risk: tradingDomain.RiskSettings{OrderSizeMode: tradingDomain.OrderFixedUSDT, OrderSizeValue: 1000, PriceMode: tradingDomain.PriceNextOpen},
	}

	if err := validateStrategyContent(s); err == nil {
		t.Fatalf("expected validation to fail when buy has more than one condition")
	}
}

func TestCreateStrategy_RequireUser(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, dummyDataProvider{})

	input := tradingDomain.Strategy{
		Name: "no-user",
		Buy: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 60}},
			},
		},
		Sell: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpLTE, Value: 40}},
			},
		},
		Risk: tradingDomain.RiskSettings{OrderSizeValue: 500},
	}

	if _, err := svc.CreateStrategy(context.Background(), input); err == nil {
		t.Fatalf("expected error when created_by is missing")
	}
	if repo.createCalled != 0 {
		t.Fatalf("repo should not be called when user missing")
	}
}

func TestCreateStrategy_DefaultsAndPersist(t *testing.T) {
	repo := &fakeRepo{id: "id-123"}
	svc := NewService(repo, dummyDataProvider{})

	input := tradingDomain.Strategy{
		Name:        "with-user",
		Description: "desc",
		CreatedBy:   "user-1",
		Buy: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 60}},
			},
		},
		Sell: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpLTE, Value: 40}},
			},
		},
		Risk: tradingDomain.RiskSettings{OrderSizeValue: 800},
	}

	out, err := svc.CreateStrategy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "id-123" {
		t.Fatalf("expected id-123, got %s", out.ID)
	}
	if repo.createCalled != 1 {
		t.Fatalf("expected repo CreateStrategy called once, got %d", repo.createCalled)
	}
	if repo.lastStrategy.BaseSymbol != "BTCUSDT" || repo.lastStrategy.Timeframe != "1d" || repo.lastStrategy.Env != tradingDomain.EnvBoth {
		t.Fatalf("defaults not applied: %+v", repo.lastStrategy)
	}
	if repo.lastStrategy.Status != tradingDomain.StatusDraft || repo.lastStrategy.Version != 1 {
		t.Fatalf("status/version defaults incorrect: %+v", repo.lastStrategy)
	}
	if repo.lastStrategy.Risk.OrderSizeValue == 0 {
		t.Fatalf("risk defaults not applied")
	}
}

type fakeRepo struct {
	createCalled int
	lastStrategy tradingDomain.Strategy
	id           string
}

func (f *fakeRepo) CreateStrategy(_ context.Context, s tradingDomain.Strategy) (string, error) {
	f.createCalled++
	f.lastStrategy = s
	if f.id == "" {
		return "fake-id", nil
	}
	return f.id, nil
}

func (f *fakeRepo) UpdateStrategy(context.Context, tradingDomain.Strategy) error { return nil }
func (f *fakeRepo) GetStrategy(context.Context, string) (tradingDomain.Strategy, error) {
	return f.lastStrategy, nil
}
func (f *fakeRepo) ListStrategies(context.Context, StrategyFilter) ([]tradingDomain.Strategy, error) {
	return nil, nil
}
func (f *fakeRepo) SetStatus(context.Context, string, tradingDomain.Status, tradingDomain.Environment) error {
	return nil
}
func (f *fakeRepo) SaveBacktest(context.Context, tradingDomain.BacktestRecord) (string, error) {
	return "", nil
}
func (f *fakeRepo) ListBacktests(context.Context, string) ([]tradingDomain.BacktestRecord, error) {
	return nil, nil
}
func (f *fakeRepo) SaveTrade(context.Context, tradingDomain.TradeRecord) error { return nil }
func (f *fakeRepo) ListTrades(context.Context, tradingDomain.TradeFilter) ([]tradingDomain.TradeRecord, error) {
	return nil, nil
}
func (f *fakeRepo) GetOpenPosition(context.Context, string, tradingDomain.Environment) (*tradingDomain.Position, error) {
	return nil, nil
}
func (f *fakeRepo) ListOpenPositions(context.Context) ([]tradingDomain.Position, error) {
	return nil, nil
}
func (f *fakeRepo) UpsertPosition(context.Context, tradingDomain.Position) error    { return nil }
func (f *fakeRepo) ClosePosition(context.Context, string, time.Time, float64) error { return nil }
func (f *fakeRepo) SaveLog(context.Context, tradingDomain.LogEntry) error           { return nil }
func (f *fakeRepo) ListLogs(context.Context, tradingDomain.LogFilter) ([]tradingDomain.LogEntry, error) {
	return nil, nil
}
func (f *fakeRepo) SaveReport(context.Context, tradingDomain.Report) (string, error) { return "", nil }
func (f *fakeRepo) ListReports(context.Context, string) ([]tradingDomain.Report, error) {
	return nil, nil
}

type dummyDataProvider struct{}

func (dummyDataProvider) FindHistory(context.Context, string, *time.Time, *time.Time, int, bool) ([]analysisDomain.DailyAnalysisResult, error) {
	return nil, nil
}

func (dummyDataProvider) PricesByPair(context.Context, string) ([]dataDomain.DailyPrice, error) {
	return nil, nil
}
