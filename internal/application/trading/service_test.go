package trading

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
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
	svc := NewService(repo, dummyDataProvider{}, &mockExchange{}, nil)

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
	svc := NewService(repo, dummyDataProvider{}, &mockExchange{}, nil)

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

func TestValidateConditionSet_InvalidLogic(t *testing.T) {
	set := tradingDomain.ConditionSet{
		Logic: "X",
		Conditions: []analysis.Condition{
			{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 1}},
		},
	}
	if err := validateConditionSet(set); err == nil {
		t.Fatalf("expected invalid logic to fail")
	}
}

func TestApplyRiskDefaults(t *testing.T) {
	r := tradingDomain.RiskSettings{}
	out := applyRiskDefaults(r)
	if out.OrderSizeMode != tradingDomain.OrderFixedUSDT || out.OrderSizeValue == 0 {
		t.Fatalf("order size defaults not applied: %+v", out)
	}
	if out.PriceMode != tradingDomain.PriceNextOpen {
		t.Fatalf("price mode default missing: %+v", out)
	}
	if out.FeesPct == 0 || out.SlippagePct == 0 {
		t.Fatalf("fees/slippage defaults missing: %+v", out)
	}
	if out.MaxPositions != 1 {
		t.Fatalf("max positions default missing: %+v", out)
	}
}

func TestMergeParamsOverrides(t *testing.T) {
	stop := 0.05
	take := 0.1
	maxDaily := 0.2
	cool := 2
	minHold := 1
	maxPos := 3
	priceMode := tradingDomain.PriceCurrentClose
	fees := 0.002
	slip := 0.003
	strategy := tradingDomain.Strategy{
		Risk: tradingDomain.RiskSettings{
			PriceMode:       tradingDomain.PriceNextOpen,
			FeesPct:         0.001,
			SlippagePct:     0.001,
			StopLossPct:     &stop,
			TakeProfitPct:   &take,
			MaxDailyLossPct: &maxDaily,
			CoolDownDays:    5,
			MinHoldDays:     2,
			MaxPositions:    1,
		},
	}
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 10)
	input := BacktestInput{
		StartDate:     start,
		EndDate:       end,
		InitialEquity: 20000,
		PriceMode:     &priceMode,
		FeesPct:       &fees,
		SlippagePct:   &slip,
		CoolDownDays:  &cool,
		MinHoldDays:   &minHold,
		MaxPositions:  &maxPos,
	}

	params := mergeParams(strategy, input)
	if params.PriceMode != priceMode || params.FeesPct != fees || params.SlippagePct != slip {
		t.Fatalf("overrides not applied: %+v", params)
	}
	if params.CoolDownDays != cool || params.MinHoldDays != minHold || params.MaxPositions != maxPos {
		t.Fatalf("int overrides not applied: %+v", params)
	}
	if params.InitialEquity != 20000 {
		t.Fatalf("initial equity override missing: %f", params.InitialEquity)
	}
	if params.StartDate != start || params.EndDate != end {
		t.Fatalf("date not propagated")
	}
	if params.StopLossPct == nil || *params.StopLossPct != stop {
		t.Fatalf("stop loss lost: %+v", params.StopLossPct)
	}
	if params.TakeProfitPct == nil || *params.TakeProfitPct != take {
		t.Fatalf("take profit lost: %+v", params.TakeProfitPct)
	}
	if params.MaxDailyLossPct == nil || *params.MaxDailyLossPct != maxDaily {
		t.Fatalf("max daily loss lost: %+v", params.MaxDailyLossPct)
	}
}

func TestMergeParams_DefaultInitialEquity(t *testing.T) {
	strategy := tradingDomain.Strategy{Risk: tradingDomain.RiskSettings{PriceMode: tradingDomain.PriceNextOpen}}
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	params := mergeParams(strategy, BacktestInput{StartDate: start, EndDate: end})
	if params.InitialEquity != 10000 {
		t.Fatalf("expected default initial equity 10000, got %f", params.InitialEquity)
	}
}

