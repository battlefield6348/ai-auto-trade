package dataingestion

import (
	"errors"
	"fmt"
	"time"
)

// Market enumerates supported markets.
type Market string

const (
	MarketTWSE Market = "TWSE"
	MarketTPEx Market = "TPEx"
)

// DailyPrice represents a single stock's daily OHLCV record.
type DailyPrice struct {
	Symbol      string
	Market      Market
	TradeDate   time.Time
	Open        float64
	High        float64
	Low         float64
	Close       float64
	Volume      int64 // 成交量（股）
	Turnover    int64 // 成交金額（可為 0，視來源而定）
	Change      float64
	ChangeRate  float64
	IsExDividend bool
}

// ValidationError groups multiple validation failures.
type ValidationError struct {
	Reasons []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("daily price validation failed: %v", e.Reasons)
}

// Validate ensures the record satisfies basic integrity constraints.
func (p DailyPrice) Validate() error {
	var reasons []string

	if p.Symbol == "" {
		reasons = append(reasons, "symbol is required")
	}

	if p.TradeDate.IsZero() {
		reasons = append(reasons, "trade_date is required")
	}

	switch p.Market {
	case MarketTWSE, MarketTPEx:
		// ok
	default:
		reasons = append(reasons, "unsupported market")
	}

	if p.Open < 0 || p.High < 0 || p.Low < 0 || p.Close < 0 {
		reasons = append(reasons, "price fields must be >= 0")
	}

	if p.High < maxFloat64(p.Open, p.Close, p.Low) {
		reasons = append(reasons, "high must be >= open/close/low")
	}

	if p.Low > minFloat64(p.Open, p.Close, p.High) {
		reasons = append(reasons, "low must be <= open/close/high")
	}

	if p.Volume < 0 {
		reasons = append(reasons, "volume must be >= 0")
	}

	if p.Turnover < 0 {
		reasons = append(reasons, "turnover must be >= 0")
	}

	if len(reasons) > 0 {
		return &ValidationError{Reasons: reasons}
	}

	return nil
}

func maxFloat64(values ...float64) float64 {
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func minFloat64(values ...float64) float64 {
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// IsValidationError helper to check error type.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
