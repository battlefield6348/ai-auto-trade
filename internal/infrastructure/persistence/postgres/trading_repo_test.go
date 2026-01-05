package postgres

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	tradingDomain "ai-auto-trade/internal/domain/trading"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateStrategy_UseCreatedBy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)
	created := time.Now()
	mock.ExpectQuery("INSERT INTO strategies").
		WithArgs(
			"策略A", "",
			"BTCUSDT", "1d", "both", "draft", 1,
			jsonMatcher(t, tradingDomain.ConditionSet{Logic: analysis.LogicAND, Conditions: []analysis.Condition{{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 50}}}}),
			jsonMatcher(t, tradingDomain.ConditionSet{Logic: analysis.LogicAND, Conditions: []analysis.Condition{{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpLTE, Value: 30}}}}),
			jsonMatcher(t, tradingDomain.RiskSettings{}),
			sqlmock.AnyArg(), // user_id (from created_by)
			sqlmock.AnyArg(), // created_by
			sqlmock.AnyArg(), // updated_by
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("s-1", created, created))

	_, err = repo.CreateStrategy(context.Background(), tradingDomain.Strategy{
		Name:       "策略A",
		BaseSymbol: "BTCUSDT",
		Timeframe:  "1d",
		Env:        tradingDomain.EnvBoth,
		Status:     tradingDomain.StatusDraft,
		Version:    1,
		Buy: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 50}},
			},
		},
		Sell: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpLTE, Value: 30}},
			},
		},
		Risk:      tradingDomain.RiskSettings{},
		CreatedBy: "user-1",
		UpdatedBy: "user-1",
	})
	if err != nil {
		t.Fatalf("create strategy: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestCreateStrategy_FallbackUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)
	created := time.Now()

	mock.ExpectQuery("SELECT id FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("admin-id"))

	mock.ExpectQuery("INSERT INTO strategies").
		WithArgs(
			"策略B", "",
			"BTCUSDT", "1d", "both", "draft", 1,
			sqlmock.AnyArg(),         // buy_conditions
			sqlmock.AnyArg(),         // sell_conditions
			sqlmock.AnyArg(),         // risk_settings
			driver.Value("admin-id"), // user_id (fallback)
			driver.Value("admin-id"), // created_by fallback
			driver.Value("admin-id"), // updated_by fallback
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("s-2", created, created))

	_, err = repo.CreateStrategy(context.Background(), tradingDomain.Strategy{
		Name:       "策略B",
		BaseSymbol: "BTCUSDT",
		Timeframe:  "1d",
		Env:        tradingDomain.EnvBoth,
		Status:     tradingDomain.StatusDraft,
		Version:    1,
		Buy: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpGTE, Value: 60}},
			},
		},
		Sell: tradingDomain.ConditionSet{
			Logic: analysis.LogicAND,
			Conditions: []analysis.Condition{
				{Type: analysis.ConditionNumeric, Numeric: &analysis.NumericCondition{Field: analysis.FieldScore, Op: analysis.OpLTE, Value: 40}},
			},
		},
		Risk: tradingDomain.RiskSettings{},
	})
	if err != nil {
		t.Fatalf("create strategy with fallback: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// jsonEqual marshals v and returns a sqlmock argument matcher.
type jsonArg struct{ expected []byte }

func (j jsonArg) Match(v driver.Value) bool {
	b, ok := v.([]byte)
	if !ok {
		return false
	}
	return string(b) == string(j.expected)
}

func jsonMatcher(t *testing.T, v interface{}) sqlmock.Argument {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return jsonArg{expected: b}
}
