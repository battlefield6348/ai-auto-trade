package postgres

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	"ai-auto-trade/internal/application/trading"
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
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("s-1"))

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
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("s-2"))

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

func TestUpdateStrategy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)
	s := tradingDomain.Strategy{
		ID:         "s-1",
		Name:       "策略A+",
		BaseSymbol: "BTCUSDT",
		Timeframe:  "1d",
		Env:        tradingDomain.EnvBoth,
		Status:     tradingDomain.StatusActive,
		Version:    1,
		UpdatedBy:  "user-2",
	}

	mock.ExpectExec("UPDATE strategies SET").
		WithArgs(
			s.Name, s.Description, s.BaseSymbol, s.Timeframe, string(s.Env), string(s.Status), s.Version,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			"user-2", "s-1",
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateStrategy(context.Background(), s)
	if err != nil {
		t.Fatalf("update strategy: %v", err)
	}
}

func TestGetStrategy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)

	rows := sqlmock.NewRows([]string{"id", "name", "slug", "description", "base_symbol", "timeframe", "env", "status", "version", "buy_conditions", "sell_conditions", "risk_settings", "created_by", "updated_by", "last_executed_at", "created_at", "updated_at"}).
		AddRow("s-1", "策略A", "slug-a", "desc", "BTCUSDT", "1d", "both", "active", 1, []byte("{}"), []byte("{}"), []byte("{}"), "u-1", "u-1", time.Now(), time.Now(), time.Now())

	mock.ExpectQuery("SELECT (.+) FROM strategies WHERE id=\\$1").
		WithArgs("s-1").
		WillReturnRows(rows)

	s, err := repo.GetStrategy(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("get strategy: %v", err)
	}
	if s.ID != "s-1" || s.Name != "策略A" {
		t.Errorf("unexpected strategy: %+v", s)
	}
}

func TestDeleteStrategy(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)

	mock.ExpectExec("DELETE FROM strategies WHERE id=\\$1").
		WithArgs("s-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.DeleteStrategy(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("delete strategy: %v", err)
	}
}

func TestListStrategies(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)

	rows := sqlmock.NewRows([]string{"id", "name", "slug", "description", "base_symbol", "timeframe", "env", "status", "version", "buy_conditions", "sell_conditions", "risk_settings", "created_by", "updated_by", "last_executed_at", "created_at", "updated_at"}).
		AddRow("s-1", "策略A", "slug-a", "desc", "BTCUSDT", "1d", "both", "active", 1, []byte("{}"), []byte("{}"), []byte("{}"), "u-1", "u-1", time.Now(), time.Now(), time.Now())

	mock.ExpectQuery("SELECT (.+) FROM strategies WHERE status = \\$1 AND env = \\$2 AND name ILIKE \\$3").
		WithArgs("active", "both", "%策略%").
		WillReturnRows(rows)

	list, err := repo.ListStrategies(context.Background(), trading.StrategyFilter{
		Status: "active",
		Env:    "both",
		Name:   "策略",
	})
	if err != nil {
		t.Fatalf("list strategies: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 strategy, got %d", len(list))
	}
}

func TestSetStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)

	mock.ExpectExec("UPDATE strategies SET status=\\$1, env=\\$2, is_active=\\$3, last_executed_at=\\$4, updated_at=NOW\\(\\) WHERE id=\\$5").
		WithArgs("active", "paper", true, sqlmock.AnyArg(), "s-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SetStatus(context.Background(), "s-1", tradingDomain.StatusActive, tradingDomain.EnvPaper)
	if err != nil {
		t.Fatalf("set status: %v", err)
	}
}

func TestSaveBacktest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)
	rec := tradingDomain.BacktestRecord{
		StrategyID:      "s-1",
		StrategyVersion: 1,
		Params: tradingDomain.BacktestParams{
			StartDate: time.Now().Add(-24 * time.Hour),
			EndDate:   time.Now(),
		},
		CreatedBy: "u-1",
	}

	mock.ExpectQuery("INSERT INTO strategy_backtests").
		WithArgs(
			rec.StrategyID, rec.StrategyVersion, rec.Params.StartDate, rec.Params.EndDate,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			"u-1",
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("b-1"))

	_, err = repo.SaveBacktest(context.Background(), rec)
	if err != nil {
		t.Fatalf("save backtest: %v", err)
	}
}

func TestSaveTrade(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)
	trade := tradingDomain.TradeRecord{
		StrategyID: "s-1",
		Env:        tradingDomain.EnvPaper,
		Symbol:     "BTCUSDT",
		Side:       "buy",
		EntryDate:  time.Now(),
		EntryPrice: 50000,
	}

	mock.ExpectExec("INSERT INTO strategy_trades").
		WithArgs(
			"s-1", trade.StrategyVersion, string(trade.Env), trade.Symbol, trade.Side,
			trade.EntryDate, trade.EntryPrice, trade.ExitDate, trade.ExitPrice,
			trade.PNL, trade.PNLPct, trade.HoldDays, trade.Reason, nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SaveTrade(context.Background(), trade)
	if err != nil {
		t.Fatalf("save trade: %v", err)
	}
}

