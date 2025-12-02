package analysis

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	domain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

// ScreenerUseCase 提供依條件組合的選股篩選。
type ScreenerUseCase struct {
	repo AnalysisQueryRepository
}

func NewScreenerUseCase(repo AnalysisQueryRepository) *ScreenerUseCase {
	return &ScreenerUseCase{repo: repo}
}

type BoolLogic string

const (
	LogicAND BoolLogic = "AND"
	LogicOR  BoolLogic = "OR"
)

// NumericField 對應分析結果中的數值欄位。
type NumericField string

const (
	FieldClose          NumericField = "close"
	FieldReturn5        NumericField = "return5"
	FieldReturn20       NumericField = "return20"
	FieldReturn60       NumericField = "return60"
	FieldVolumeMultiple NumericField = "volume_multiple"
	FieldDeviation20    NumericField = "deviation20"
	FieldScore          NumericField = "score"
	FieldAmplitude      NumericField = "amplitude"
	FieldRangePos20     NumericField = "range_pos20"
	FieldMA5            NumericField = "ma5"
	FieldMA10           NumericField = "ma10"
	FieldMA20           NumericField = "ma20"
	FieldMA60           NumericField = "ma60"
	FieldAvgAmplitude20 NumericField = "avg_amplitude20"
)

type NumericOp string

const (
	OpGT      NumericOp = "gt"
	OpGTE     NumericOp = "gte"
	OpLT      NumericOp = "lt"
	OpLTE     NumericOp = "lte"
	OpBetween NumericOp = "between"
)

type NumericCondition struct {
	Field NumericField
	Op    NumericOp
	Value float64
	Min   float64
	Max   float64
}

type CategoryField string

const (
	CategoryMarket   CategoryField = "market"
	CategoryIndustry CategoryField = "industry"
)

type CategoryCondition struct {
	Field  CategoryField
	Values []string
}

type TagsCondition struct {
	IncludeAny []domain.Tag
	IncludeAll []domain.Tag
	ExcludeAny []domain.Tag
}

type SymbolCondition struct {
	Include []string
	Exclude []string
}

type ConditionType string

const (
	ConditionNumeric  ConditionType = "numeric"
	ConditionCategory ConditionType = "category"
	ConditionTags     ConditionType = "tags"
	ConditionSymbols  ConditionType = "symbols"
)

type Condition struct {
	Type      ConditionType
	Numeric   *NumericCondition
	Category  *CategoryCondition
	Tags      *TagsCondition
	Symbols   *SymbolCondition
}

type ScreenerInput struct {
	Date       time.Time
	Logic      BoolLogic
	Conditions []Condition
	Sort       SortOption
	Pagination Pagination
}

type ScreenerOutput struct {
	Results []domain.DailyAnalysisResult
	Total   int
	HasMore bool
}

// PresetTemplate 提供預設的選股組合，便於前端或排程使用。
type PresetTemplate struct {
	ID          string
	Name        string
	Description string
	Input       ScreenerInput
}

// PresetTemplates 回傳內建模板集合。
func PresetTemplates(date time.Time) []PresetTemplate {
	return []PresetTemplate{
		{
			ID:          "short_term_strong",
			Name:        "短期強勢股",
			Description: "近5日報酬佳、量能放大、位於區間上半段",
			Input: ScreenerInput{
				Date:  date,
				Logic: LogicAND,
				Conditions: []Condition{
					numericCond(FieldReturn5, OpGTE, 0.05),
					numericCond(FieldRangePos20, OpGTE, 0.6),
					numericCond(FieldVolumeMultiple, OpGTE, 1.5),
					{Type: ConditionTags, Tags: &TagsCondition{IncludeAny: []domain.Tag{domain.TagShortTermStrong, domain.TagVolumeSurge}}},
				},
				Sort: SortOption{Field: SortScore, Desc: true},
			},
		},
		{
			ID:          "volume_surge",
			Name:        "量能放大股",
			Description: "當日量能明顯放大，搭配正向報酬",
			Input: ScreenerInput{
				Date:  date,
				Logic: LogicAND,
				Conditions: []Condition{
					numericCond(FieldVolumeMultiple, OpGTE, 2.0),
					numericCond(FieldReturn5, OpGTE, 0.0),
				},
				Sort: SortOption{Field: SortVolumeMultiple, Desc: true},
			},
		},
		{
			ID:          "bullish_breakout",
			Name:        "多頭排列突破",
			Description: "靠近前高且整體分數佳，模擬多頭排列突破情境",
			Input: ScreenerInput{
				Date:  date,
				Logic: LogicAND,
				Conditions: []Condition{
					numericCond(FieldRangePos20, OpGTE, 0.9),
					numericCond(FieldScore, OpGTE, 65),
					numericCond(FieldDeviation20, OpGTE, 0.0),
					numericCond(FieldReturn20, OpGTE, 0.0),
				},
				Sort: SortOption{Field: SortScore, Desc: true},
			},
		},
		{
			ID:          "low_vol_base",
			Name:        "低波動蓄勢",
			Description: "近 20 日波動度低，價位貼近均線，等待突破",
			Input: ScreenerInput{
				Date:  date,
				Logic: LogicAND,
				Conditions: []Condition{
					numericCond(FieldAvgAmplitude20, OpLTE, 0.02),
					numericBetween(FieldDeviation20, -0.02, 0.02),
					{Type: ConditionTags, Tags: &TagsCondition{ExcludeAny: []domain.Tag{domain.TagHighVolatility}}},
				},
				Sort: SortOption{Field: SortScore, Desc: true},
			},
		},
		{
			ID:          "near_high",
			Name:        "接近前高",
			Description: "收盤價接近近20日高點，且分數較佳",
			Input: ScreenerInput{
				Date:  date,
				Logic: LogicAND,
				Conditions: []Condition{
					numericCond(FieldRangePos20, OpGTE, 0.8),
					numericCond(FieldScore, OpGTE, 60),
				},
				Sort: SortOption{Field: SortRangePos20, Desc: true},
			},
		},
	}
}

