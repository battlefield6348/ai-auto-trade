package httpapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	"ai-auto-trade/internal/infra/memory"
)

const (
	errCodeBadRequest         = "BAD_REQUEST"
	errCodeInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	errCodeUnauthorized       = "AUTH_UNAUTHORIZED"
	errCodeForbidden          = "AUTH_FORBIDDEN"
	errCodeAnalysisNotReady   = "ANALYSIS_NOT_READY"
	errCodeMethodNotAllowed   = "METHOD_NOT_ALLOWED"
	errCodeNotFound           = "NOT_FOUND"
	errCodeInternal           = "INTERNAL_ERROR"
	refreshCookieName         = "refresh_token"
)

var errNoPrices = errors.New("ingestion data not ready")

// DataRepository 定義 ingestion/analysis 讀寫與查詢接口。
type DataRepository interface {
	analysis.AnalysisQueryRepository
	UpsertTradingPair(ctx context.Context, pair, name string, market dataDomain.Market, industry string) (string, error)
	InsertDailyPrice(ctx context.Context, stockID string, price dataDomain.DailyPrice) error
	PricesByDate(ctx context.Context, date time.Time) ([]dataDomain.DailyPrice, error)
	PricesByPair(ctx context.Context, pair string, timeframe string) ([]dataDomain.DailyPrice, error)
	FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error)
	Get(ctx context.Context, symbol string, date time.Time, timeframe string) (analysisDomain.DailyAnalysisResult, error)
	InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error
	HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error)
	LatestAnalysisDate(ctx context.Context) (time.Time, error)
}

// memoryRepoAdapter 讓 memory.Store 相容 DataRepository。
type memoryRepoAdapter struct {
	store *memory.Store
}

func (m memoryRepoAdapter) UpsertTradingPair(ctx context.Context, pair, name string, market dataDomain.Market, industry string) (string, error) {
	return m.store.UpsertTradingPair(pair, name, market, industry), nil
}

func (m memoryRepoAdapter) InsertDailyPrice(ctx context.Context, stockID string, price dataDomain.DailyPrice) error {
	m.store.InsertDailyPrice(price)
	return nil
}

func (m memoryRepoAdapter) PricesByDate(ctx context.Context, date time.Time) ([]dataDomain.DailyPrice, error) {
	return m.store.PricesByDate(date), nil
}

func (m memoryRepoAdapter) PricesByPair(ctx context.Context, pair string, timeframe string) ([]dataDomain.DailyPrice, error) {
	return m.store.PricesByPair(pair), nil
}

func (m memoryRepoAdapter) InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error {
	m.store.InsertAnalysisResult(res)
	return nil
}

func (m memoryRepoAdapter) HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error) {
	return m.store.HasAnalysisForDate(date), nil
}

func (m memoryRepoAdapter) LatestAnalysisDate(ctx context.Context) (time.Time, error) {
	d, ok := m.store.LatestAnalysisDate()
	if !ok {
		return time.Time{}, fmt.Errorf("no analysis data")
	}
	return d, nil
}

func (m memoryRepoAdapter) FindByDate(ctx context.Context, date time.Time, filter analysis.QueryFilter, sortOpt analysis.SortOption, pagination analysis.Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	return m.store.FindByDate(ctx, date, filter, sortOpt, pagination)
}

func (m memoryRepoAdapter) FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error) {
	return m.store.FindHistory(ctx, symbol, from, to, limit, onlySuccess)
}

func (m memoryRepoAdapter) Get(ctx context.Context, symbol string, date time.Time, timeframe string) (analysisDomain.DailyAnalysisResult, error) {
	return m.store.Get(ctx, symbol, date)
}
