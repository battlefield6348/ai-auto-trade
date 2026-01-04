package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-auto-trade/internal/application/trading"
	tradingDomain "ai-auto-trade/internal/domain/trading"
)

// TradingRepo 實作 trading.Repository，使用 Postgres 儲存。
type TradingRepo struct {
	db *sql.DB
}

// NewTradingRepo 建立新實例。
func NewTradingRepo(db *sql.DB) *TradingRepo {
	return &TradingRepo{db: db}
}

// CreateStrategy 建立策略。
func (r *TradingRepo) CreateStrategy(ctx context.Context, s tradingDomain.Strategy) (string, error) {
	buyJSON, err := json.Marshal(s.Buy)
	if err != nil {
		return "", err
	}
	sellJSON, err := json.Marshal(s.Sell)
	if err != nil {
		return "", err
	}
	riskJSON, err := json.Marshal(s.Risk)
	if err != nil {
		return "", err
	}
	const q = `
INSERT INTO strategies (name, description, base_symbol, timeframe, env, status, version, buy_conditions, sell_conditions, risk_settings, user_id, created_by, updated_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
RETURNING id, created_at, updated_at;
`
	var id string
	var createdAt, updatedAt time.Time
	userID := nullableUUID(s.CreatedBy)
	if !userID.Valid {
		userID = nullableUUID(s.UpdatedBy)
	}
	if !userID.Valid {
		return "", fmt.Errorf("user_id (created_by) is required")
	}
	if err := r.db.QueryRowContext(ctx, q,
		s.Name, s.Description, s.BaseSymbol, s.Timeframe, string(s.Env), string(s.Status), s.Version,
		buyJSON, sellJSON, riskJSON, userID, nullableUUID(s.CreatedBy), nullableUUID(s.UpdatedBy),
	).Scan(&id, &createdAt, &updatedAt); err != nil {
		return "", err
	}
	return id, nil
}

// UpdateStrategy 更新策略。
func (r *TradingRepo) UpdateStrategy(ctx context.Context, s tradingDomain.Strategy) error {
	buyJSON, err := json.Marshal(s.Buy)
	if err != nil {
		return err
	}
	sellJSON, err := json.Marshal(s.Sell)
	if err != nil {
		return err
	}
	riskJSON, err := json.Marshal(s.Risk)
	if err != nil {
		return err
	}
	const q = `
UPDATE strategies
SET name=$1, description=$2, base_symbol=$3, timeframe=$4, env=$5, status=$6, version=$7,
    buy_conditions=$8, sell_conditions=$9, risk_settings=$10, updated_by=$11, updated_at=NOW()
WHERE id=$12;
`
	_, err = r.db.ExecContext(ctx, q,
		s.Name, s.Description, s.BaseSymbol, s.Timeframe, string(s.Env), string(s.Status), s.Version,
		buyJSON, sellJSON, riskJSON, nullableUUID(s.UpdatedBy), s.ID,
	)
	return err
}

// GetStrategy 取得策略。
func (r *TradingRepo) GetStrategy(ctx context.Context, id string) (tradingDomain.Strategy, error) {
	const q = `
SELECT id, name, description, base_symbol, timeframe, env, status, version, buy_conditions, sell_conditions, risk_settings, created_by, updated_by, created_at, updated_at
FROM strategies WHERE id=$1;
`
	var s tradingDomain.Strategy
	var buyRaw, sellRaw, riskRaw []byte
	var env, status string
	var createdBy, updatedBy sql.NullString
	if err := r.db.QueryRowContext(ctx, q, id).Scan(
		&s.ID, &s.Name, &s.Description, &s.BaseSymbol, &s.Timeframe,
		&env, &status, &s.Version, &buyRaw, &sellRaw, &riskRaw,
		&createdBy, &updatedBy, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return s, err
	}
	_ = json.Unmarshal(buyRaw, &s.Buy)
	_ = json.Unmarshal(sellRaw, &s.Sell)
	_ = json.Unmarshal(riskRaw, &s.Risk)
	s.Env = tradingDomain.Environment(env)
	s.Status = tradingDomain.Status(status)
	if createdBy.Valid {
		s.CreatedBy = createdBy.String
	}
	if updatedBy.Valid {
		s.UpdatedBy = updatedBy.String
	}
	return s, nil
}