// Run 執行選股，依條件組合過濾當日分析結果。
func (u *ScreenerUseCase) Run(ctx context.Context, input ScreenerInput) (ScreenerOutput, error) {
	var out ScreenerOutput

	if input.Date.IsZero() {
		return out, fmt.Errorf("date is required")
	}
	if input.Logic == "" {
		input.Logic = LogicAND
	}

	queryInput := QueryByDateInput{
		Date: input.Date,
		Filter: QueryFilter{
			OnlySuccess: true,
		},
		Sort: input.Sort,
		Pagination: Pagination{
			Offset: 0,
			Limit:  maxLimit,
		},
	}

	// 嘗試把部分條件下推到儲存層。
	for _, c := range input.Conditions {
		switch c.Type {
		case ConditionCategory:
			if c.Category == nil {
				continue
			}
			switch c.Category.Field {
			case CategoryMarket:
				for _, v := range c.Category.Values {
					mv := dataingestion.Market(v)
					queryInput.Filter.Markets = append(queryInput.Filter.Markets, mv)
				}
			case CategoryIndustry:
				queryInput.Filter.Industries = append(queryInput.Filter.Industries, c.Category.Values...)
			}
		case ConditionTags:
			if c.Tags == nil {
				continue
			}
			queryInput.Filter.TagsAny = append(queryInput.Filter.TagsAny, c.Tags.IncludeAny...)
			queryInput.Filter.TagsAll = append(queryInput.Filter.TagsAll, c.Tags.IncludeAll...)
		case ConditionNumeric:
			if c.Numeric == nil {
				continue
			}
			switch c.Numeric.Field {
			case FieldScore:
				applyNumericToRange(c.Numeric, &queryInput.Filter.ScoreMin, &queryInput.Filter.ScoreMax)
			case FieldReturn5:
				applyNumericToRange(c.Numeric, &queryInput.Filter.Return5Min, &queryInput.Filter.Return5Max)
			case FieldReturn20:
				applyNumericToRange(c.Numeric, &queryInput.Filter.Return20Min, &queryInput.Filter.Return20Max)
			case FieldVolumeMultiple:
				applyNumericToRange(c.Numeric, &queryInput.Filter.VolumeMultipleMin, &queryInput.Filter.VolumeMultipleMax)
			}
		case ConditionSymbols:
			if c.Symbols != nil {
				queryInput.Filter.Symbols = append(queryInput.Filter.Symbols, c.Symbols.Include...)
			}
		}
	}

	baseResults, _, err := u.repo.FindByDate(ctx, queryInput.Date, queryInput.Filter, queryInput.Sort, queryInput.Pagination)
	if err != nil {
		return out, err
	}

	filtered := make([]domain.DailyAnalysisResult, 0, len(baseResults))
	for _, r := range baseResults {
		if matchConditions(r, input.Conditions, input.Logic) {
			filtered = append(filtered, r)
		}
	}

	applySort(filtered, input.Sort)

	offset := input.Pagination.Offset
	limit := input.Pagination.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}

	if offset > len(filtered) {
		offset = len(filtered)
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	out.Total = len(filtered)
	out.HasMore = end < len(filtered)
	out.Results = filtered[offset:end]
	return out, nil
}

func applyNumericToRange(cond *NumericCondition, minPtr **float64, maxPtr **float64) {
	switch cond.Op {
	case OpGT:
		val := cond.Value
		*minPtr = &val
	case OpGTE:
		val := cond.Value
		*minPtr = &val
	case OpLT:
		val := cond.Value
		*maxPtr = &val
	case OpLTE:
		val := cond.Value
		*maxPtr = &val
	case OpBetween:
		minVal := cond.Min
		maxVal := cond.Max
		*minPtr = &minVal
		*maxPtr = &maxVal
	}
}