func TestOrderSizePercentEquity(t *testing.T) {
	engine := backtestEngine{
		params: tradingDomain.BacktestParams{
			Strategy: tradingDomain.Strategy{
				Risk: tradingDomain.RiskSettings{
					OrderSizeMode:  tradingDomain.OrderPercentEquity,
					OrderSizeValue: 0.1,
				},
			},
		},
	}
	if v := engine.orderSize(10000); v != 1000 {
		t.Fatalf("percent equity order size wrong: %f", v)
	}
}

func TestPickPriceModes(t *testing.T) {
	date := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	priceMap := map[string]dataDomain.DailyPrice{
		"2025-01-02": {Open: 10, Close: 12},
	}
	engine := backtestEngine{}

	price, ok := engine.pickPrice(date, priceMap, tradingDomain.PriceNextOpen, 9)
	if !ok || price != 10 {
		t.Fatalf("next open failed, price=%f ok=%v", price, ok)
	}
	price, ok = engine.pickPrice(date, priceMap, tradingDomain.PriceNextClose, 9)
	if !ok || price != 12 {
		t.Fatalf("next close failed, price=%f ok=%v", price, ok)
	}
	_, ok = engine.pickPrice(date, priceMap, "unknown", 9)
	if ok {
		t.Fatalf("unknown mode should fail")
	}
}

func TestComputeStats(t *testing.T) {
	trades := []tradingDomain.BacktestTrade{
		{PNL: 100}, {PNL: -50}, {PNL: 150},
	}
	equity := []tradingDomain.EquityPoint{
		{Date: time.Now(), Equity: 10000},
		{Date: time.Now(), Equity: 9000},
		{Date: time.Now(), Equity: 12000},
	}
	stats := computeStats(trades, equity, 10000)
	if stats.TotalReturn < 0.19 || stats.TotalReturn > 0.21 {
		t.Fatalf("unexpected total return %f", stats.TotalReturn)
	}
	if stats.MaxDrawdown < 0.09 || stats.MaxDrawdown > 0.11 {
		t.Fatalf("unexpected max drawdown %f", stats.MaxDrawdown)
	}
	if stats.TradeCount != 3 || stats.WinRate < 0.65 || stats.WinRate > 0.67 {
		t.Fatalf("unexpected trade count/win rate: %d %f", stats.TradeCount, stats.WinRate)
	}
	if stats.ProfitFactor < 4.9 || stats.ProfitFactor > 5.1 {
		t.Fatalf("unexpected profit factor %f", stats.ProfitFactor)
	}
	if stats.AvgGain < 124 || stats.AvgGain > 126 {
		t.Fatalf("unexpected avg gain %f", stats.AvgGain)
	}
	if stats.AvgLoss > -49 || stats.AvgLoss < -51 {
		t.Fatalf("unexpected avg loss %f", stats.AvgLoss)
	}
}

func TestBacktest_RequireDates(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, dummyDataProvider{}, &mockExchange{}, nil)
	if _, err := svc.Backtest(context.Background(), BacktestInput{}); err == nil {
		t.Fatalf("expected error when dates missing")
	}
	if repo.saveBacktestCalled != 0 {
		t.Fatalf("should not save backtest on invalid input")
	}
}

