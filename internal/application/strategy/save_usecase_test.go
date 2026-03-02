package strategy

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupSaveMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %s", err)
	}
	return gormDB, mock, db
}

func TestSaveScoringStrategyUseCase_Validation(t *testing.T) {
	gormDB, _, db := setupSaveMock(t)
	defer db.Close()
	usecase := NewSaveScoringStrategyUseCase(gormDB)

	tests := []struct {
		name    string
		input   SaveScoringStrategyInput
		wantErr string
	}{
		{
			name: "No rules",
			input: SaveScoringStrategyInput{
				Rules: []SaveRuleInput{},
			},
			wantErr: "至少一個進場規則",
		},
		{
			name: "No exit rule",
			input: SaveScoringStrategyInput{
				Rules: []SaveRuleInput{
					{RuleType: "entry"},
				},
			},
			wantErr: "至少一個出場規則",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := usecase.Execute(context.Background(), tt.input)
			if err == nil || !contains(err.Error(), tt.wantErr) {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveScoringStrategyUseCase_Execute(t *testing.T) {
	gormDB, mock, db := setupSaveMock(t)
	defer db.Close()

	usecase := NewSaveScoringStrategyUseCase(gormDB)

	input := SaveScoringStrategyInput{
		UserID:        "user-123",
		Name:          "Test Strategy",
		Slug:          "test-slug",
		Threshold:     60.0,
		ExitThreshold: 40.0,
		BaseSymbol:    "BTCUSDT",
		Timeframe:     "1d",
		Rules: []SaveRuleInput{
			{
				ConditionName: "Entry Rule",
				Type:          "BASE_SCORE",
				Params:        map[string]interface{}{},
				Weight:        50.0,
				RuleType:      "entry",
			},
			{
				ConditionName: "Exit Rule",
				Type:          "PRICE_RETURN",
				Params:        map[string]interface{}{"min": -0.01},
				Weight:        30.0,
				RuleType:      "exit",
			},
		},
	}

	mock.ExpectBegin()
	// GORM SQL for Create with OnConflict
	mock.ExpectQuery("INSERT INTO \"strategies\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("strat-id-123"))
	mock.ExpectExec("DELETE FROM \"strategy_rules\"").WillReturnResult(sqlmock.NewResult(0, 1))

	// Rule 1
	mock.ExpectQuery("SELECT id FROM \"conditions\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cond-1"))
	mock.ExpectExec("INSERT INTO \"strategy_rules\"").WillReturnResult(sqlmock.NewResult(1, 1))

	// Rule 2
	mock.ExpectQuery("SELECT id FROM \"conditions\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cond-2"))
	mock.ExpectExec("INSERT INTO \"strategy_rules\"").WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	err := usecase.Execute(context.Background(), input)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}
