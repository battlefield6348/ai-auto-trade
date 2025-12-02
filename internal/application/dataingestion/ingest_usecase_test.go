package dataingestion

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "ai-auto-trade/internal/domain/dataingestion"
)

type fakeSource struct {
	prices []domain.DailyPrice
	err    error
}

func (f fakeSource) FetchDaily(_ context.Context, _ time.Time, _ []string, _ *domain.Market) ([]domain.DailyPrice, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.prices, nil
}

type fakeRepo struct {
	stored []domain.DailyPrice
	err    error
}

func (r *fakeRepo) UpsertDailyPrice(_ context.Context, price domain.DailyPrice, _ bool) error {
	if r.err != nil {
		return r.err
	}
	r.stored = append(r.stored, price)
	return nil
}

func TestIngestUseCase_SuccessAndValidation(t *testing.T) {
	valid := domain.DailyPrice{
		Symbol:    "2330",
		Market:    domain.MarketTWSE,
		TradeDate: time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC),
		Open:      10,
		High:      12,
		Low:       9,
		Close:     11,
		Volume:    1000,
	}

	invalid := domain.DailyPrice{
		Symbol:    "",
		Market:    domain.MarketTWSE,
		TradeDate: time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC),
		Open:      -1,
		High:      -1,
		Low:       -1,
		Close:     -1,
		Volume:    1000,
	}

	source := fakeSource{prices: []domain.DailyPrice{valid, invalid}}
	repo := &fakeRepo{}

	usecase := NewIngestUseCase(source, repo)
	res, err := usecase.Execute(context.Background(), IngestInput{
		Date:    valid.TradeDate,
		Mode:    IngestModeDaily,
		Replace: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.SuccessCount != 1 || res.FailedCount != 1 {
		t.Fatalf("unexpected result counts: %+v", res)
	}

	if len(repo.stored) != 1 || repo.stored[0].Symbol != "2330" {
		t.Fatalf("expected repository stored valid record, got: %+v", repo.stored)
	}
}

func TestIngestUseCase_FetchError(t *testing.T) {
	source := fakeSource{err: errors.New("fetch fail")}
	repo := &fakeRepo{}
	usecase := NewIngestUseCase(source, repo)

	_, err := usecase.Execute(context.Background(), IngestInput{
		Date: time.Now(),
	})

	if err == nil {
		t.Fatalf("expected error from fetch")
	}
}

func TestIngestUseCase_StoreError(t *testing.T) {
	valid := domain.DailyPrice{
		Symbol:    "1101",
		Market:    domain.MarketTWSE,
		TradeDate: time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC),
		Open:      10,
		High:      12,
		Low:       9,
		Close:     11,
		Volume:    1000,
	}

	source := fakeSource{prices: []domain.DailyPrice{valid}}
	repo := &fakeRepo{err: errors.New("db fail")}
	usecase := NewIngestUseCase(source, repo)

	res, err := usecase.Execute(context.Background(), IngestInput{
		Date: valid.TradeDate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.SuccessCount != 0 || res.FailedCount != 1 {
		t.Fatalf("unexpected result counts: %+v", res)
	}
}