func TestBacktest_InlineStrategyAndSave(t *testing.T) {
	day1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 := day1.AddDate(0, 0, 1)
	history := []analysisDomain.DailyAnalysisResult{
		{TradeDate: day1, Close: 100, Score: 60},
		{TradeDate: day2, Close: 110, Score: 80},
	}
	prices := []dataDomain.DailyPrice{
		{TradeDate: day1, Open: 100, Close: 100},
		{TradeDate: day2, Open: 110, Close: 110},
	}
	repo := &fakeRepo{}
	svc := NewService(repo, stubDataProvider{history: history, prices: prices}, &mockExchange{}, nil)

	strategy := tradingDomain.Strategy{
		ID:         "s1",
		Name:       "inline",
		BaseSymbol: "BTCUSDT",
		Buy: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 50}},
			},
		},
		Sell: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 70}},
			},
		},
		Risk: tradingDomain.RiskSettings{OrderSizeValue: 1000, PriceMode: tradingDomain.PriceCurrentClose},
	}

	input := BacktestInput{
		Inline:        &strategy,
		StartDate:     day1,
		EndDate:       day2,
		InitialEquity: 5000,
		CreatedBy:     "u1",
		Save:          true,
	}
	rec, err := svc.Backtest(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.saveBacktestCalled != 1 {
		t.Fatalf("expected save backtest once, got %d", repo.saveBacktestCalled)
	}
	if rec.Result.Stats.TradeCount == 0 {
		t.Fatalf("expected trades in result")
	}
	if repo.lastBacktest.StrategyID != "s1" {
		t.Fatalf("strategy id not propagated")
	}
}

func TestBacktest_GetStrategyError(t *testing.T) {
	repo := &fakeRepo{getErr: fmt.Errorf("boom")}
	svc := NewService(repo, dummyDataProvider{}, &mockExchange{}, nil)
	_, err := svc.Backtest(context.Background(), BacktestInput{
		StrategyID: "s1",
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 0, 1),
	})
	if err == nil {
		t.Fatalf("expected error when repo get fails")
	}
}

func TestUpdateStrategy_VersionBump(t *testing.T) {
	repo := &fakeRepo{
		lastStrategy: tradingDomain.Strategy{
			ID:         "s1",
			Name:       "old",
			BaseSymbol: "BTCUSDT",
			Timeframe:  "1d",
			Env:        tradingDomain.EnvBoth,
			Status:     tradingDomain.StatusDraft,
			Version:    1,
			Buy: tradingDomain.ConditionSet{
				Logic: analysis.LogicAND,
				Conditions: []analysis.Condition{
					{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 50}},
				},
			},
			Sell: tradingDomain.ConditionSet{
				Logic: analysis.LogicAND,
				Conditions: []analysis.Condition{
					{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpLTE, Value: 30}},
				},
			},
			Risk: tradingDomain.RiskSettings{OrderSizeValue: 1000, PriceMode: tradingDomain.PriceNextOpen},
		},
	}
	svc := NewService(repo, dummyDataProvider{}, &mockExchange{}, nil)
	update := tradingDomain.Strategy{
		Name:        "new",
		Description: "desc",
		Status:      tradingDomain.StatusActive,
		Buy: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 60}},
			},
		},
		Sell: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpLTE, Value: 20}},
			},
		},
		Risk:      tradingDomain.RiskSettings{OrderSizeValue: 2000},
		UpdatedBy: "u1",
	}

	out, err := svc.UpdateStrategy(context.Background(), "s1", update)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Version != 2 {
		t.Fatalf("expected version bump to 2, got %d", out.Version)
	}
	if repo.updateCalled != 1 {
		t.Fatalf("expected repo update called once")
	}
	if repo.lastStrategy.Risk.OrderSizeValue != 2000 {
		t.Fatalf("risk update not applied: %+v", repo.lastStrategy.Risk)
	}
	if repo.lastStrategy.Name != "new" || repo.lastStrategy.Status != tradingDomain.StatusActive {
		t.Fatalf("fields not updated")
	}
}

type fakeRepo struct {
	createCalled       int
	updateCalled       int
	deleteCalled       int
	saveBacktestCalled int
	lastStrategy       tradingDomain.Strategy
	lastBacktest       tradingDomain.BacktestRecord
	id                 string
	getErr             error
}

