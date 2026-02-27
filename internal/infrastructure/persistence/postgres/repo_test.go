package postgres

import (
	"context"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRepo_InsertAnalysisResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	ctx := context.Background()

	res := analysisDomain.DailyAnalysisResult{
		Symbol:    "BTCUSDT",
		Timeframe: "1d",
		TradeDate: time.Now(),
		Version:   "test-v1",
		Close:     50000.0,
		Success:   true,
		Score:     75.5,
	}

	mock.ExpectExec("INSERT INTO analysis_results").
		WithArgs(
			"stock-123",
			res.Timeframe,
			res.TradeDate,
			res.Version,
			res.Close,
			res.Change,
			res.ChangeRate,
			sqlmock.AnyArg(), // return_5d
			sqlmock.AnyArg(), // return_20d
			sqlmock.AnyArg(), // return_60d
			res.Volume,
			sqlmock.AnyArg(), // volume_ratio
			res.Score,
			sqlmock.AnyArg(), // ma_20
			sqlmock.AnyArg(), // price_position_20d
			sqlmock.AnyArg(), // high_20d
			sqlmock.AnyArg(), // low_20d
			"success",
			sqlmock.AnyArg(), // error_reason
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.InsertAnalysisResult(ctx, "stock-123", res)
	if err != nil {
		t.Errorf("InsertAnalysisResult failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}

func TestRepo_PricesByPair(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"trading_pair", "market_type", "timeframe", "trade_date", "open_price", "high_price", "low_price", "close_price", "volume"}).
		AddRow("BTCUSDT", "crypto", "1d", time.Now(), 49000.0, 51000.0, 48500.0, 50000.0, 1000)

	mock.ExpectQuery("SELECT (.+) FROM daily_prices").
		WithArgs("BTCUSDT", "1d").
		WillReturnRows(rows)

	prices, err := repo.PricesByPair(ctx, "BTCUSDT", "1d")
	if err != nil {
		t.Errorf("PricesByPair failed: %v", err)
	}

	if len(prices) != 1 {
		t.Errorf("expected 1 price, got %d", len(prices))
	}

	if prices[0].Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", prices[0].Symbol)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}

func TestRepo_UpsertTradingPair(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	ctx := context.Background()

	mock.ExpectQuery("INSERT INTO stocks").
		WithArgs("BTCUSDT", "CRYPTO", "Bitcoin", "Crypto").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("uuid-123"))

	id, err := repo.UpsertTradingPair(ctx, "BTCUSDT", "Bitcoin", dataDomain.MarketCrypto, "Crypto")
	if err != nil {
		t.Errorf("UpsertTradingPair failed: %v", err)
	}
	if id != "uuid-123" {
		t.Errorf("expected uuid-123, got %s", id)
	}
}

func TestRepo_InsertDailyPrice(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	ctx := context.Background()

	price := dataDomain.DailyPrice{
		Timeframe: "1d",
		TradeDate: time.Now(),
		Open:      100,
		High:      110,
		Low:       90,
		Close:     105,
		Volume:    1000,
	}

	mock.ExpectExec("INSERT INTO daily_prices").
		WithArgs("stock-1", price.Timeframe, price.TradeDate, price.Open, price.High, price.Low, price.Close, price.Volume, int64(0), 0.0, 0.0, false).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.InsertDailyPrice(ctx, "stock-1", price)
	if err != nil {
		t.Errorf("InsertDailyPrice failed: %v", err)
	}
}

func TestRepo_PricesByDate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"trading_pair", "market_type", "timeframe", "trade_date", "open_price", "high_price", "low_price", "close_price", "volume"}).
		AddRow("BTCUSDT", "CRYPTO", "1d", now, 100, 110, 90, 105, 1000)

	mock.ExpectQuery("SELECT (.+) FROM daily_prices (.+) WHERE dp.trade_date = \\$1").
		WithArgs(now).
		WillReturnRows(rows)

	prices, err := repo.PricesByDate(ctx, now)
	if err != nil {
		t.Errorf("PricesByDate failed: %v", err)
	}
	if len(prices) != 1 {
		t.Errorf("expected 1 price, got %d", len(prices))
	}
}

func TestRepo_HasAnalysisForDate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(now).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	ok, err := repo.HasAnalysisForDate(ctx, now)
	if err != nil {
		t.Errorf("HasAnalysisForDate failed: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
}
func TestRepo_FindByDate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	ctx := context.Background()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"trading_pair", "market_type", "industry", "timeframe", "trade_date", "analysis_version", "close_price", "change", "change_percent", "return_5d", "return_20d", "return_60d", "volume", "volume_ratio", "score", "ma_20", "price_position_20d", "high_20d", "low_20d", "status", "error_reason"}).
		AddRow("BTCUSDT", "crypto", "Finance", "1d", now, "v1", 50000.0, 1000.0, 0.02, 0.05, 0.1, 0.2, 1000, 1.5, 80.0, 48000.0, 0.8, 51000.0, 45000.0, "success", nil)

	mock.ExpectQuery("SELECT (.+) FROM analysis_results (.+) WHERE ar.trade_date = \\$1").
		WithArgs(now, true, 100, 0).
		WillReturnRows(rows)

	mock.ExpectQuery("SELECT count(.+) FROM analysis_results").
		WithArgs(now, true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	results, total, err := repo.FindByDate(ctx, now, analysis.QueryFilter{OnlySuccess: true}, analysis.SortOption{}, analysis.Pagination{Limit: 100, Offset: 0})
	if err != nil {
		t.Fatalf("FindByDate failed: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Errorf("expected 1 result, got %d (total %d)", len(results), total)
	}
}

func TestRepo_FindHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewRepo(db)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"trading_pair", "market_type", "timeframe", "trade_date", "analysis_version", "close_price", "change", "change_percent", "return_5d", "return_20d", "return_60d", "volume", "volume_ratio", "score", "ma_20", "price_position_20d", "high_20d", "low_20d", "status", "error_reason"}).
		AddRow("BTCUSDT", "crypto", "1d", time.Now(), "v1", 50000.0, 1000.0, 0.02, 0.05, 0.1, 0.2, 1000, 1.5, 80.0, 48000.0, 0.8, 51000.0, 45000.0, "success", nil)

	mock.ExpectQuery("SELECT (.+) FROM analysis_results (.+) WHERE s.trading_pair = \\$1").
		WithArgs("BTCUSDT", "1d", 10).
		WillReturnRows(rows)

	results, err := repo.FindHistory(ctx, "BTCUSDT", "1d", nil, nil, 10, false)
	if err != nil {
		t.Fatalf("FindHistory failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}
