package analysis

import (
	"testing"
	"time"

	"ai-auto-trade/internal/domain/dataingestion"
)

func TestDailyAnalysisResult_Validate(t *testing.T) {
	tests := []struct {
		name    string
		res     DailyAnalysisResult
		wantErr bool
	}{
		{
			name: "Valid TWSE",
			res: DailyAnalysisResult{
				Symbol:    "2330",
				TradeDate: time.Now(),
				Market:    dataingestion.MarketTWSE,
			},
			wantErr: false,
		},
		{
			name: "Missing Symbol",
			res: DailyAnalysisResult{
				TradeDate: time.Now(),
				Market:    dataingestion.MarketTWSE,
			},
			wantErr: true,
		},
		{
			name: "Missing Date",
			res: DailyAnalysisResult{
				Symbol: "2330",
				Market: dataingestion.MarketTWSE,
			},
			wantErr: true,
		},
		{
			name: "Invalid Market",
			res: DailyAnalysisResult{
				Symbol:    "2330",
				TradeDate: time.Now(),
				Market:    "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.res.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