func (f *fakeRepo) CreateStrategy(_ context.Context, s tradingDomain.Strategy) (string, error) {
	f.createCalled++
	f.lastStrategy = s
	if f.id == "" {
		return "fake-id", nil
	}
	return f.id, nil
}

func (f *fakeRepo) UpdateStrategy(_ context.Context, s tradingDomain.Strategy) error {
	f.updateCalled++
	f.lastStrategy = s
	return nil
}
func (f *fakeRepo) DeleteStrategy(_ context.Context, id string) error {
	f.deleteCalled++
	return nil
}
func (f *fakeRepo) GetStrategy(context.Context, string) (tradingDomain.Strategy, error) {
	if f.getErr != nil {
		return tradingDomain.Strategy{}, f.getErr
	}
	return f.lastStrategy, nil
}
func (f *fakeRepo) ListStrategies(context.Context, StrategyFilter) ([]tradingDomain.Strategy, error) {
	return nil, nil
}
func (f *fakeRepo) GetStrategyBySlug(context.Context, string) (tradingDomain.Strategy, error) {
	return f.lastStrategy, nil
}
func (f *fakeRepo) SetStatus(context.Context, string, tradingDomain.Status, tradingDomain.Environment) error {
	return nil
}
func (f *fakeRepo) UpdateLastActivatedAt(context.Context, string, time.Time) error {
	return nil
}
func (f *fakeRepo) UpdateRiskSettings(context.Context, string, tradingDomain.RiskSettings) error {
	return nil
}
func (f *fakeRepo) SaveBacktest(_ context.Context, rec tradingDomain.BacktestRecord) (string, error) {
	f.saveBacktestCalled++
	f.lastBacktest = rec
	return "bt-1", nil
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
func (f *fakeRepo) GetPosition(context.Context, string) (*tradingDomain.Position, error) {
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
func (f *fakeRepo) LoadScoringStrategyBySlug(ctx context.Context, slug string) (*strategyDomain.ScoringStrategy, error) {
	return nil, nil
}
func (f *fakeRepo) LoadScoringStrategyByID(ctx context.Context, id string) (*strategyDomain.ScoringStrategy, error) {
	return nil, nil
}
func (f *fakeRepo) ListActiveScoringStrategies(ctx context.Context) ([]*strategyDomain.ScoringStrategy, error) {
	return nil, nil
}

type dummyDataProvider struct{}

func (dummyDataProvider) FindHistory(context.Context, string, string, *time.Time, *time.Time, int, bool) ([]analysisDomain.DailyAnalysisResult, error) {
	return nil, nil
}

func (dummyDataProvider) PricesByPair(context.Context, string, string) ([]dataDomain.DailyPrice, error) {
	return nil, nil
}

type stubDataProvider struct {
	history []analysisDomain.DailyAnalysisResult
	prices  []dataDomain.DailyPrice
}

func (s stubDataProvider) FindHistory(context.Context, string, string, *time.Time, *time.Time, int, bool) ([]analysisDomain.DailyAnalysisResult, error) {
	return s.history, nil
}

func (s stubDataProvider) PricesByPair(context.Context, string, string) ([]dataDomain.DailyPrice, error) {
	return s.prices, nil
}

type mockExchange struct{}

func (m *mockExchange) GetBalance(ctx context.Context, asset string) (float64, error) {
	return 0, nil
}
func (m *mockExchange) GetOrder(ctx context.Context, symbol, orderID string) (OrderResponse, error) {
	return OrderResponse{}, nil
}
func (m *mockExchange) GetPrice(ctx context.Context, symbol string) (float64, error) {
	return 0, nil
}
func (m *mockExchange) PlaceMarketOrder(ctx context.Context, symbol, side string, qty float64) (float64, float64, error) {
	return 0, 0, nil
}
func (m *mockExchange) PlaceMarketOrderQuote(ctx context.Context, symbol, side string, quoteAmount float64) (float64, float64, error) {
	return 0, 0, nil
}