// ListStrategies 列出策略。
func (r *TradingRepo) ListStrategies(ctx context.Context, filter trading.StrategyFilter) ([]tradingDomain.Strategy, error) {
	q := `
SELECT id, name, description, base_symbol, timeframe, env, status, version, buy_conditions, sell_conditions, risk_settings, created_by, updated_by, created_at, updated_at
FROM strategies
`
	conds := []string{}
	args := []interface{}{}
	if filter.Status != "" {
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, string(filter.Status))
	}
	if filter.Env != "" {
		conds = append(conds, fmt.Sprintf("env = $%d", len(args)+1))
		args = append(args, string(filter.Env))
	}
	if filter.Name != "" {
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", len(args)+1))
		args = append(args, "%"+filter.Name+"%")
	}
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY updated_at DESC LIMIT 200"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tradingDomain.Strategy
	for rows.Next() {
		var s tradingDomain.Strategy
		var buyRaw, sellRaw, riskRaw []byte
		var env, status string
		var createdBy, updatedBy sql.NullString
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Description, &s.BaseSymbol, &s.Timeframe,
			&env, &status, &s.Version, &buyRaw, &sellRaw, &riskRaw,
			&createdBy, &updatedBy, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(buyRaw, &s.Buy)
		_ = json.Unmarshal(sellRaw, &s.Sell)
		_ = json.Unmarshal(riskRaw, &s.Risk)
		s.Env = tradingDomain.Environment(env)
		s.Status = tradingDomain.Status(status)
		if createdBy.Valid {
			s.CreatedBy = createdBy.String
		}
		if updatedBy.Valid {
			s.UpdatedBy = updatedBy.String
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SetStatus 切換狀態。
func (r *TradingRepo) SetStatus(ctx context.Context, id string, status tradingDomain.Status, env tradingDomain.Environment) error {
	const q = `UPDATE strategies SET status=$1, env=$2, updated_at=NOW() WHERE id=$3;`
	_, err := r.db.ExecContext(ctx, q, string(status), string(env), id)
	return err
}

// SaveBacktest 儲存回測紀錄。
func (r *TradingRepo) SaveBacktest(ctx context.Context, rec tradingDomain.BacktestRecord) (string, error) {
	paramsJSON, err := json.Marshal(rec.Params)
	if err != nil {
		return "", err
	}
	const q = `
INSERT INTO strategy_backtests (strategy_id, strategy_version, start_date, end_date, params, stats, equity_curve, trades, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING id;
`
	statsJSON, _ := json.Marshal(rec.Result.Stats)
	equityJSON, _ := json.Marshal(rec.Result.EquityCurve)
	tradesJSON, _ := json.Marshal(rec.Result.Trades)

	var id string
	if err := r.db.QueryRowContext(ctx, q,
		rec.StrategyID, rec.StrategyVersion, rec.Params.StartDate, rec.Params.EndDate,
		paramsJSON, statsJSON, equityJSON, tradesJSON, nullableUUID(rec.CreatedBy),
	).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

// ListBacktests 取回測清單。
func (r *TradingRepo) ListBacktests(ctx context.Context, strategyID string) ([]tradingDomain.BacktestRecord, error) {
	const q = `
SELECT id, strategy_id, strategy_version, start_date, end_date, params, equity_curve, trades, stats, created_by, created_at
FROM strategy_backtests
WHERE strategy_id=$1
ORDER BY created_at DESC
LIMIT 50;
`
	rows, err := r.db.QueryContext(ctx, q, strategyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tradingDomain.BacktestRecord
	for rows.Next() {
		var rec tradingDomain.BacktestRecord
		var paramsRaw, equityRaw, tradesRaw, statsRaw []byte
		var createdBy sql.NullString
		if err := rows.Scan(
			&rec.ID, &rec.StrategyID, &rec.StrategyVersion, &rec.Params.StartDate, &rec.Params.EndDate,
			&paramsRaw, &equityRaw, &tradesRaw, &statsRaw, &createdBy, &rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(paramsRaw, &rec.Params)
		_ = json.Unmarshal(statsRaw, &rec.Result.Stats)
		_ = json.Unmarshal(equityRaw, &rec.Result.EquityCurve)
		_ = json.Unmarshal(tradesRaw, &rec.Result.Trades)
		if createdBy.Valid {
			rec.CreatedBy = createdBy.String
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// SaveTrade 寫入交易紀錄。
func (r *TradingRepo) SaveTrade(ctx context.Context, trade tradingDomain.TradeRecord) error {
	const q = `
INSERT INTO strategy_trades (strategy_id, strategy_version, env, side, entry_date, entry_price, exit_date, exit_price, pnl_usdt, pnl_pct, hold_days, reason, params_snapshot)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);
`
	_, err := r.db.ExecContext(ctx, q,
		trade.StrategyID, trade.StrategyVersion, string(trade.Env), trade.Side, trade.EntryDate, trade.EntryPrice,
		trade.ExitDate, trade.ExitPrice, trade.PNL, trade.PNLPct, trade.HoldDays, trade.Reason, nil,
	)
	return err
}

// ListTrades 查詢交易紀錄。
func (r *TradingRepo) ListTrades(ctx context.Context, filter tradingDomain.TradeFilter) ([]tradingDomain.TradeRecord, error) {
	q := `
SELECT id, strategy_id, strategy_version, env, side, entry_date, entry_price, exit_date, exit_price, pnl_usdt, pnl_pct, hold_days, reason, created_at
FROM strategy_trades
`
	conds := []string{}
	args := []interface{}{}
	if filter.StrategyID != "" {
		conds = append(conds, fmt.Sprintf("strategy_id = $%d", len(args)+1))
		args = append(args, filter.StrategyID)
	}
	if filter.Env != "" {
		conds = append(conds, fmt.Sprintf("env = $%d", len(args)+1))
		args = append(args, string(filter.Env))
	}
	if filter.StartDate != nil {
		conds = append(conds, fmt.Sprintf("entry_date >= $%d", len(args)+1))
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		conds = append(conds, fmt.Sprintf("entry_date <= $%d", len(args)+1))
		args = append(args, *filter.EndDate)
	}
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY entry_date DESC LIMIT 200"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tradingDomain.TradeRecord
	for rows.Next() {
		var rec tradingDomain.TradeRecord
		var env, side string
		if err := rows.Scan(
			&rec.ID, &rec.StrategyID, &rec.StrategyVersion, &env, &side,
			&rec.EntryDate, &rec.EntryPrice, &rec.ExitDate, &rec.ExitPrice, &rec.PNL, &rec.PNLPct, &rec.HoldDays, &rec.Reason, &rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		rec.Env = tradingDomain.Environment(env)
		rec.Side = side
		out = append(out, rec)
	}
	return out, rows.Err()
}

// GetOpenPosition 取得當前持倉。
func (r *TradingRepo) GetOpenPosition(ctx context.Context, strategyID string, env tradingDomain.Environment) (*tradingDomain.Position, error) {
	const q = `
SELECT id, strategy_id, env, entry_date, entry_price, size, stop_loss, take_profit, status, updated_at
FROM strategy_positions
WHERE strategy_id=$1 AND env=$2 AND status='open'
LIMIT 1;
`
	var p tradingDomain.Position
	var envStr, status string
	var stop, tp sql.NullFloat64
	err := r.db.QueryRowContext(ctx, q, strategyID, string(env)).Scan(
		&p.ID, &p.StrategyID, &envStr, &p.EntryDate, &p.EntryPrice, &p.Size, &stop, &tp, &status, &p.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Env = tradingDomain.Environment(envStr)
	p.Status = status
	if stop.Valid {
		p.StopLoss = &stop.Float64
	}
	if tp.Valid {
		p.TakeProfit = &tp.Float64
	}
	return &p, nil
}

// ListOpenPositions 列出所有未平倉。
func (r *TradingRepo) ListOpenPositions(ctx context.Context) ([]tradingDomain.Position, error) {
	const q = `
SELECT id, strategy_id, env, entry_date, entry_price, size, stop_loss, take_profit, status, updated_at
FROM strategy_positions
WHERE status='open'
ORDER BY updated_at DESC
LIMIT 200;
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tradingDomain.Position
	for rows.Next() {
		var p tradingDomain.Position
		var envStr, status string
		var stop, tp sql.NullFloat64
		if err := rows.Scan(&p.ID, &p.StrategyID, &envStr, &p.EntryDate, &p.EntryPrice, &p.Size, &stop, &tp, &status, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Env = tradingDomain.Environment(envStr)
		p.Status = status
		if stop.Valid {
			p.StopLoss = &stop.Float64
		}
		if tp.Valid {
			p.TakeProfit = &tp.Float64
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertPosition 新增或更新持倉。
func (r *TradingRepo) UpsertPosition(ctx context.Context, p tradingDomain.Position) error {
	if p.ID == "" {
		const q = `
INSERT INTO strategy_positions (strategy_id, env, entry_date, entry_price, size, stop_loss, take_profit, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8);
`
		_, err := r.db.ExecContext(ctx, q, p.StrategyID, string(p.Env), p.EntryDate, p.EntryPrice, p.Size, p.StopLoss, p.TakeProfit, p.Status)
		return err
	}
	const q = `
UPDATE strategy_positions
SET entry_date=$1, entry_price=$2, size=$3, stop_loss=$4, take_profit=$5, status=$6, updated_at=NOW()
WHERE id=$7;
`
	_, err := r.db.ExecContext(ctx, q, p.EntryDate, p.EntryPrice, p.Size, p.StopLoss, p.TakeProfit, p.Status, p.ID)
	return err
}

// ClosePosition 將持倉標記為結束。
func (r *TradingRepo) ClosePosition(ctx context.Context, id string, exitDate time.Time, exitPrice float64) error {
	const q = `UPDATE strategy_positions SET status='closed', updated_at=NOW() WHERE id=$1;`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// SaveLog 寫入日誌。
func (r *TradingRepo) SaveLog(ctx context.Context, log tradingDomain.LogEntry) error {
	payload, _ := json.Marshal(log.Payload)
	const q = `
INSERT INTO strategy_logs (strategy_id, strategy_version, env, date, phase, message, payload)
VALUES ($1,$2,$3,$4,$5,$6,$7);
`
	_, err := r.db.ExecContext(ctx, q, log.StrategyID, log.StrategyVersion, string(log.Env), log.Date, log.Phase, log.Message, payload)
	return err
}

// ListLogs 查詢日誌。
func (r *TradingRepo) ListLogs(ctx context.Context, filter tradingDomain.LogFilter) ([]tradingDomain.LogEntry, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	q := `
SELECT id, strategy_id, strategy_version, env, date, phase, message, payload, created_at
FROM strategy_logs
WHERE strategy_id = $1
`
	args := []interface{}{filter.StrategyID}
	if filter.Env != "" {
		q += fmt.Sprintf(" AND env = $%d", len(args)+1)
		args = append(args, string(filter.Env))
	}
	q += " ORDER BY created_at DESC"
	q += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tradingDomain.LogEntry
	for rows.Next() {
		var log tradingDomain.LogEntry
		var env sql.NullString
		var payloadRaw []byte
		if err := rows.Scan(&log.ID, &log.StrategyID, &log.StrategyVersion, &env, &log.Date, &log.Phase, &log.Message, &payloadRaw, &log.CreatedAt); err != nil {
			return nil, err
		}
		log.Env = tradingDomain.Environment(env.String)
		if len(payloadRaw) > 0 {
			_ = json.Unmarshal(payloadRaw, &log.Payload)
		}
		out = append(out, log)
	}
	return out, rows.Err()
}

// SaveReport 儲存報告。
func (r *TradingRepo) SaveReport(ctx context.Context, rep tradingDomain.Report) (string, error) {
	summary, _ := json.Marshal(rep.Summary)
	trades, _ := json.Marshal(rep.TradesRef)
	const q = `
INSERT INTO strategy_reports (strategy_id, strategy_version, env, period_start, period_end, summary, trades_ref, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id;
`
	var id string
	if err := r.db.QueryRowContext(ctx, q, rep.StrategyID, rep.StrategyVersion, string(rep.Env), rep.PeriodStart, rep.PeriodEnd, summary, trades, nullableUUID(rep.CreatedBy)).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

// ListReports 查詢報告列表。
func (r *TradingRepo) ListReports(ctx context.Context, strategyID string) ([]tradingDomain.Report, error) {
	const q = `
SELECT id, strategy_id, strategy_version, env, period_start, period_end, summary, trades_ref, created_by, created_at
FROM strategy_reports
WHERE strategy_id=$1
ORDER BY created_at DESC
LIMIT 100;
`
	rows, err := r.db.QueryContext(ctx, q, strategyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tradingDomain.Report
	for rows.Next() {
		var rep tradingDomain.Report
		var env string
		var summaryRaw, tradesRaw []byte
		var createdBy sql.NullString
		if err := rows.Scan(
			&rep.ID, &rep.StrategyID, &rep.StrategyVersion, &env, &rep.PeriodStart, &rep.PeriodEnd, &summaryRaw, &tradesRaw, &createdBy, &rep.CreatedAt,
		); err != nil {
			return nil, err
		}
		rep.Env = tradingDomain.Environment(env)
		_ = json.Unmarshal(summaryRaw, &rep.Summary)
		_ = json.Unmarshal(tradesRaw, &rep.TradesRef)
		if createdBy.Valid {
			rep.CreatedBy = createdBy.String
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

func nullableUUID(id string) sql.NullString {
	if id == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: id, Valid: true}
}
