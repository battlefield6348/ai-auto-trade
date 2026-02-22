package strategy

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
	tradingDomain "ai-auto-trade/internal/domain/trading"
)

type BacktestResult struct {
	Symbol      string                   `json:"symbol"`
	StartDate   string                   `json:"start_date"`
	EndDate     string                   `json:"end_date"`
	TotalEvents int                      `json:"total_events"`
	Events      []BacktestEvent          `json:"events"`
	Stats       map[string]BacktestStats `json:"stats"`
	Trades      []BacktestTrade          `json:"trades"`
	Summary     SimulationSummary       `json:"summary"`
}

type BacktestTrade struct {
	EntryDate  string  `json:"entry_date"`
	EntryPrice float64 `json:"entry_price"`
	ExitDate   string  `json:"exit_date"`
	ExitPrice  float64 `json:"exit_price"`
	PnL        float64 `json:"pnl"`
	PnLPct     float64 `json:"pnl_pct"`
	Reason     string  `json:"reason"`
}

type SimulationSummary struct {
	TotalTrades int     `json:"total_trades"`
	TotalReturn float64 `json:"total_return"`
	WinRate     float64 `json:"win_rate"`
}

type BacktestEvent struct {
	TradeDate      string             `json:"trade_date"`
	ClosePrice     float64            `json:"close_price"`
	ChangePercent  float64            `json:"change_percent"`
	TotalScore     float64            `json:"total_score"`
	EntryScore     float64            `json:"entry_score"`
	ExitScore      float64            `json:"exit_score"`
	IsTriggered    bool               `json:"is_triggered"`
	Return5d       *float64           `json:"return_5d"`
	ForwardReturns map[string]float64 `json:"forward_returns,omitempty"`
}

type BacktestStats struct {
	AvgReturn float64 `json:"avg_return"`
	WinRate   float64 `json:"win_rate"`
}

type DataProvider interface {
	FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error)
}

type BacktestUseCase struct {
	db       strategyDomain.DBQueryer
	dataProv DataProvider
}

func NewBacktestUseCase(db strategyDomain.DBQueryer, dataProv DataProvider) *BacktestUseCase {
	return &BacktestUseCase{db: db, dataProv: dataProv}
}

func (u *BacktestUseCase) Execute(ctx context.Context, slug string, symbol string, start, end time.Time, horizons []int) (*BacktestResult, error) {
	if u.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	// Defensive check for typed nil
	if reflect.ValueOf(u.db).IsNil() {
		return nil, fmt.Errorf("database storage not initialized")
	}
	// 1. Load Strategy
	s, err := strategyDomain.LoadScoringStrategyBySlug(ctx, u.db, slug)
	if err != nil {
		return nil, fmt.Errorf("load strategy failed: %w", err)
	}

	return u.ExecuteWithStrategy(ctx, s, symbol, start, end, horizons)
}

func (u *BacktestUseCase) ExecuteWithStrategy(ctx context.Context, s *strategyDomain.ScoringStrategy, symbol string, start, end time.Time, horizons []int) (*BacktestResult, error) {
	// 2. Load History
	history, err := u.dataProv.FindHistory(ctx, symbol, s.Timeframe, &start, &end, 5000, true)
	if err != nil {
		return nil, fmt.Errorf("fetch history failed: %w", err)
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].TradeDate.Before(history[j].TradeDate)
	})

	if len(horizons) == 0 {
		horizons = []int{3, 5, 10}
	}
	var events []BacktestEvent
	retStats := make(map[int][]float64)

	// Simulation state
	var trades []BacktestTrade
	var currentPosition *BacktestTrade
	totalReturn := 1.0

	for idx, res := range history {
		triggered, score, err := s.IsTriggered(res)
		if err != nil {
			continue
		}

		exitScore, _ := s.CalculateScoreForRules(s.ExitRules, res)
		
		// Record all events for accurate charting
		var forward map[string]float64
		if triggered {
			forward = calculateForwardReturns(history, idx, horizons)
			for _, h := range horizons {
				if val, ok := forward[fmt.Sprintf("d%d", h)]; ok {
					retStats[h] = append(retStats[h], val)
				}
			}
		}

		events = append(events, BacktestEvent{
			TradeDate:      res.TradeDate.Format("2006-01-02"),
			ClosePrice:     res.Close,
			ChangePercent:  res.ChangeRate,
			TotalScore:     score,
			EntryScore:     score,
			ExitScore:      exitScore,
			IsTriggered:    triggered,
			Return5d:       res.Return5,
			ForwardReturns: forward,
		})

		// Simulation Logic (Sequential)
		if currentPosition == nil {
			if triggered {
				// Open position at next day open (approximated by current close for simplicity, or we could use next day open if available)
				// For simplicity in this backtester, we use current close as entry price.
				currentPosition = &BacktestTrade{
					EntryDate:  res.TradeDate.Format("2006-01-02"),
					EntryPrice: res.Close,
				}
			}
		} else {
			// Check Exit
			// Check Exit using unified domain logic
			dummyPos := tradingDomain.Position{
				EntryPrice: currentPosition.EntryPrice,
				EntryDate:  start, // Not strictly used for TP/SL but good for completeness
			}
			exitTriggered, reason := s.ShouldExit(res, dummyPos)
			if exitTriggered {
				currentPosition.Reason = reason
			}

			if exitTriggered {
				currentPosition.ExitDate = res.TradeDate.Format("2006-01-02")
				currentPosition.ExitPrice = res.Close
				
				// Apply 0.1% slippage/fee on exit
				exitPriceWithFee := currentPosition.ExitPrice * 0.999
				
				currentPosition.PnL = exitPriceWithFee - currentPosition.EntryPrice
				currentPosition.PnLPct = (exitPriceWithFee / currentPosition.EntryPrice) - 1.0
				
				trades = append(trades, *currentPosition)
				totalReturn *= (1.0 + currentPosition.PnLPct)
				currentPosition = nil
			}
		}
	}
	
	// Force close last position if still open at end of data
	if currentPosition != nil && len(history) > 0 {
		last := history[len(history)-1]
		currentPosition.ExitDate = last.TradeDate.Format("2006-01-02")
		currentPosition.ExitPrice = last.Close
		currentPosition.PnL = currentPosition.ExitPrice - currentPosition.EntryPrice
		currentPosition.PnLPct = (currentPosition.ExitPrice / currentPosition.EntryPrice) - 1.0
		currentPosition.Reason = "回測結束前尚未出場 (Simulation End)"
		
		trades = append(trades, *currentPosition)
		totalReturn *= (1.0 + currentPosition.PnLPct)
		currentPosition = nil
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

	summary := SimulationSummary{
		TotalTrades: len(trades),
		TotalReturn: (totalReturn - 1.0) * 100, // as percentage
	}
	if len(trades) > 0 {
		wins := 0
		for _, t := range trades {
			if t.PnLPct > 0 {
				wins++
			}
		}
		summary.WinRate = float64(wins) / float64(len(trades)) * 100
	}

	return &BacktestResult{
		Symbol:      symbol,
		StartDate:   start.Format("2006-01-02"),
		EndDate:     end.Format("2006-01-02"),
		TotalEvents: len(events),
		Events:      events,
		Stats:       stats,
		Trades:      trades,
		Summary:     summary,
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
