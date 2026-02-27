package strategy

import (
	"ai-auto-trade/internal/domain/analysis"
	tradingDomain "ai-auto-trade/internal/domain/trading"
	"testing"
)

func TestEvalPriceReturn(t *testing.T) {
	val := 0.05
	data := analysis.DailyAnalysisResult{
		Return5: &val,
	}

	tests := []struct {
		name    string
		params  map[string]interface{}
		want    float64
		wantErr bool
	}{
		{
			name:   "5-day return exactly threshold",
			params: map[string]interface{}{"days": float64(5), "min": 0.05},
			want:   1.0,
		},
		{
			name:   "5-day return below threshold",
			params: map[string]interface{}{"days": float64(5), "min": 0.06},
			want:   0.0,
		},
		{
			name:   "Continuous scoring fallback",
			params: map[string]interface{}{"days": float64(5)},
			want:   5.0, // 0.05 * 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evalPriceReturn(tt.params, data)
			if (err != nil) != tt.wantErr {
				t.Errorf("evalPriceReturn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("evalPriceReturn() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalVolumeSurge(t *testing.T) {
	vol := 2.0
	data := analysis.DailyAnalysisResult{
		VolumeMultiple: &vol,
	}

	tests := []struct {
		name   string
		params map[string]interface{}
		want   float64
	}{
		{
			name:   "Volume surge triggered",
			params: map[string]interface{}{"min": 1.5},
			want:   1.0,
		},
		{
			name:   "Volume surge not triggered",
			params: map[string]interface{}{"min": 2.5},
			want:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := evalVolumeSurge(tt.params, data)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalAmplitudeSurge(t *testing.T) {
	amp := 0.05
	avg := 0.02
	data := analysis.DailyAnalysisResult{
		Amplitude:      &amp,
		AvgAmplitude20: &avg,
	}

	tests := []struct {
		name   string
		params map[string]interface{}
		want   float64
	}{
		{
			name:   "Amplitude surge triggered",
			params: map[string]interface{}{"min": 2.0},
			want:   1.0,
		},
		{
			name:   "Continuous scoring",
			params: map[string]interface{}{},
			want:   15.0, // (0.05/0.02 - 1.0) * 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := evalAmplitudeSurge(tt.params, data)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTriggered(t *testing.T) {
	s := &ScoringStrategy{
		Threshold: 70,
		EntryRules: []StrategyRule{
			{Condition: Condition{Type: "BASE_SCORE"}, Weight: 100},
		},
	}

	dataTriggered := analysis.DailyAnalysisResult{Score: 80}
	dataNotTriggered := analysis.DailyAnalysisResult{Score: 60}

	triggered, score, _ := s.IsTriggered(dataTriggered)
	if !triggered || score != 80 {
		t.Error("Expected triggered")
	}

	triggered, _, _ = s.IsTriggered(dataNotTriggered)
	if triggered {
		t.Error("Expected not triggered")
	}
}

func TestEvalMADeviation(t *testing.T) {
	dev := 0.03
	data := analysis.DailyAnalysisResult{
		Deviation20: &dev,
	}

	tests := []struct {
		name   string
		params map[string]interface{}
		want   float64
	}{
		{
			name:   "MA deviation triggered",
			params: map[string]interface{}{"min": 0.02},
			want:   1.0,
		},
		{
			name:   "MA deviation not triggered",
			params: map[string]interface{}{"min": 0.04},
			want:   0.0,
		},
		{
			name:   "Continuous scoring",
			params: map[string]interface{}{},
			want:   3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := evalMADeviation(tt.params, data)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalRangePos(t *testing.T) {
	pos := 0.9
	data := analysis.DailyAnalysisResult{
		RangePos20: &pos,
	}

	tests := []struct {
		name   string
		params map[string]interface{}
		want   float64
	}{
		{
			name:   "Range position triggered",
			params: map[string]interface{}{"min": 0.8},
			want:   1.0,
		},
		{
			name:   "Continuous scoring",
			params: map[string]interface{}{},
			want:   4.0, // (0.9 - 0.5) * 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := evalRangePos(tt.params, data)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldExit(t *testing.T) {
	s := &ScoringStrategy{
		Threshold: 70,
		ExitThreshold: 40,
		Risk: tradingDomain.RiskSettings{
			TakeProfitPct: floatPtr(5.0),
			StopLossPct:   floatPtr(2.0),
		},
		EntryRules: []StrategyRule{
			{Condition: Condition{Type: "BASE_SCORE"}, Weight: 100},
		},
	}

	// Case 1: Stop Loss
	dataSL := analysis.DailyAnalysisResult{Close: 97, Score: 80}
	pos := tradingDomain.Position{EntryPrice: 100}
	exit, reason := s.ShouldExit(dataSL, pos)
	if !exit || reason == "" {
		t.Errorf("Expected SL exit, got exit=%v, reason=%s", exit, reason)
	}

	// Case 2: Take Profit
	dataTP := analysis.DailyAnalysisResult{Close: 106, Score: 80}
	exit, reason = s.ShouldExit(dataTP, pos)
	if !exit || reason == "" {
		t.Errorf("Expected TP exit, got exit=%v, reason=%s", exit, reason)
	}

	// Case 3: Signal Decay (Score < Threshold * 0.5)
	// Threshold 70, so decay < 35
	dataDecay := analysis.DailyAnalysisResult{Close: 100, Score: 30}
	exit, reason = s.ShouldExit(dataDecay, pos)
	if !exit || reason == "" {
		t.Errorf("Expected Decay exit, got exit=%v, reason=%s", exit, reason)
	}

	// Case 4: No Exit
	dataNoExit := analysis.DailyAnalysisResult{Close: 101, Score: 80}
	exit, reason = s.ShouldExit(dataNoExit, pos)
	if exit {
		t.Errorf("Expected no exit, got exit=%v, reason=%s", exit, reason)
	}
}

func floatPtr(v float64) *float64 { return &v }

func TestCalculateScore_Weighted(t *testing.T) {
	s := &ScoringStrategy{
		Threshold: 50,
		EntryRules: []StrategyRule{
			{
				Condition: Condition{Type: "BASE_SCORE"},
				Weight:    60,
			},
			{
				Condition: Condition{Type: "PRICE_RETURN", ParamsRaw: []byte(`{"min": 0.01, "days": 5}`)},
				Weight:    40,
			},
		},
	}

	pr := 0.02
	data := analysis.DailyAnalysisResult{
		Score:   80, // Contribution 0.8
		Return5: &pr, // Contribution 1.0 (>= 0.01)
	}

	// Expected: (0.8 * 60 + 1.0 * 40) / 100 * 100 = 88.0
	score, err := s.CalculateScore(data)
	if err != nil {
		t.Fatal(err)
	}

	if score != 88.0 {
		t.Errorf("Expected score 88.0, got %f", score)
	}
}
