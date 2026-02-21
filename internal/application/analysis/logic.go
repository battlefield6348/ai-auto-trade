package analysis

import (
	domain "ai-auto-trade/internal/domain/analysis"
)

// BoolLogic defines the logical operator for combining conditions.
type BoolLogic string

const (
	LogicAND BoolLogic = "AND"
	LogicOR  BoolLogic = "OR"
)

// ConditionType defines the supported types of conditions.
type ConditionType string

const (
	ConditionNumeric  ConditionType = "numeric"
	ConditionCategory ConditionType = "category"
	ConditionTags     ConditionType = "tags"
	ConditionSymbols  ConditionType = "symbols"
)

// FieldName represents common field names used in conditions.
type FieldName string

const (
	FieldScore FieldName = "score"
)

// Op represents operators used in conditions.
type Op string

const (
	OpGTE Op = ">="
	OpLTE Op = "<="
	OpIN  Op = "IN"
)

// NumericCondition defines a numeric comparison rule.
type NumericCondition struct {
	Field FieldName `json:"field"`
	Op    Op        `json:"op"`
	Value float64   `json:"value"`
}

// CategoryCondition defines a categorical membership rule.
type CategoryCondition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}

// TagsCondition defines a tag-based screening rule.
type TagsCondition struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// SymbolsCondition defines a symbol-based filtering rule.
type SymbolsCondition struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// Condition represents a single criteria in a ConditionSet.
type Condition struct {
	Type     ConditionType      `json:"type"`
	Numeric  *NumericCondition  `json:"numeric,omitempty"`
	Category *CategoryCondition `json:"category,omitempty"`
	Tags     *TagsCondition     `json:"tags,omitempty"`
	Symbols  *SymbolsCondition  `json:"symbols,omitempty"`
}

// MatchConditions checks if a record satisfies the given conditions and logic.
func MatchConditions(r domain.DailyAnalysisResult, conditions []Condition, logic BoolLogic) bool {
	if len(conditions) == 0 {
		return false
	}

	results := make([]bool, len(conditions))
	for i, c := range conditions {
		results[i] = matchOne(r, c)
	}

	if logic == LogicOR {
		for _, res := range results {
			if res {
				return true
			}
		}
		return false
	}

	// Default to AND
	for _, res := range results {
		if !res {
			return false
		}
	}
	return true
}

func matchOne(r domain.DailyAnalysisResult, c Condition) bool {
	switch c.Type {
	case ConditionNumeric:
		return matchNumeric(r, c.Numeric)
	case ConditionCategory:
		return matchCategory(r, c.Category)
	case ConditionTags:
		return matchTags(r, c.Tags)
	case ConditionSymbols:
		return matchSymbols(r, c.Symbols)
	}
	return false
}

func matchNumeric(r domain.DailyAnalysisResult, nc *NumericCondition) bool {
	if nc == nil {
		return false
	}
	var val float64
	switch nc.Field {
	case FieldScore:
		val = r.Score
	default:
		// TODO: Implement more fields as needed
		return false
	}

	switch nc.Op {
	case OpGTE:
		return val >= nc.Value
	case OpLTE:
		return val <= nc.Value
	}
	return false
}

func matchCategory(r domain.DailyAnalysisResult, cc *CategoryCondition) bool {
	if cc == nil {
		return false
	}
	// TODO: Implement category matching
	return false
}

func matchTags(r domain.DailyAnalysisResult, tc *TagsCondition) bool {
	if tc == nil {
		return false
	}
	// TODO: Implement tags matching
	return false
}

func matchSymbols(r domain.DailyAnalysisResult, sc *SymbolsCondition) bool {
	if sc == nil {
		return false
	}
	// TODO: Implement symbols matching
	return false
}
