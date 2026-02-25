package strategy

import (
	"ai-auto-trade/internal/domain/analysis"
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
