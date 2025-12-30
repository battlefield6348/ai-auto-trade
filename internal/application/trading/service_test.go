package trading

import (
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
