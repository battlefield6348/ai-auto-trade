package strategy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSaveScoringStrategyUseCase_Execute(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	usecase := NewSaveScoringStrategyUseCase(db)

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

	// Expectation 1: Insert or Update strategy
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO strategies").
		WithArgs(
			input.UserID, input.Name, input.Slug, 
			input.Threshold, input.ExitThreshold, 
			input.BaseSymbol, input.Timeframe,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("strat-id-123"))

	// Expectation 2: Delete old rules for THIS strategy ID only
	mock.ExpectExec("DELETE FROM strategy_rules").
		WithArgs("strat-id-123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expectation 3: Process Rule 1 (Entry)
	params1, _ := json.Marshal(input.Rules[0].Params)
	mock.ExpectQuery("SELECT id FROM conditions").
		WithArgs(input.Rules[0].Type, sqlmock.AnyArg()). // Use AnyArg for params::jsonb matching if needed or exact
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO conditions").
		WithArgs(input.Rules[0].ConditionName, input.Rules[0].Type, params1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cond-1"))
	mock.ExpectExec("INSERT INTO strategy_rules").
		WithArgs("strat-id-123", "cond-1", 50.0, "entry").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expectation 4: Process Rule 2 (Exit)
	params2, _ := json.Marshal(input.Rules[1].Params)
	mock.ExpectQuery("SELECT id FROM conditions").
		WithArgs(input.Rules[1].Type, params2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cond-2"))
	mock.ExpectExec("INSERT INTO strategy_rules").
		WithArgs("strat-id-123", "cond-2", 30.0, "exit").
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	err = usecase.Execute(context.Background(), input)
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestSaveScoringStrategyUseCase_Validation(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()
	usecase := NewSaveScoringStrategyUseCase(db)

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

func TestSaveScoringStrategyUseCase_ConditionReuse(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	usecase := NewSaveScoringStrategyUseCase(db)

	input := SaveScoringStrategyInput{
		UserID: "u1", Name: "N1", Slug: "S1",
		Rules: []SaveRuleInput{
			{Type: "T1", RuleType: "entry", Params: map[string]interface{}{"p": 1}},
			{Type: "T2", RuleType: "exit"},
		},
	}

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO strategies").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("sid"))
	mock.ExpectExec("DELETE FROM strategy_rules").WillReturnResult(sqlmock.NewResult(0, 0))

	// T1 already exists
	mock.ExpectQuery("SELECT id FROM conditions").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cid-1"))
	mock.ExpectExec("INSERT INTO strategy_rules").WithArgs("sid", "cid-1", 0.0, "entry").WillReturnResult(sqlmock.NewResult(1, 1))

	// T2 new
	mock.ExpectQuery("SELECT id FROM conditions").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO conditions").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("cid-2"))
	mock.ExpectExec("INSERT INTO strategy_rules").WithArgs("sid", "cid-2", 0.0, "exit").WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectCommit()

	if err := usecase.Execute(context.Background(), input); err != nil {
		t.Fatal(err)
	}
}

func TestSaveScoringStrategyUseCase_NoInterference(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	usecase := NewSaveScoringStrategyUseCase(db)

	// Strategy A
	inputA := SaveScoringStrategyInput{
		UserID: "u", Name: "A", Slug: "slug-a",
		Rules: []SaveRuleInput{{Type: "T1", RuleType: "entry"}, {Type: "T2", RuleType: "exit"}},
	}

	mock.ExpectBegin()
	// Prove that we specifically use the Slug to get the correct Strategy ID
	mock.ExpectQuery("INSERT INTO strategies").WithArgs(sqlmock.AnyArg(), "A", "slug-a", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("uuid-a"))
	
	// Prove that we ONLY delete rules for the returned uuid-a
	mock.ExpectExec("DELETE FROM strategy_rules").WithArgs("uuid-a").WillReturnResult(sqlmock.NewResult(0, 0))
	
	mock.ExpectQuery("SELECT id FROM conditions").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("c1"))
	mock.ExpectExec("INSERT INTO strategy_rules").WithArgs("uuid-a", "c1", sqlmock.AnyArg(), "entry").WillReturnResult(sqlmock.NewResult(1, 1))
	
	mock.ExpectQuery("SELECT id FROM conditions").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("c2"))
	mock.ExpectExec("INSERT INTO strategy_rules").WithArgs("uuid-a", "c2", sqlmock.AnyArg(), "exit").WillReturnResult(sqlmock.NewResult(1, 1))
	
	mock.ExpectCommit()

	if err := usecase.Execute(context.Background(), inputA); err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Interference check failed for Strategy A: %s", err)
	}
}

func TestSaveScoringStrategyUseCase_Rollback(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()
	usecase := NewSaveScoringStrategyUseCase(db)

	input := SaveScoringStrategyInput{
		UserID: "u", Name: "A", Slug: "slug-a",
		Rules: []SaveRuleInput{{Type: "T1", RuleType: "entry"}, {Type: "T2", RuleType: "exit"}},
	}

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO strategies").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("uuid-a"))
	mock.ExpectExec("DELETE FROM strategy_rules").WillReturnResult(sqlmock.NewResult(0, 0))
	
	// Force error on condition lookup/insert
	mock.ExpectQuery("SELECT id FROM conditions").WillReturnError(fmt.Errorf("db error"))
	
	mock.ExpectRollback()

	err := usecase.Execute(context.Background(), input)
	if err == nil {
		t.Error("Expected error but got nil")
	}
	
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Rollback expectations not met: %s", err)
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