func TestGetOpenPosition(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)

	rows := sqlmock.NewRows([]string{"id", "strategy_id", "env", "symbol", "entry_date", "entry_price", "size", "stop_loss", "take_profit", "status", "updated_at"}).
		AddRow("p-1", "s-1", "paper", "BTCUSDT", time.Now(), 50000, 0.1, nil, nil, "open", time.Now())

	mock.ExpectQuery("SELECT (.+) FROM strategy_positions WHERE strategy_id=\\$2 AND env=\\$1 AND status='open'").
		WithArgs("paper", "s-1").
		WillReturnRows(rows)

	p, err := repo.GetOpenPosition(context.Background(), "s-1", tradingDomain.EnvPaper)
	if err != nil {
		t.Fatalf("get open position: %v", err)
	}
	if p == nil || p.ID != "p-1" {
		t.Errorf("expected position p-1, got %+v", p)
	}
}

func TestUpsertPosition(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)
	p := tradingDomain.Position{
		ID:         "p-1",
		EntryDate:  time.Now(),
		EntryPrice: 50000,
		Size:       0.1,
		Status:     "open",
	}

	mock.ExpectExec("UPDATE strategy_positions SET").
		WithArgs(p.EntryDate, p.EntryPrice, p.Size, p.StopLoss, p.TakeProfit, p.Status, "p-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpsertPosition(context.Background(), p)
	if err != nil {
		t.Fatalf("upsert position: %v", err)
	}
}

func TestSaveReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)
	rep := tradingDomain.Report{
		StrategyID: "s-1",
		Env:        tradingDomain.EnvPaper,
		Summary:    tradingDomain.ReportSummary{TotalPNL: 100},
	}

	mock.ExpectQuery("INSERT INTO strategy_reports").
		WithArgs("s-1", rep.StrategyVersion, string(rep.Env), rep.PeriodStart, rep.PeriodEnd, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r-1"))

	_, err = repo.SaveReport(context.Background(), rep)
	if err != nil {
		t.Fatalf("save report: %v", err)
	}
}

func TestSaveLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)
	entry := tradingDomain.LogEntry{
		StrategyID: "s-1",
		Env:        tradingDomain.EnvPaper,
		Message:    "test log",
	}

	mock.ExpectExec("INSERT INTO strategy_logs").
		WithArgs("s-1", entry.StrategyVersion, string(entry.Env), entry.Date, entry.Phase, entry.Message, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SaveLog(context.Background(), entry)
	if err != nil {
		t.Fatalf("save log: %v", err)
	}
}

func TestListTrades(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)

	rows := sqlmock.NewRows([]string{"id", "strategy_id", "strategy_version", "env", "symbol", "side", "entry_date", "entry_price", "exit_date", "exit_price", "pnl_usdt", "pnl_pct", "hold_days", "reason", "created_at"}).
		AddRow("t-1", "s-1", 1, "paper", "BTCUSDT", "buy", time.Now(), 50000, nil, nil, nil, nil, nil, "reason", time.Now())

	mock.ExpectQuery("SELECT (.+) FROM strategy_trades WHERE strategy_id = \\$1 AND env = \\$2").
		WithArgs("s-1", "paper").
		WillReturnRows(rows)

	list, err := repo.ListTrades(context.Background(), tradingDomain.TradeFilter{
		StrategyID: "s-1",
		Env:        tradingDomain.EnvPaper,
	})
	if err != nil {
		t.Fatalf("list trades: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 trade, got %d", len(list))
	}
}

func TestListLogs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewTradingRepo(db)

	rows := sqlmock.NewRows([]string{"id", "strategy_id", "strategy_version", "env", "date", "phase", "message", "payload", "created_at"}).
		AddRow("l-1", "s-1", 1, "paper", time.Now(), "entry", "msg", []byte("{}"), time.Now())

	mock.ExpectQuery("SELECT (.+) FROM strategy_logs WHERE strategy_id = \\$1 AND env = \\$2").
		WithArgs("s-1", "paper").
		WillReturnRows(rows)

	list, err := repo.ListLogs(context.Background(), tradingDomain.LogFilter{
		StrategyID: "s-1",
		Env:        tradingDomain.EnvPaper,
	})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 log, got %d", len(list))
	}
}

// jsonEqual marshals v and returns a sqlmock argument matcher.
type jsonArg struct{ expected []byte }

func (j jsonArg) Match(v driver.Value) bool {
	b, ok := v.([]byte)
	if !ok {
		return false
	}
	// Simple string comparison for tests
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