func matchConditions(r domain.DailyAnalysisResult, conditions []Condition, logic BoolLogic) bool {
	if len(conditions) == 0 {
		return true
	}

	matches := func(c Condition) bool {
		switch c.Type {
		case ConditionNumeric:
			return evalNumeric(r, c.Numeric)
		case ConditionCategory:
			return evalCategory(r, c.Category)
		case ConditionTags:
			return evalTags(r, c.Tags)
		case ConditionSymbols:
			return evalSymbols(r, c.Symbols)
		default:
			return false
		}
	}

	if logic == LogicOR {
		for _, c := range conditions {
			if matches(c) {
				return true
			}
		}
		return false
	}

	for _, c := range conditions {
		if !matches(c) {
			return false
		}
	}
	return true
}

func evalNumeric(r domain.DailyAnalysisResult, c *NumericCondition) bool {
	if c == nil {
		return true
	}
	valPtr := numericValue(r, c.Field)
	if valPtr == nil {
		return false
	}
	v := *valPtr
	switch c.Op {
	case OpGT:
		return v > c.Value
	case OpGTE:
		return v >= c.Value
	case OpLT:
		return v < c.Value
	case OpLTE:
		return v <= c.Value
	case OpBetween:
		return v >= c.Min && v <= c.Max
	default:
		return false
	}
}

func numericValue(r domain.DailyAnalysisResult, field NumericField) *float64 {
	switch field {
	case FieldClose:
		return &r.Close
	case FieldReturn5:
		return r.Return5
	case FieldReturn20:
		return r.Return20
	case FieldReturn60:
		return r.Return60
	case FieldVolumeMultiple:
		return r.VolumeMultiple
	case FieldDeviation20:
		return r.Deviation20
	case FieldScore:
		return &r.Score
	case FieldAmplitude:
		return r.Amplitude
	case FieldRangePos20:
		return r.RangePos20
	case FieldMA5:
		return r.MA5
	case FieldMA10:
		return r.MA10
	case FieldMA20:
		return r.MA20
	case FieldMA60:
		return r.MA60
	case FieldAvgAmplitude20:
		return r.AvgAmplitude20
	default:
		return nil
	}
}

func evalCategory(r domain.DailyAnalysisResult, c *CategoryCondition) bool {
	if c == nil {
		return true
	}
	switch c.Field {
	case CategoryMarket:
		for _, v := range c.Values {
			if string(r.Market) == v {
				return true
			}
		}
		return false
	case CategoryIndustry:
		for _, v := range c.Values {
			if strings.EqualFold(r.Industry, v) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func evalTags(r domain.DailyAnalysisResult, c *TagsCondition) bool {
	if c == nil {
		return true
	}

	has := func(tag domain.Tag) bool {
		for _, t := range r.Tags {
			if t == tag {
				return true
			}
		}
		return false
	}

	if len(c.IncludeAll) > 0 {
		for _, t := range c.IncludeAll {
			if !has(t) {
				return false
			}
		}
	}

	if len(c.IncludeAny) > 0 {
		ok := false
		for _, t := range c.IncludeAny {
			if has(t) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if len(c.ExcludeAny) > 0 {
		for _, t := range c.ExcludeAny {
			if has(t) {
				return false
			}
		}
	}

	return true
}

func evalSymbols(r domain.DailyAnalysisResult, c *SymbolCondition) bool {
	if c == nil {
		return true
	}
	if len(c.Include) > 0 {
		found := false
		for _, s := range c.Include {
			if r.Symbol == s {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(c.Exclude) > 0 {
		for _, s := range c.Exclude {
			if r.Symbol == s {
				return false
			}
		}
	}
	return true
}

func applySort(results []domain.DailyAnalysisResult, sortOpt SortOption) {
	field := sortOpt.Field
	if field == "" {
		field = SortScore
		sortOpt.Field = field
		sortOpt.Desc = true
	}

	slices.SortFunc(results, func(a, b domain.DailyAnalysisResult) int {
		var av, bv float64
		switch field {
		case SortScore:
			av, bv = a.Score, b.Score
		case SortReturn5:
			av, bv = deref(a.Return5), deref(b.Return5)
		case SortReturn20:
			av, bv = deref(a.Return20), deref(b.Return20)
		case SortVolumeMultiple:
			av, bv = deref(a.VolumeMultiple), deref(b.VolumeMultiple)
		case SortChangeRate:
			av, bv = a.ChangeRate, b.ChangeRate
		case SortRangePos20:
			av, bv = deref(a.RangePos20), deref(b.RangePos20)
		default:
			av, bv = a.Score, b.Score
		}
		if sortOpt.Desc {
			if av > bv {
				return -1
			}
			if av < bv {
				return 1
			}
			return strings.Compare(a.Symbol, b.Symbol)
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
		return strings.Compare(a.Symbol, b.Symbol)
	})
}

func deref(ptr *float64) float64 {
	if ptr == nil {
		return -1e9
	}
	return *ptr
}

// helpers
func numericCond(field NumericField, op NumericOp, value float64) Condition {
	return Condition{
		Type: ConditionNumeric,
		Numeric: &NumericCondition{
			Field: field,
			Op:    op,
			Value: value,
		},
	}
}

func numericBetween(field NumericField, min, max float64) Condition {
	return Condition{
		Type: ConditionNumeric,
		Numeric: &NumericCondition{
			Field: field,
			Op:    OpBetween,
			Min:   min,
			Max:   max,
		},
	}
}
