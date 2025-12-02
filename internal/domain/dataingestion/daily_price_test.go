package dataingestion

import (
	"testing"
	"time"
)

func TestDailyPriceValidateSuccess(t *testing.T) {
	p := DailyPrice{
		Symbol:    "2330",
		Market:    MarketTWSE,
		TradeDate: time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC),
		Open:      600,
		High:      605,
		Low:       590,
		Close:     602,
		Volume:    1000000,
		Turnover:  1000000000,
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("expected valid price, got error: %v", err)
	}
}

func TestDailyPriceValidateErrors(t *testing.T) {
	p := DailyPrice{
		Symbol:    "",
		Market:    "UNKNOWN",
		TradeDate: time.Time{},
		Open:      -1,
		High:      1,
		Low:       2,
		Close:     -2,
		Volume:    -1,
		Turnover:  -1,
	}

	err := p.Validate()
	if err == nil {
		t.Fatalf("expected validation errors")
	}

	if !IsValidationError(err) {
		t.Fatalf("expected validation error type, got %T", err)
	}
}
