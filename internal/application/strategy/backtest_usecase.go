package strategy

import (
	"context"
	"fmt"
	"sort"
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
)

type BacktestResult struct {
	Symbol      string                   `json:"symbol"`
	StartDate   string                   `json:"start_date"`
	EndDate     string                   `json:"end_date"`
	TotalEvents int                      `json:"total_events"`
	Events      []BacktestEvent          `json:"events"`
	Stats       map[string]BacktestStats `json:"stats"`
}

type BacktestEvent struct {
	TradeDate     string             `json:"trade_date"`
	ClosePrice    float64            `json:"close_price"`
	ChangePercent float64            `json:"change_percent"`
	TotalScore    float64            `json:"total_score"`
	IsTriggered   bool               `json:"is_triggered"`
	Return5d      *float64           `json:"return_5d"`
	ForwardReturns map[string]float64 `json:"forward_returns,omitempty"`
}

type BacktestStats struct {
	AvgReturn float64 `json:"avg_return"`
	WinRate   float64 `json:"win_rate"`
}

type DataProvider interface {
	FindHistory(ctx context.Context, symbol string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error)
}

type BacktestUseCase struct {
	db       strategyDomain.DBQueryer
	dataProv DataProvider
}

func NewBacktestUseCase(db strategyDomain.DBQueryer, dataProv DataProvider) *BacktestUseCase {
	return &BacktestUseCase{db: db, dataProv: dataProv}
}

func (u *BacktestUseCase) Execute(ctx context.Context, slug string, symbol string, start, end time.Time) (*BacktestResult, error) {
	// 1. Load Strategy
	s, err := strategyDomain.LoadScoringStrategy(ctx, u.db, slug)
	if err != nil {
		return nil, fmt.Errorf("load strategy failed: %w", err)
	}

	// 2. Load History
	history, err := u.dataProv.FindHistory(ctx, symbol, &start, &end, 5000, true)
	if err != nil {
		return nil, fmt.Errorf("fetch history failed: %w", err)
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].TradeDate.Before(history[j].TradeDate)
	})

	horizons := []int{3, 5, 10}
	var events []BacktestEvent
	retStats := make(map[int][]float64)

	for idx, res := range history {
		triggered, score, err := s.IsTriggered(res)
		if err != nil {
			continue
		}

		if !triggered {
			continue
		}

		// Calculate forward returns (simulated "what if we bought here")
		forward := calculateForwardReturns(history, idx, horizons)
		for _, h := range horizons {
			if val, ok := forward[fmt.Sprintf("d%d", h)]; ok {
				retStats[h] = append(retStats[h], val)
			}
		}

		events = append(events, BacktestEvent{
			TradeDate:      res.TradeDate.Format("2006-01-02"),
			ClosePrice:     res.Close,
			ChangePercent:  res.ChangeRate,
			TotalScore:     score,
			IsTriggered:    triggered,
			Return5d:       res.Return5,
			ForwardReturns: forward,
		})
	}

	// 3. Summarize Stats
	stats := make(map[string]BacktestStats)
	for h, vals := range retStats {
		if len(vals) == 0 {
			continue
		}
		var sum float64
		wins := 0
		for _, v := range vals {
			sum += v
			if v > 0 {
				wins++
			}
		}
		stats[fmt.Sprintf("d%d", h)] = BacktestStats{
			AvgReturn: sum / float64(len(vals)),
			WinRate:   float64(wins) / float64(len(vals)),
		}
	}

	return &BacktestResult{
		Symbol:      symbol,
		StartDate:   start.Format("2006-01-02"),
		EndDate:     end.Format("2006-01-02"),
		TotalEvents: len(events),
		Events:      events,
		Stats:       stats,
	}, nil
}

func calculateForwardReturns(history []analysisDomain.DailyAnalysisResult, idx int, horizons []int) map[string]float64 {
	out := make(map[string]float64)
	if idx < 0 || idx >= len(history) {
		return out
	}
	base := history[idx]
	if base.Close <= 0 {
		return out
	}
	for _, h := range horizons {
		target := idx + h
		if target >= len(history) {
			continue
		}
		next := history[target]
		if next.Close <= 0 {
			continue
		}
		out[fmt.Sprintf("d%d", h)] = (next.Close / base.Close) - 1
	}
	return out
}
