package analysis

import (
	"context"
	"encoding/csv"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	domain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

// AnalysisQueryRepository 定義查詢介面，具體儲存層自行實作。
type AnalysisQueryRepository interface {
	FindByDate(ctx context.Context, date time.Time, filter QueryFilter, sort SortOption, pagination Pagination) ([]domain.DailyAnalysisResult, int, error)
	FindHistory(ctx context.Context, symbol string, from, to *time.Time, limit int, onlySuccess bool) ([]domain.DailyAnalysisResult, error)
	Get(ctx context.Context, symbol string, date time.Time) (domain.DailyAnalysisResult, error)
}

// QueryFilter 為列表查詢的過濾條件。
type QueryFilter struct {
	Markets           []dataingestion.Market
	Industries        []string
	Symbols           []string
	ScoreMin          *float64
	ScoreMax          *float64
	Return5Min        *float64
	Return5Max        *float64
	Return20Min       *float64
	Return20Max       *float64
	VolumeMultipleMin *float64
	VolumeMultipleMax *float64
	TagsAny           []domain.Tag // 符合任一
	TagsAll           []domain.Tag // 必須全包含
	OnlySuccess       bool
}

// SortField 定義列表排序欄位。
type SortField string

const (
	SortScore          SortField = "score"
	SortReturn5        SortField = "return5"
	SortReturn20       SortField = "return20"
	SortVolumeMultiple SortField = "volume_multiple"
	SortChangeRate     SortField = "change_rate"
	SortRangePos20     SortField = "range_pos20"
)

// SortOption 指定排序欄位與方向。
type SortOption struct {
	Field SortField
	Desc  bool
}

// Pagination 控制分頁。
type Pagination struct {
	Offset int
	Limit  int
}

const (
	defaultLimit = 100
	maxLimit     = 1000
)

// QueryByDateInput 對應「依日期查詢列表」。
type QueryByDateInput struct {
	Date       time.Time
	Filter     QueryFilter
	Sort       SortOption
	Pagination Pagination
}

// QueryByDateOutput 返回結果與筆數資訊。
type QueryByDateOutput struct {
	Results []domain.DailyAnalysisResult
	Total   int
	HasMore bool
}

// QueryHistoryInput 對應「依股票查詢歷史」。
type QueryHistoryInput struct {
	Symbol      string
	From        *time.Time
	To          *time.Time
	Limit       int // 若未指定則由實作決定，預設 120
	OnlySuccess bool
}

// QueryDetailInput 對應單筆明細。
type QueryDetailInput struct {
	Symbol string
	Date   time.Time
}

// ExportDailyStrongInput 對應「當日強勢股清單」匯出。
type ExportDailyStrongInput struct {
	Date   time.Time
	Filter QueryFilter
	Sort   SortOption
	Limit  int // 可限制匯出筆數，空值則使用列表預設
}

// QueryUseCase 聚合查詢與匯出行為。
type QueryUseCase struct {
	repo AnalysisQueryRepository
}

// NewQueryUseCase 建立分析結果查詢用例，包裝各種查詢與匯出行為。
func NewQueryUseCase(repo AnalysisQueryRepository) *QueryUseCase {
	return &QueryUseCase{repo: repo}
}

// QueryByDate 依日期查詢列表，提供分頁與排序。
func (u *QueryUseCase) QueryByDate(ctx context.Context, input QueryByDateInput) (QueryByDateOutput, error) {
	var out QueryByDateOutput

	if input.Date.IsZero() {
		return out, fmt.Errorf("date is required")
	}

	limit := input.Pagination.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := input.Pagination.Offset
	if offset < 0 {
		offset = 0
	}

	results, total, err := u.repo.FindByDate(ctx, input.Date, input.Filter, input.Sort, Pagination{
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		return out, err
	}

	out.Results = results
	out.Total = total
	out.HasMore = offset+len(results) < total
	return out, nil
}

// QueryHistory 取得單檔股票的歷史分析結果。
func (u *QueryUseCase) QueryHistory(ctx context.Context, input QueryHistoryInput) ([]domain.DailyAnalysisResult, error) {
	if input.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 120
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return u.repo.FindHistory(ctx, input.Symbol, input.From, input.To, limit, input.OnlySuccess)
}

// QueryDetail 取得單一股票在指定日期的分析明細。
func (u *QueryUseCase) QueryDetail(ctx context.Context, input QueryDetailInput) (domain.DailyAnalysisResult, error) {
	if input.Symbol == "" {
		return domain.DailyAnalysisResult{}, fmt.Errorf("symbol is required")
	}
	if input.Date.IsZero() {
		return domain.DailyAnalysisResult{}, fmt.Errorf("date is required")
	}
	return u.repo.Get(ctx, input.Symbol, input.Date)
}

// ExportDailyStrong 匯出「當日強勢股清單」為 CSV 字串。
// 具體儲存/傳輸方式交由上層決定。
func (u *QueryUseCase) ExportDailyStrong(ctx context.Context, input ExportDailyStrongInput) (string, error) {
	if input.Date.IsZero() {
		return "", fmt.Errorf("date is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	out, err := u.QueryByDate(ctx, QueryByDateInput{
		Date:   input.Date,
		Filter: input.Filter,
		Sort:   input.Sort,
		Pagination: Pagination{
			Offset: 0,
			Limit:  limit,
		},
	})
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	cw := csv.NewWriter(&sb)
	header := []string{
		"date", "symbol", "market", "industry",
		"close", "change_rate", "return5", "return20",
		"volume", "volume_multiple", "score", "tags",
	}
	if err := cw.Write(header); err != nil {
		return "", err
	}

	for _, r := range out.Results {
		record := []string{
			r.TradeDate.Format("2006-01-02"),
			r.Symbol,
			string(r.Market),
			r.Industry,
			formatFloat(r.Close),
			formatFloat(r.ChangeRate),
			formatPtr(r.Return5),
			formatPtr(r.Return20),
			strconv.FormatInt(r.Volume, 10),
			formatPtr(r.VolumeMultiple),
			formatFloat(r.Score),
			strings.Join(tagStrings(r.Tags), "|"),
		}
		if err := cw.Write(record); err != nil {
			return "", err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 4, 64)
}

func formatPtr(v *float64) string {
	if v == nil {
		return ""
	}
	return formatFloat(*v)
}

func tagStrings(tags []domain.Tag) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = string(t)
	}
	slices.Sort(out)
	return out
}
