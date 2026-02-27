package trading

import (
	"ai-auto-trade/internal/application/analysis"
	"testing"
)

func TestStrategy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		strat   Strategy
		wantErr bool
	}{
		{
			name: "Valid Strategy",
			strat: Strategy{
				Name:       "Test",
				BaseSymbol: "BTCUSDT",
				Env:        EnvTest,
				Status:     StatusActive,
				Buy:        ConditionSet{Conditions: []analysis.Condition{{}}},
				Sell:       ConditionSet{Conditions: []analysis.Condition{{}}},
			},
			wantErr: false,
		},
		{
			name: "Missing Name",
			strat: Strategy{
				BaseSymbol: "BTCUSDT",
			},
			wantErr: true,
		},
		{
			name: "Missing Symbol",
			strat: Strategy{
				Name: "Test",
			},
			wantErr: true,
		},
		{
			name: "Unsupported Env",
			strat: Strategy{
				Name:       "Test",
				BaseSymbol: "BTCUSDT",
				Env:        "invalid",
			},
			wantErr: true,
		},
		{
			name: "Missing Conditions",
			strat: Strategy{
				Name:       "Test",
				BaseSymbol: "BTCUSDT",
				Env:        EnvTest,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.strat.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
