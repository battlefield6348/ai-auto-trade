package strategy

import (
	"ai-auto-trade/internal/domain/analysis"
	tradingDomain "ai-auto-trade/internal/domain/trading"
	"fmt"
)

// ConditionEvaluator is a function that takes a condition's raw params and analysis data,
// and returns a score contribution (usually normalized before weighting).
type ConditionEvaluator func(params map[string]interface{}, data analysis.DailyAnalysisResult) (float64, error)

// EvaluatorRegistry maps condition types to their respective evaluator functions.
var EvaluatorRegistry = map[string]ConditionEvaluator{
	"PRICE_RETURN": evalPriceReturn,
	"VOLUME_SURGE": evalVolumeSurge,
	"MA_DEVIATION": evalMADeviation,
	"RANGE_POS":    evalRangePos,
	"AMPLITUDE_SURGE": evalAmplitudeSurge,
	"BASE_SCORE":   evalBaseScore,
}

// evalAmplitudeSurge computes score based on current amplitude relative to average amplitude.
// Params: {"min": 1.5}
func evalAmplitudeSurge(params map[string]interface{}, data analysis.DailyAnalysisResult) (float64, error) {
	if data.Amplitude == nil || data.AvgAmplitude20 == nil || *data.AvgAmplitude20 == 0 {
		return 0, nil
	}
	ratio := *data.Amplitude / *data.AvgAmplitude20
	threshold, hasThreshold := params["min"].(float64)
	if hasThreshold {
		if ratio >= threshold {
			return 1.0, nil
		}
		return 0, nil
	}
	return (ratio - 1.0) * 10, nil
}

// evalPriceReturn computes score based on price return over N days.
// Params: {"days": 5, "min": 0.05}
func evalPriceReturn(params map[string]interface{}, data analysis.DailyAnalysisResult) (float64, error) {
	days, _ := params["days"].(float64)
	threshold, hasThreshold := params["min"].(float64)
	var val *float64
	switch int(days) {
	case 5:
		val = data.Return5
	case 20:
		val = data.Return20
	case 60:
		val = data.Return60
	default:
		// Fallback to 1-day change if days not specified or invalid
		v := data.ChangeRate
		val = &v
	}
	if val == nil {
		return 0, nil
	}
	if hasThreshold {
		if *val >= threshold {
			return 1.0, nil
		}
		return 0, nil
	}
	// Legacy continuous scoring
	return *val * 100, nil
}

// evalVolumeSurge computes score based on volume relative to average.
// Params: {"min": 1.5}
func evalVolumeSurge(params map[string]interface{}, data analysis.DailyAnalysisResult) (float64, error) {
	if data.VolumeMultiple == nil {
		return 0, nil
	}
	threshold, hasThreshold := params["min"].(float64)
	if hasThreshold {
		if *data.VolumeMultiple >= threshold {
			return 1.0, nil
		}
		return 0, nil
	}
	// Legacy continuous scoring
	return (*data.VolumeMultiple - 1.0) * 10, nil
}

// evalMADeviation computes score based on deviation from moving average.
// Params: {"ma": 20, "min": 0.02}
func evalMADeviation(params map[string]interface{}, data analysis.DailyAnalysisResult) (float64, error) {
	if data.Deviation20 == nil {
		return 0, nil
	}
	threshold, hasThreshold := params["min"].(float64)
	if hasThreshold {
		if *data.Deviation20 >= threshold {
			return 1.0, nil
		}
		return 0, nil
	}
	// Legacy continuous scoring
	return *data.Deviation20 * 100, nil
}

// evalRangePos computes score based on where the price is in its N-day range.
// Params: {"days": 20, "min": 0.8}
func evalRangePos(params map[string]interface{}, data analysis.DailyAnalysisResult) (float64, error) {
	if data.RangePos20 == nil {
		return 0, nil
	}
	threshold, hasThreshold := params["min"].(float64)
	if hasThreshold {
		if *data.RangePos20 >= threshold {
			return 1.0, nil
		}
		return 0, nil
	}
	// Legacy continuous scoring
	return (*data.RangePos20 - 0.5) * 10, nil
}

