package analysis

import (
	domain "ai-auto-trade/internal/domain/analysis"
	"testing"
)

func TestMatchConditions_Numeric(t *testing.T) {
	r := domain.DailyAnalysisResult{Score: 80}

	tests := []struct {
		name       string
		conditions []Condition
		logic      BoolLogic
		want       bool
	}{
		{
			name: "Score GTE 70 (True)",
			conditions: []Condition{
				{
					Type: ConditionNumeric,
					Numeric: &NumericCondition{
						Field: FieldScore,
						Op:    OpGTE,
						Value: 70,
					},
				},
			},
			logic: LogicAND,
			want:  true,
		},
		{
			name: "Score GTE 90 (False)",
			conditions: []Condition{
				{
					Type: ConditionNumeric,
					Numeric: &NumericCondition{
						Field: FieldScore,
						Op:    OpGTE,
						Value: 90,
					},
				},
			},
			logic: LogicAND,
			want:  false,
		},
		{
			name: "AND logic (True)",
			conditions: []Condition{
				{
					Type: ConditionNumeric,
					Numeric: &NumericCondition{Field: FieldScore, Op: OpGTE, Value: 70},
				},
				{
					Type: ConditionNumeric,
					Numeric: &NumericCondition{Field: FieldScore, Op: OpLTE, Value: 90},
				},
			},
			logic: LogicAND,
			want:  true,
		},
		{
			name: "OR logic (True)",
			conditions: []Condition{
				{
					Type: ConditionNumeric,
					Numeric: &NumericCondition{Field: FieldScore, Op: OpGTE, Value: 90},
				},
				{
					Type: ConditionNumeric,
					Numeric: &NumericCondition{Field: FieldScore, Op: OpLTE, Value: 85},
				},
			},
			logic: LogicOR,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchConditions(r, tt.conditions, tt.logic); got != tt.want {
				t.Errorf("MatchConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchOne_NilSafely(t *testing.T) {
	r := domain.DailyAnalysisResult{Score: 80}
	
	if matchOne(r, Condition{Type: ConditionNumeric, Numeric: nil}) {
		t.Error("expected false for nil numeric condition")
	}
	if matchOne(r, Condition{Type: ConditionCategory, Category: nil}) {
		t.Error("expected false for nil category condition")
	}
	if matchOne(r, Condition{Type: ConditionTags, Tags: nil}) {
		t.Error("expected false for nil tags condition")
	}
	if matchOne(r, Condition{Type: ConditionSymbols, Symbols: nil}) {
		t.Error("expected false for nil symbols condition")
	}
}
