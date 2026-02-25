package analysis

import (
	"testing"
	"time"

	"ai-auto-trade/internal/domain/dataingestion"
)

func TestAnalyzeOne_ScoreCalculation(t *testing.T) {
	tradeDate := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)
	info := BasicInfo{Symbol: "BTCUSDT"}

	// Create 21 days of history to satisfy MA20/AvgVolume20
	// Day 0 to 20 (21 entries)
	history := make([]dataingestion.DailyPrice, 21)
	for i := 0; i < 21; i++ {
		history[i] = dataingestion.DailyPrice{
			TradeDate: tradeDate.AddDate(0, 0, i-20),
			Close:     100,
			Volume:    1000,
			High:      110,
			Low:       90,
		}
	}

	// Case 1: Neutral
	res, _ := analyzeOne(info, tradeDate, history, "v1")
	if res.Score != 50.0 {
		t.Errorf("Expected neutral score 50.0, got %f", res.Score)
	}

	// Case 2: Positive - Price up 10% in 5 days, Volume surge
	// Modify history: Day 15 (5 days ago) price was lower
	history[15].Close = 90.9 // 100/90.9 - 1 approx 10%
	// Modify latest (Day 20) volume surge
	history[20].Volume = 2000 // Multiple = 2.0 (relative to avg 1000)
	
	res2, _ := analyzeOne(info, tradeDate, history, "v1")
	if res2.Score <= 50.0 {
		t.Errorf("Expected positive score > 50, got %f", res2.Score)
	}
	t.Logf("Positive Case Score: %f", res2.Score)
}

func TestMovingAverage(t *testing.T) {
	history := []dataingestion.DailyPrice{
		{Close: 10}, {Close: 20}, {Close: 30},
	}
	
	// MA2
	avg2 := movingAverage(history, 2)
	if *avg2 != 25.0 {
		t.Errorf("Expected MA2=25, got %v", *avg2)
	}
	
	// Not enough data
	avg5 := movingAverage(history, 5)
	if avg5 != nil {
		t.Error("Expected nil for MA5 with only 3 points")
	}
}

func TestHighLowRange(t *testing.T) {
	history := []dataingestion.DailyPrice{
		{High: 100, Low: 80},
		{High: 120, Low: 90}, 
		{High: 110, Low: 85},
	}
	
	h, l, pos := highLowRange(history, 3, 110)
	if *h != 120.0 || *l != 80.0 {
		t.Errorf("Wrong H/L: expected 120/80, got %f/%f", *h, *l)
	}
	// (110 - 80) / (120 - 80) = 30/40 = 0.75
	if *pos != 0.75 {
		t.Errorf("Expected post 0.75, got %f", *pos)
	}
}
