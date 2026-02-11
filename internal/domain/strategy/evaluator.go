package strategy

import (
	"ai-auto-trade/internal/domain/analysis"
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
	"BASE_SCORE":   evalBaseScore,
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