// evalBaseScore extracts the pre-calculated score from analysis result.
// Params: none required, but maybe weighting.
func evalBaseScore(params map[string]interface{}, data analysis.DailyAnalysisResult) (float64, error) {
	// data.Score is 0-100? No, it's float64. Is it normalized?
	// Assuming data.Score is 0-100.
	return data.Score / 100.0, nil
}

// CalculateScore executes all rules in a strategy (defaults to EntryRules) and returns the total score.
func (s *ScoringStrategy) CalculateScore(data analysis.DailyAnalysisResult) (float64, error) {
	return s.CalculateScoreForRules(s.EntryRules, data)
}

// CalculateScoreForRules executes a specific set of rules and returns the total score.
func (s *ScoringStrategy) CalculateScoreForRules(rules []StrategyRule, data analysis.DailyAnalysisResult) (float64, error) {
	totalScore := 0.0
	totalWeight := 0.0

	for _, rule := range rules {
		evaluator, ok := EvaluatorRegistry[rule.Condition.Type]
		if !ok {
			continue
		}

		params, err := rule.Condition.ParseParams()
		if err != nil {
			return 0, fmt.Errorf("failed to parse params for condition %s: %w", rule.Condition.ID, err)
		}

		contribution, err := evaluator(params, data)
		if err != nil {
			return 0, fmt.Errorf("evaluation error for rule %s: %w", rule.Condition.Type, err)
		}

		totalWeight += rule.Weight
		totalScore += contribution * rule.Weight
	}

	if totalWeight > 0 {
		return (totalScore / totalWeight) * 100.0, nil
	}

	return totalScore, nil
}

// IsTriggered checks if the entry score exceeds the strategy's threshold.
func (s *ScoringStrategy) IsTriggered(data analysis.DailyAnalysisResult) (bool, float64, error) {
	score, err := s.CalculateScoreForRules(s.EntryRules, data)
	if err != nil {
		return false, 0, err
	}
	return score >= s.Threshold, score, nil
}

// IsExitTriggered checks if the exit rules are satisfied.
func (s *ScoringStrategy) IsExitTriggered(data analysis.DailyAnalysisResult) (bool, float64, error) {
	if len(s.ExitRules) == 0 {
		return false, 0, nil
	}
	score, err := s.CalculateScoreForRules(s.ExitRules, data)
	if err != nil {
		return false, 0, err
	}
	// Note: User requires exit when score drops below threshold
	return score < s.ExitThreshold, score, nil
}

// ShouldExit evaluates all exit conditions including TP/SL, signal decay, and custom rules.
func (s *ScoringStrategy) ShouldExit(data analysis.DailyAnalysisResult, pos tradingDomain.Position) (bool, string) {
	// 1. Fixed Take Profit and Stop Loss
	sl := -0.02
	if s.Risk.StopLossPct != nil {
		sl = -(*s.Risk.StopLossPct)
		if sl > 0 {
			sl = -sl
		}
	}
	tp := 0.05
	if s.Risk.TakeProfitPct != nil {
		tp = *s.Risk.TakeProfitPct
	}

	if pos.EntryPrice > 0 {
		change := (data.Close - pos.EntryPrice) / pos.EntryPrice
		if change <= sl {
			return true, fmt.Sprintf("止損 (%.1f%%)", sl*100)
		}
		if change >= tp {
			return true, fmt.Sprintf("止盈 (%.1f%%)", tp*100)
		}
	}

	// 2. AI Signal Decay (Entry score drops below 50% of entry threshold)
	score, _ := s.CalculateScore(data)
	if score < (s.Threshold * 0.5) {
		return true, fmt.Sprintf("AI信號轉弱 (分數 %.1f < %.1f)", score, s.Threshold*0.5)
	}

	// 3. Custom Exit Rules
	if triggered, exitScore, _ := s.IsExitTriggered(data); triggered {
		return true, fmt.Sprintf("觸發策略出場條件 (分數 %.1f < %.1f)", exitScore, s.ExitThreshold)
	}

	return false, ""
}
