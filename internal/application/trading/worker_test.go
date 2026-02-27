package trading

import (
	"testing"
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
)

func TestBackgroundWorker_RunOnce(t *testing.T) {
	day1 := time.Now().Add(-24 * time.Hour)
	history := []analysisDomain.DailyAnalysisResult{
		{TradeDate: day1, Close: 50000, Score: 75},
	}
	
	activeStrats := []*strategyDomain.ScoringStrategy{
		{
			ID:         "strat-1",
			Name:       "Alpha",
			Slug:       "alpha",
			BaseSymbol: "BTCUSDT",
			Threshold:  60,
			Env:        "paper",
			EntryRules: []strategyDomain.StrategyRule{
				{
					Weight: 1.0,
					RuleType: "entry",
					Condition: strategyDomain.Condition{
						Type: "BASE_SCORE",
					},
				},
			},
		},
	}

	repo := &fakeRepo{activeStrats: activeStrats}
	ex := &mockExchange{}
	svc := NewService(repo, stubDataProvider{history: history}, ex, nil)
	svc.now = func() time.Time { return time.Now() }

	worker := NewBackgroundWorker(svc, 1*time.Hour)
	worker.runOnce()

	if repo.upsertPositionCalled == 0 {
		t.Errorf("expected worker to trigger trade and upsert position")
	}
}
