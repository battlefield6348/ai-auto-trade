package strategy

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCondition_ParseParams(t *testing.T) {
	c := Condition{
		ParamsRaw: json.RawMessage(`{"min": 1.5, "days": 5}`),
	}

	p, err := c.ParseParams()
	if err != nil {
		t.Fatalf("ParseParams failed: %v", err)
	}

	if p["min"] != 1.5 || p["days"] != float64(5) {
		t.Errorf("Unexpected params: %v", p)
	}

	// Test empty params
	cEmpty := Condition{ParamsRaw: nil}
	pEmpty, err := cEmpty.ParseParams()
	if err != nil || pEmpty != nil {
		t.Errorf("Empty params failed: err=%v, p=%v", err, pEmpty)
	}
}

func TestCondition_ParseParamsInto(t *testing.T) {
	type testParams struct {
		Min  float64 `json:"min"`
		Days int     `json:"days"`
	}

	c := Condition{
		ParamsRaw: json.RawMessage(`{"min": 1.5, "days": 5}`),
	}

	var tp testParams
	err := c.ParseParamsInto(&tp)
	if err != nil {
		t.Fatalf("ParseParamsInto failed: %v", err)
	}

	if tp.Min != 1.5 || tp.Days != 5 {
		t.Errorf("Unexpected params into: %+v", tp)
	}
}

func TestLoadScoringStrategy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	ctx := context.Background()
	slug := "test-strat"
	now := time.Now()

	// 1. Mock Strategy Row
	stratRows := sqlmock.NewRows([]string{"id", "user_id", "name", "slug", "description", "timeframe", "base_symbol", "threshold", "exit_threshold", "is_active", "env", "risk_settings", "created_at", "updated_at"}).
		AddRow("s-123", "u-1", "Test Strategy", slug, "desc", "1d", "BTCUSDT", 70.0, 40.0, true, "both", []byte(`{"take_profit_pct": 5.0}`), now, now)

	mock.ExpectQuery("SELECT (.+) FROM strategies WHERE slug = \\$1").
		WithArgs(slug).
		WillReturnRows(stratRows)

	// 2. Mock Rules Rows
	ruleRows := sqlmock.NewRows([]string{"strategy_id", "weight", "rule_type", "id", "name", "type", "params"}).
		AddRow("s-123", 100.0, "entry", "c-1", "Cond 1", "BASE_SCORE", []byte(`{}`)).
		AddRow("s-123", 50.0, "exit", "c-2", "Cond 2", "PRICE_RETURN", []byte(`{"min": -0.01}`))

	mock.ExpectQuery("SELECT (.+) FROM strategy_rules (.+) JOIN conditions").
		WithArgs("s-123").
		WillReturnRows(ruleRows)

	s, err := LoadScoringStrategyBySlug(ctx, db, slug)
	if err != nil {
		t.Fatalf("LoadScoringStrategyBySlug failed: %v", err)
	}

	if s.Name != "Test Strategy" {
		t.Errorf("Expected Test Strategy, got %s", s.Name)
	}
	if len(s.EntryRules) != 1 {
		t.Errorf("Expected 1 entry rule, got %d", len(s.EntryRules))
	}
	if len(s.ExitRules) != 1 {
		t.Errorf("Expected 1 exit rule, got %d", len(s.ExitRules))
	}
	if s.Risk.TakeProfitPct == nil || *s.Risk.TakeProfitPct != 5.0 {
		t.Error("Risk settings not correctly unmarshaled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}

func TestLoadScoringStrategy_NotFound(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM strategies").
		WillReturnError(sql.ErrNoRows)

	s, err := LoadScoringStrategyBySlug(context.Background(), db, "none")
	if err == nil || s != nil {
		t.Error("Expected error for non-existent strategy")
	}
}
