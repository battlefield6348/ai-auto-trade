package analysis

import (
	"context"
	"errors"
	"testing"
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

type fakeBasicProvider struct {
	list []BasicInfo
	err  error
}

func (f fakeBasicProvider) ListBasicInfo(_ context.Context, _ []string, _ time.Time) ([]BasicInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

type fakeHistoryProvider struct {
	history map[string][]dataingestion.DailyPrice
	err     error
}

func (f fakeHistoryProvider) GetHistory(_ context.Context, symbol string, _ time.Time, _ int) ([]dataingestion.DailyPrice, error) {
	if f.err != nil {
		return nil, f.err
	}
	h, ok := f.history[symbol]
	if !ok {
		return nil, errors.New("history not found")
	}
	return h, nil
}

type fakeAnalysisRepo struct {
	results []analysisDomain.DailyAnalysisResult
	err     error
}

func (r *fakeAnalysisRepo) SaveDailyResult(_ context.Context, result analysisDomain.DailyAnalysisResult) error {
	if r.err != nil {
		return r.err
	}
	r.results = append(r.results, result)
	return nil
}

func TestAnalyzeUseCase_Success(t *testing.T) {
	tradeDate := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	closes := make([]float64, 25)
	volumes := make([]int64, 25)
	for i := 0; i < 25; i++ {
		closes[i] = 10 + float64(i)
		volumes[i] = 100 + int64(i*5)
	}
	volumes[24] = 400 // 放大量以觸發量能標籤
	history := buildHistory(tradeDate, closes, volumes)

	basicProvider := fakeBasicProvider{list: []BasicInfo{{Symbol: "2330", Market: dataingestion.MarketTWSE, Industry: "半導體"}}}
	historyProvider := fakeHistoryProvider{history: map[string][]dataingestion.DailyPrice{"2330": history}}
	repo := &fakeAnalysisRepo{}

	usecase := NewAnalyzeUseCase(basicProvider, historyProvider, repo)
	res, err := usecase.Execute(context.Background(), AnalyzeInput{
		TradeDate: tradeDate,
		Version:   "v1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.SuccessCount != 1 || res.FailedCount != 0 {
		t.Fatalf("unexpected result counts: %+v", res)
	}

	if len(repo.results) != 1 {
		t.Fatalf("expected repo to store one result")
	}

	r := repo.results[0]
	if !r.Success {
		t.Fatalf("analysis marked as failed: %v", r.ErrorReason)
	}

	if r.Return5 == nil || *r.Return5 <= 0 {
		t.Fatalf("expected positive return5, got %+v", r.Return5)
	}

	if r.VolumeMultiple == nil || *r.VolumeMultiple <= 1.0 {
		t.Fatalf("expected volume multiple > 1, got %+v", r.VolumeMultiple)
	}

	if !hasTag(r.Tags, analysisDomain.TagShortTermStrong) {
		t.Fatalf("expected short term strong tag, got %v", r.Tags)
	}

	if r.Score <= 50 {
		t.Fatalf("expected score > 50, got %f", r.Score)
	}
}

func TestAnalyzeUseCase_FailureOnHistory(t *testing.T) {
	tradeDate := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	basicProvider := fakeBasicProvider{list: []BasicInfo{{Symbol: "2330", Market: dataingestion.MarketTWSE}}}
	historyProvider := fakeHistoryProvider{err: errors.New("source down")}
	repo := &fakeAnalysisRepo{}

	usecase := NewAnalyzeUseCase(basicProvider, historyProvider, repo)
	res, err := usecase.Execute(context.Background(), AnalyzeInput{
		TradeDate: tradeDate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.SuccessCount != 0 || res.FailedCount != 1 {
		t.Fatalf("unexpected result counts: %+v", res)
	}
}

func TestAnalyzeUseCase_RepoError(t *testing.T) {
	tradeDate := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	closes := make([]float64, 25)
	volumes := make([]int64, 25)
	for i := 0; i < 25; i++ {
		closes[i] = 20 + float64(i)
		volumes[i] = 120 + int64(i*3)
	}
	history := buildHistory(tradeDate, closes, volumes)

	basicProvider := fakeBasicProvider{list: []BasicInfo{{Symbol: "2330", Market: dataingestion.MarketTWSE}}}
	historyProvider := fakeHistoryProvider{history: map[string][]dataingestion.DailyPrice{"2330": history}}
	repo := &fakeAnalysisRepo{err: errors.New("db unavailable")}

	usecase := NewAnalyzeUseCase(basicProvider, historyProvider, repo)
	res, err := usecase.Execute(context.Background(), AnalyzeInput{
		TradeDate: tradeDate,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.SuccessCount != 0 || res.FailedCount != 1 {
		t.Fatalf("unexpected result counts: %+v", res)
	}
}

func buildHistory(tradeDate time.Time, closes []float64, volumes []int64) []dataingestion.DailyPrice {
	if len(closes) != len(volumes) {
		panic("closes and volumes length mismatch")
	}
	start := tradeDate.AddDate(0, 0, -len(closes)+1)
	history := make([]dataingestion.DailyPrice, 0, len(closes))
	for i := 0; i < len(closes); i++ {
		day := start.AddDate(0, 0, i)
		close := closes[i]
		history = append(history, dataingestion.DailyPrice{
			Symbol:    "2330",
			Market:    dataingestion.MarketTWSE,
			TradeDate: day,
			Open:      close - 0.5,
			High:      close + 1,
			Low:       close - 1,
			Close:     close,
			Volume:    volumes[i],
		})
	}
	return history
}

func hasTag(tags []analysisDomain.Tag, target analysisDomain.Tag) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}
