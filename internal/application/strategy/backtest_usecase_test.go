package strategy

import (
	"context"
	"testing"
	"time"

	"ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/strategy"
	tradingDomain "ai-auto-trade/internal/domain/trading"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockDataProvider struct {
	history []analysis.DailyAnalysisResult
}

func (m *mockDataProvider) FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysis.DailyAnalysisResult, error) {
	return m.history, nil
}

func TestBacktestUseCase_ExecuteWithStrategy(t *testing.T) {
	// Setup simple history
	h := []analysis.DailyAnalysisResult{
		{TradeDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Score: 80}, // Trigger!
		{TradeDate: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), Close: 105, Score: 80},
		{TradeDate: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), Close: 95, Score: 20},  // Weak signal -> Exit
		{TradeDate: time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC), Close: 110, Score: 80}, // Trigger again
		{TradeDate: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC), Close: 120, Score: 80}, // TP? (depends on TP setting)
	}

	dataProv := &mockDataProvider{history: h}
	usecase := NewBacktestUseCase(nil, dataProv)

	// Setup Strategy
	tp := 100.0 // Set TP very high so it doesn't trigger
	s := &strategy.ScoringStrategy{
		Name:      "Test Strategy",
		Threshold: 70.0,
		Risk: tradingDomain.RiskSettings{
			TakeProfitPct: &tp,
		},
		EntryRules: []strategy.StrategyRule{
			{
				Condition: strategy.Condition{Type: "BASE_SCORE"},
				Weight:    1.0,
			},
		},
	}

	res, err := usecase.ExecuteWithStrategy(context.Background(), s, "BTCUSDT", h[0].TradeDate, h[4].TradeDate, []int{1})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if res.TotalEvents != 5 {
		t.Errorf("Expected 5 events, got %d", res.TotalEvents)
	}

	// Trade 1: Entry at 100 (Day 1), Exit at 95 (Day 3) due to weak signal (20 < 70*0.5)
	if len(res.Trades) < 1 {
		t.Fatal("Expected at least 1 trade")
	}

	trade1 := res.Trades[0]
	if trade1.EntryPrice != 100 {
		t.Errorf("Trade1 entry price expected 100, got %f", trade1.EntryPrice)
	}
	if trade1.ExitPrice != 95 {
		t.Errorf("Trade1 exit price expected 95, got %f", trade1.ExitPrice)
	}
	// Check reason - should mention AI signal decay
	if trade1.Reason == "" {
		t.Error("Trade1 reason should not be empty")
	}
	
	t.Logf("Trade 1: %+v", trade1)
	t.Logf("Summary: %+v", res.Summary)
}

func TestBacktestUseCase_TPSL(t *testing.T) {
	// Setup history for SL
	h := []analysis.DailyAnalysisResult{
		{TradeDate: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Score: 90}, 
		{TradeDate: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), Close: 97, Score: 90}, // -3% -> trigger SL -2%
	}

	dataProv := &mockDataProvider{history: h}
	usecase := NewBacktestUseCase(nil, dataProv)

	s := &strategy.ScoringStrategy{
		Threshold: 70.0,
		EntryRules: []strategy.StrategyRule{
			{Condition: strategy.Condition{Type: "BASE_SCORE"}, Weight: 1.0},
		},
	}
	// Default SL is -2% (from evaluator.go)
	
	res, err := usecase.ExecuteWithStrategy(context.Background(), s, "BTCUSDT", h[0].TradeDate, h[1].TradeDate, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Trades) != 1 {
		t.Fatalf("Expected 1 trade (SL), got %d", len(res.Trades))
	}

	if res.Trades[0].ExitPrice != 97 {
		t.Errorf("Expected exit at 97, got %f", res.Trades[0].ExitPrice)
	}
}
func TestBacktestUseCase_Execute(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	// Mock LoadScoringStrategyBySlug
	// 1. Fetch the base Strategy
	rows := sqlmock.NewRows([]string{"id", "user_id", "name", "slug", "description", "timeframe", "base_symbol", "threshold", "exit_threshold", "is_active", "env", "risk_settings", "created_at", "updated_at"}).
		AddRow("s-1", "u-1", "Alpha", "alpha", "desc", "1d", "BTCUSDT", 60.0, 30.0, true, "paper", []byte("{}"), time.Now(), time.Now())

	mock.ExpectQuery("SELECT (.+) FROM strategies WHERE slug = \\$1").
		WithArgs("alpha").
		WillReturnRows(rows)

	// 2. Fetch Rules and Conditions
	mock.ExpectQuery("SELECT (.+) FROM strategy_rules (.+) WHERE sr.strategy_id = \\$1").
		WithArgs("s-1").
		WillReturnRows(sqlmock.NewRows([]string{"strategy_id", "weight", "rule_type", "id", "name", "type", "params"}).
			AddRow("s-1", 1.0, "entry", "c-1", "Base Score", "BASE_SCORE", []byte("{}")))

	h := []analysis.DailyAnalysisResult{
		{TradeDate: time.Now().Add(-24 * time.Hour), Close: 50000, Score: 70},
	}
	dataProv := &mockDataProvider{history: h}
	usecase := NewBacktestUseCase(db, dataProv)

	res, err := usecase.Execute(context.Background(), "alpha", "BTCUSDT", time.Now().Add(-48*time.Hour), time.Now(), nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if res.Symbol != "BTCUSDT" {
		t.Errorf("Expected BTCUSDT, got %s", res.Symbol)
	}
}
