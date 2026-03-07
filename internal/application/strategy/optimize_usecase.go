package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	strategyDomain "ai-auto-trade/internal/domain/strategy"
	tradingDomain "ai-auto-trade/internal/domain/trading"
)

type OptimizeRequest struct {
	Symbol    string    `json:"symbol"`
	Days      int       `json:"days"`
	SaveTop   bool      `json:"save_top"`
	CreatedBy string    `json:"created_by"`
}

type OptimizeResult struct {
	BestStrategy *strategyDomain.ScoringStrategy `json:"best_strategy"`
	TotalReturn  float64                         `json:"total_return"`
	WinRate      float64                         `json:"win_rate"`
	TotalTrades  int                             `json:"total_trades"`
}

type OptimizeScoringStrategyUseCase struct {
	backtestUC *BacktestUseCase
	saveUC     *SaveScoringStrategyUseCase
}

func NewOptimizeScoringStrategyUseCase(bt *BacktestUseCase, sv *SaveScoringStrategyUseCase) *OptimizeScoringStrategyUseCase {
	return &OptimizeScoringStrategyUseCase{
		backtestUC: bt,
		saveUC:     sv,
	}
}

func (u *OptimizeScoringStrategyUseCase) Execute(ctx context.Context, req OptimizeRequest) (*OptimizeResult, error) {
	if req.Days == 0 {
		req.Days = 90
	}
	if req.Symbol == "" {
		req.Symbol = "BTCUSDT"
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -req.Days)
	// We need extra data for MA calculations, fetch more history
	fetchStartTime := startTime.AddDate(0, 0, -30)

	history, err := u.backtestUC.dataProv.FindHistory(ctx, req.Symbol, "1d", &fetchStartTime, &endTime, 500, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	if len(history) < 30 {
		return nil, fmt.Errorf("insufficient data for optimization (got %d days)", len(history))
	}

	// Permutation parameters
	entryThresholds := []float64{60, 70, 75}
	exitThresholds := []float64{40, 50, 60}
	takeProfits := []float64{0.05, 0.10, 0.15}
	stopLosses := []float64{-0.03, -0.05, -0.08}
	
	var results []optimizerResultInternal

	log.Printf("[Optimizer] Starting optimization for %s with %d permutations", req.Symbol, len(entryThresholds)*len(exitThresholds)*len(takeProfits)*len(stopLosses))

	for _, eth := range entryThresholds {
		for _, xth := range exitThresholds {
			for _, tp := range takeProfits {
				for _, sl := range stopLosses {
					strat := &strategyDomain.ScoringStrategy{
						Name:          fmt.Sprintf("Optimized %s", req.Symbol),
						Timeframe:     "1d",
						BaseSymbol:    req.Symbol,
						Threshold:     eth,
						ExitThreshold: xth,
						Risk: tradingDomain.RiskSettings{
							StopLossPct:   &sl,
							TakeProfitPct: &tp,
						},
					}
					
					// Basic rules matching the optimizer tool
					strat.EntryRules = u.buildStandardRules(1.0, 1.2, 1.0, 0.8, "entry")
					strat.ExitRules = u.buildStandardRules(-1.5, 0.8, -0.5, 0.8, "exit")

					res, err := u.backtestUC.ExecuteWithStrategy(ctx, strat, req.Symbol, startTime, endTime, []int{3, 5, 10})
					if err != nil || res == nil || res.Summary.TotalTrades == 0 {
						continue
					}

					results = append(results, optimizerResultInternal{
						strat:       strat,
						totalReturn: res.Summary.TotalReturn,
						winRate:     res.Summary.WinRate,
						totalTrades: res.Summary.TotalTrades,
					})
				}
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no profitable strategies found in search space")
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].totalReturn > results[j].totalReturn
	})

	best := results[0]

	if req.SaveTop {
		saveInput := SaveScoringStrategyInput{
			UserID:        req.CreatedBy,
			Name:          fmt.Sprintf("Best Optimized %s - %s", req.Symbol, time.Now().Format("0102")),
			Slug:          fmt.Sprintf("optimized-%s-%s", req.Symbol, time.Now().Format("0102")),
			BaseSymbol:    req.Symbol,
			Timeframe:     "1d",
			Threshold:     best.strat.Threshold,
			ExitThreshold: best.strat.ExitThreshold,
		}
		
		// Convert strategyDomain.StrategyRule to SaveRuleInput
		for _, r := range best.strat.EntryRules {
			params := make(map[string]interface{})
			json.Unmarshal(r.Condition.ParamsRaw, &params)
			saveInput.Rules = append(saveInput.Rules, SaveRuleInput{
				Type:     r.Condition.Type,
				Weight:   r.Weight,
				Params:   params,
				RuleType: "entry",
			})
		}
		for _, r := range best.strat.ExitRules {
			params := make(map[string]interface{})
			json.Unmarshal(r.Condition.ParamsRaw, &params)
			saveInput.Rules = append(saveInput.Rules, SaveRuleInput{
				Type:     r.Condition.Type,
				Weight:   r.Weight,
				Params:   params,
				RuleType: "exit",
			})
		}

		if err := u.saveUC.Execute(ctx, saveInput); err != nil {
			log.Printf("[Optimizer] Failed to save best strategy: %v", err)
		}
	}

	return &OptimizeResult{
		BestStrategy: best.strat,
		TotalReturn:  best.totalReturn,
		WinRate:      best.winRate,
		TotalTrades:  best.totalTrades,
	}, nil
}

type optimizerResultInternal struct {
	strat       *strategyDomain.ScoringStrategy
	totalReturn float64
	winRate     float64
	totalTrades int
}

func (u *OptimizeScoringStrategyUseCase) buildStandardRules(changeMin, volMin, maMin, rangeMin float64, ruleType string) []strategyDomain.StrategyRule {
	var rules []strategyDomain.StrategyRule

	// Base Score Weight
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   50.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type: "BASE_SCORE",
		},
	})

	// Price Return
	p1, _ := json.Marshal(map[string]interface{}{"days": 1, "min": changeMin})
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   20.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type:      "PRICE_RETURN",
			ParamsRaw: p1,
		},
	})

	// Volume Surge
	p2, _ := json.Marshal(map[string]interface{}{"min": volMin})
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   15.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type:      "VOLUME_SURGE",
			ParamsRaw: p2,
		},
	})

	// MA Deviation
	p3, _ := json.Marshal(map[string]interface{}{"ma": 20, "min": maMin})
	rules = append(rules, strategyDomain.StrategyRule{
		Weight:   15.0,
		RuleType: ruleType,
		Condition: strategyDomain.Condition{
			Type:      "MA_DEVIATION",
			ParamsRaw: p3,
		},
	})

	return rules
}
