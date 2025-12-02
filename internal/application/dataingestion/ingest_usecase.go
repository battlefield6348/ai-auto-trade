package dataingestion

import (
	"context"
	"fmt"
	"time"

	"ai-auto-trade/internal/domain/dataingestion"
)

// PriceSource 抽象化外部資料來源（交易所、檔案等）。
type PriceSource interface {
	FetchDaily(ctx context.Context, date time.Time, symbols []string, market *dataingestion.Market) ([]dataingestion.DailyPrice, error)
}

// PriceRepository 定義儲存介面。
type PriceRepository interface {
	UpsertDailyPrice(ctx context.Context, price dataingestion.DailyPrice, replace bool) error
}

// IngestUseCase 提供每日例行/回補/重抓的共用流程。
type IngestUseCase struct {
	source PriceSource
	repo   PriceRepository
}

func NewIngestUseCase(source PriceSource, repo PriceRepository) *IngestUseCase {
	return &IngestUseCase{
		source: source,
		repo:   repo,
	}
}

type IngestMode string

const (
	IngestModeDaily    IngestMode = "daily"
	IngestModeBackfill IngestMode = "backfill"
	IngestModeRecovery IngestMode = "recovery"
)

// IngestInput 控制一次資料抓取行為。
type IngestInput struct {
	Date         time.Time
	Mode         IngestMode
	Replace      bool
	Symbols      []string
	MarketFilter *dataingestion.Market
}

type Failure struct {
	Symbol string
	Reason string
}

type IngestResult struct {
	SuccessCount int
	FailedCount  int
	Failures     []Failure
}

// Execute 執行一次資料抓取與寫入。
func (u *IngestUseCase) Execute(ctx context.Context, input IngestInput) (IngestResult, error) {
	result := IngestResult{}

	if input.Date.IsZero() {
		return result, fmt.Errorf("date is required")
	}

	if input.Mode == "" {
		input.Mode = IngestModeDaily
	}

	rawPrices, err := u.source.FetchDaily(ctx, input.Date, input.Symbols, input.MarketFilter)
	if err != nil {
		return result, fmt.Errorf("fetch daily prices: %w", err)
	}

	for _, p := range rawPrices {
		if err := p.Validate(); err != nil {
			result.FailedCount++
			result.Failures = append(result.Failures, Failure{
				Symbol: p.Symbol,
				Reason: err.Error(),
			})
			continue
		}

		if err := u.repo.UpsertDailyPrice(ctx, p, input.Replace); err != nil {
			result.FailedCount++
			result.Failures = append(result.Failures, Failure{
				Symbol: p.Symbol,
				Reason: fmt.Sprintf("store failed: %v", err),
			})
			continue
		}

		result.SuccessCount++
	}

	return result, nil
}
