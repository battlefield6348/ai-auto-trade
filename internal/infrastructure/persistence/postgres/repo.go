package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
)

// Repo 提供 Postgres 資料存取，涵蓋 ingestion / analysis 讀寫與查詢。
type Repo struct {
	db *sql.DB
}

// NewRepo 建立 Postgres 資料存取實例。
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// UpsertTradingPair 以 trading_pair + market_type 作為唯一鍵，回傳 id。
func (r *Repo) UpsertTradingPair(ctx context.Context, pair, name string, market dataDomain.Market, industry string) (string, error) {
	const q = `
INSERT INTO stocks (trading_pair, market_type, name_zh, industry, status)
VALUES ($1, $2, $3, $4, 'active')
ON CONFLICT (trading_pair, market_type)
DO UPDATE SET name_zh = EXCLUDED.name_zh, industry = EXCLUDED.industry, updated_at = NOW()
RETURNING id;
`
	var id string
	if err := r.db.QueryRowContext(ctx, q, pair, string(market), name, industry).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

// InsertDailyPrice 寫入或更新單日日 K。
func (r *Repo) InsertDailyPrice(ctx context.Context, stockID string, price dataDomain.DailyPrice) error {
	const q = `
INSERT INTO daily_prices (stock_id, timeframe, trade_date, open_price, high_price, low_price, close_price, volume, turnover, change, change_percent, is_dividend_date)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (stock_id, timeframe, trade_date)
DO UPDATE SET open_price = EXCLUDED.open_price,
              high_price = EXCLUDED.high_price,
              low_price = EXCLUDED.low_price,
              close_price = EXCLUDED.close_price,
              volume = EXCLUDED.volume,
              turnover = EXCLUDED.turnover,
              change = EXCLUDED.change,
              change_percent = EXCLUDED.change_percent,
              is_dividend_date = EXCLUDED.is_dividend_date,
              updated_at = NOW();
`
	_, err := r.db.ExecContext(ctx, q,
		stockID,
		price.Timeframe,
		price.TradeDate,
		price.Open,
		price.High,
		price.Low,
		price.Close,
		price.Volume,
		price.Turnover,
		price.Change,
		price.ChangeRate,
		price.IsExDividend,
	)
	return err
}

// PricesByDate 取某交易日全市場日 K。
func (r *Repo) PricesByDate(ctx context.Context, date time.Time) ([]dataDomain.DailyPrice, error) {
	const q = `
SELECT s.trading_pair, s.market_type, dp.timeframe, dp.trade_date, dp.open_price, dp.high_price, dp.low_price, dp.close_price, dp.volume
FROM daily_prices dp
JOIN stocks s ON dp.stock_id = s.id
WHERE dp.trade_date = $1
ORDER BY s.trading_pair;
`
	rows, err := r.db.QueryContext(ctx, q, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dataDomain.DailyPrice
	for rows.Next() {
		var p dataDomain.DailyPrice
		var market string
		if err := rows.Scan(&p.Symbol, &market, &p.Timeframe, &p.TradeDate, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume); err != nil {
			return nil, err
		}
		p.Market = dataDomain.Market(market)
		out = append(out, p)
	}
	return out, rows.Err()
}

// PricesByPair 取單檔交易對歷史日 K（遞增日期）。
func (r *Repo) PricesByPair(ctx context.Context, pair string, timeframe string) ([]dataDomain.DailyPrice, error) {
	const q = `
SELECT s.trading_pair, s.market_type, dp.timeframe, dp.trade_date, dp.open_price, dp.high_price, dp.low_price, dp.close_price, dp.volume
FROM daily_prices dp
JOIN stocks s ON dp.stock_id = s.id
WHERE s.trading_pair = $1 AND dp.timeframe = $2
ORDER BY dp.trade_date;
`
	rows, err := r.db.QueryContext(ctx, q, pair, timeframe)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []dataDomain.DailyPrice
	for rows.Next() {
		var p dataDomain.DailyPrice
		var market string
		if err := rows.Scan(&p.Symbol, &market, &p.Timeframe, &p.TradeDate, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume); err != nil {
			return nil, err
		}
		p.Market = dataDomain.Market(market)
		out = append(out, p)
	}
	return out, rows.Err()
}

// InsertAnalysisResult 寫入或更新分析結果。
func (r *Repo) InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error {
	const q = `
INSERT INTO analysis_results (
    stock_id, timeframe, trade_date, analysis_version, close_price, change, change_percent,
    return_5d, return_20d, return_60d, volume, volume_ratio, score,
    ma_20, price_position_20d, high_20d, low_20d,
    status, error_reason, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13,
    $14, $15, $16, $17,
    $18, $19, NOW(), NOW()
) ON CONFLICT (stock_id, timeframe, trade_date, analysis_version)
DO UPDATE SET close_price = EXCLUDED.close_price,
              change = EXCLUDED.change,
              change_percent = EXCLUDED.change_percent,
              return_5d = EXCLUDED.return_5d,
              return_20d = EXCLUDED.return_20d,
              return_60d = EXCLUDED.return_60d,
              volume = EXCLUDED.volume,
              volume_ratio = EXCLUDED.volume_ratio,
              score = EXCLUDED.score,
              ma_20 = EXCLUDED.ma_20,
              price_position_20d = EXCLUDED.price_position_20d,
              high_20d = EXCLUDED.high_20d,
              low_20d = EXCLUDED.low_20d,
              status = EXCLUDED.status,
              error_reason = EXCLUDED.error_reason,
              updated_at = NOW();
`
	_, err := r.db.ExecContext(ctx, q,
		stockID,
		res.Timeframe,
		res.TradeDate,
		res.Version,
		res.Close,
		res.Change,
		res.ChangeRate,
		nullFloat(res.Return5),
		nullFloat(res.Return20),
		nullFloat(res.Return60),
		res.Volume,
		nullFloat(res.VolumeMultiple),
		res.Score,
		nullFloat(res.MA20),
		nullFloat(res.RangePos20),
		nullFloat(res.High20),
		nullFloat(res.Low20),
		statusValue(res.Success),
		nullableString(res.ErrorReason),
	)
	return err
}

// HasAnalysisForDate 判斷該日是否已有分析結果。
func (r *Repo) HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM analysis_results WHERE trade_date = $1);`
	var ok bool
	if err := r.db.QueryRowContext(ctx, q, date).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

// LatestAnalysisDate 回傳最新分析日期。
func (r *Repo) LatestAnalysisDate(ctx context.Context) (time.Time, error) {
	const q = `SELECT COALESCE(MAX(trade_date), '0001-01-01') FROM analysis_results;`
	var d time.Time
	if err := r.db.QueryRowContext(ctx, q).Scan(&d); err != nil {
		return time.Time{}, err
	}
	if d.IsZero() || d.Year() <= 1 {
		return time.Time{}, fmt.Errorf("no analysis data")
	}
	return d, nil
}

// FindByDate 供 QueryUseCase 使用。
func (r *Repo) FindByDate(ctx context.Context, date time.Time, filter analysis.QueryFilter, sort analysis.SortOption, pagination analysis.Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	// MVP 僅支援基本條件：OnlySuccess + 分頁
	const q = `
SELECT s.trading_pair, s.market_type, s.industry, ar.timeframe, ar.trade_date, ar.analysis_version,
       ar.close_price, ar.change, ar.change_percent, ar.return_5d, ar.return_20d, ar.return_60d,
       ar.volume, ar.volume_ratio, ar.score, ar.ma_20, ar.price_position_20d, ar.high_20d, ar.low_20d,
       ar.status, ar.error_reason
FROM analysis_results ar
JOIN stocks s ON ar.stock_id = s.id
WHERE ar.trade_date = $1
AND ($2::bool IS FALSE OR ar.status = 'success')
ORDER BY s.trading_pair
LIMIT $3 OFFSET $4;
`
	rows, err := r.db.QueryContext(ctx, q, date, filter.OnlySuccess, pagination.Limit, pagination.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []analysisDomain.DailyAnalysisResult
	for rows.Next() {
		var rres analysisDomain.DailyAnalysisResult
		var market string
		var return5, return20, return60 sql.NullFloat64
		var volRatio sql.NullFloat64
		var ma20, rangePos, high20, low20 sql.NullFloat64
		var status string
		var errReason sql.NullString
		if err := rows.Scan(
			&rres.Symbol,
			&market,
			&rres.Industry,
			&rres.Timeframe,
			&rres.TradeDate,
			&rres.Version,
			&rres.Close,
			&rres.Change,
			&rres.ChangeRate,
			&return5, &return20, &return60,
			&rres.Volume,
			&volRatio,
			&rres.Score,
			&ma20, &rangePos, &high20, &low20,
			&status,
			&errReason,
		); err != nil {
			return nil, 0, err
		}
		rres.Market = dataDomain.Market(market)
		if return5.Valid { rres.Return5 = &return5.Float64 }
		if return20.Valid { rres.Return20 = &return20.Float64 }
		if return60.Valid { rres.Return60 = &return60.Float64 }
		if volRatio.Valid { rres.VolumeMultiple = &volRatio.Float64 }
		if ma20.Valid {
			rres.MA20 = &ma20.Float64
			if ma20.Float64 > 0 {
				dev := (rres.Close - ma20.Float64) / ma20.Float64
				rres.Deviation20 = &dev
			}
		}
		if rangePos.Valid { rres.RangePos20 = &rangePos.Float64 }
		if high20.Valid { rres.High20 = &high20.Float64 }
		if low20.Valid { rres.Low20 = &low20.Float64 }
		if errReason.Valid {
			rres.ErrorReason = errReason.String
		}
		rres.Success = status == "success"
		results = append(results, rres)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// count total
	const countQ = `
SELECT count(*) FROM analysis_results ar WHERE ar.trade_date = $1 AND ($2::bool IS FALSE OR ar.status = 'success');
`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, date, filter.OnlySuccess).Scan(&total); err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// FindHistory 供 QueryUseCase 使用，MVP 版。
func (r *Repo) FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error) {
	q := `
SELECT s.trading_pair, s.market_type, ar.timeframe, ar.trade_date, ar.analysis_version,
       ar.close_price, ar.change, ar.change_percent, ar.return_5d, ar.return_20d, ar.return_60d,
       ar.volume, ar.volume_ratio, ar.score, ar.ma_20, ar.price_position_20d, ar.high_20d, ar.low_20d,
       ar.status, ar.error_reason
FROM analysis_results ar
JOIN stocks s ON ar.stock_id = s.id
WHERE s.trading_pair = $1
`
	args := []interface{}{symbol}
	if timeframe != "" {
		q += fmt.Sprintf(" AND ar.timeframe = $%d", len(args)+1)
		args = append(args, timeframe)
	}
	if from != nil {
		q += fmt.Sprintf(" AND ar.trade_date >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		q += fmt.Sprintf(" AND ar.trade_date <= $%d", len(args)+1)
		args = append(args, *to)
	}
	if onlySuccess {
		q += " AND ar.status = 'success'"
	}
	q += " ORDER BY ar.trade_date DESC"
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []analysisDomain.DailyAnalysisResult
	for rows.Next() {
		var rres analysisDomain.DailyAnalysisResult
		var market string
		var return5, return20, return60 sql.NullFloat64
		var volRatio sql.NullFloat64
		var ma20, rangePos, high20, low20 sql.NullFloat64
		var status string
		var errReason sql.NullString
		if err := rows.Scan(
			&rres.Symbol,
			&market,
			&rres.Timeframe,
			&rres.TradeDate,
			&rres.Version,
			&rres.Close,
			&rres.Change,
			&rres.ChangeRate,
			&return5, &return20, &return60,
			&rres.Volume,
			&volRatio,
			&rres.Score,
			&ma20, &rangePos, &high20, &low20,
			&status,
			&errReason,
		); err != nil {
			return nil, err
		}
		rres.Market = dataDomain.Market(market)
		if return5.Valid { rres.Return5 = &return5.Float64 }
		if return20.Valid { rres.Return20 = &return20.Float64 }
		if return60.Valid { rres.Return60 = &return60.Float64 }
		if volRatio.Valid { rres.VolumeMultiple = &volRatio.Float64 }
		if ma20.Valid {
			rres.MA20 = &ma20.Float64
			if ma20.Float64 > 0 {
				dev := (rres.Close - ma20.Float64) / ma20.Float64
				rres.Deviation20 = &dev
			}
		}
		if rangePos.Valid { rres.RangePos20 = &rangePos.Float64 }
		if high20.Valid { rres.High20 = &high20.Float64 }
		if low20.Valid { rres.Low20 = &low20.Float64 }
		if errReason.Valid {
			rres.ErrorReason = errReason.String
		}
		rres.Success = status == "success"
		results = append(results, rres)
	}
	return results, rows.Err()
}

// Get 單筆查詢。
func (r *Repo) Get(ctx context.Context, symbol string, date time.Time, timeframe string) (analysisDomain.DailyAnalysisResult, error) {
	const q = `
SELECT s.trading_pair, s.market_type, ar.timeframe, ar.trade_date, ar.analysis_version,
       ar.close_price, ar.change, ar.change_percent, ar.return_5d, ar.return_20d, ar.return_60d,
       ar.volume, ar.volume_ratio, ar.score, ar.ma_20, ar.price_position_20d, ar.high_20d, ar.low_20d,
       ar.status, ar.error_reason
FROM analysis_results ar
JOIN stocks s ON ar.stock_id = s.id
WHERE s.trading_pair = $1 AND ar.trade_date = $2 AND ar.timeframe = $3
LIMIT 1;
`
	var rres analysisDomain.DailyAnalysisResult
	var market string
	var return5, return20, return60 sql.NullFloat64
	var volRatio sql.NullFloat64
	var ma20, rangePos, high20, low20 sql.NullFloat64
	var status string
	var errReason sql.NullString
	err := r.db.QueryRowContext(ctx, q, symbol, date, timeframe).Scan(
		&rres.Symbol,
		&market,
		&rres.Timeframe,
		&rres.TradeDate,
		&rres.Version,
		&rres.Close,
		&rres.Change,
		&rres.ChangeRate,
		&return5, &return20, &return60,
		&rres.Volume,
		&volRatio,
		&rres.Score,
		&ma20, &rangePos, &high20, &low20,
		&status,
		&errReason,
	)
	if err != nil {
		return analysisDomain.DailyAnalysisResult{}, err
	}
	rres.Market = dataDomain.Market(market)
	if return5.Valid { rres.Return5 = &return5.Float64 }
	if return20.Valid { rres.Return20 = &return20.Float64 }
	if return60.Valid { rres.Return60 = &return60.Float64 }
	if volRatio.Valid { rres.VolumeMultiple = &volRatio.Float64 }
	if ma20.Valid {
		rres.MA20 = &ma20.Float64
		if ma20.Float64 > 0 {
			dev := (rres.Close - ma20.Float64) / ma20.Float64
			rres.Deviation20 = &dev
		}
	}
	if rangePos.Valid { rres.RangePos20 = &rangePos.Float64 }
	if high20.Valid { rres.High20 = &high20.Float64 }
	if low20.Valid { rres.Low20 = &low20.Float64 }
	if errReason.Valid {
		rres.ErrorReason = errReason.String
	}
	rres.Success = status == "success"
	return rres, nil
}

func nullFloat(v *float64) interface{} {
	if v == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *v, Valid: true}
}

func statusValue(success bool) string {
	if success {
		return "success"
	}
	return "failure"
}

func nullableString(s string) interface{} {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
