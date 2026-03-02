package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repo 提供 Postgres 資料存取，涵蓋 ingestion / analysis 讀寫與查詢。
type Repo struct {
	db *gorm.DB
}

// NewRepo 建立 Postgres 資料存取實例。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// UpsertTradingPair 以 trading_pair + market_type 作為唯一鍵，回傳 id。
func (r *Repo) UpsertTradingPair(ctx context.Context, pair, name string, market dataDomain.Market, industry string) (string, error) {
	m := StockModel{
		TradingPair: pair,
		MarketType:  string(market),
		NameZh:      name,
		Industry:    industry,
		Status:      "active",
	}

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "trading_pair"}, {Name: "market_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"name_zh", "industry", "updated_at"}),
	}).Create(&m).Error

	if err != nil {
		return "", err
	}
	return m.ID, nil
}

// InsertDailyPrice 寫入或更新單日日 K。
func (r *Repo) InsertDailyPrice(ctx context.Context, stockID string, price dataDomain.DailyPrice) error {
	m := DailyPriceModel{
		StockID:        stockID,
		Timeframe:      price.Timeframe,
		TradeDate:      price.TradeDate,
		OpenPrice:      price.Open,
		HighPrice:      price.High,
		LowPrice:       price.Low,
		ClosePrice:     price.Close,
		Volume:         price.Volume,
		Turnover:       price.Turnover,
		Change:         price.Change,
		ChangePercent:  price.ChangeRate,
		IsDividendDate: price.IsExDividend,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "stock_id"}, {Name: "timeframe"}, {Name: "trade_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"open_price", "high_price", "low_price", "close_price", "volume", "turnover", "change", "change_percent", "is_dividend_date", "updated_at"}),
	}).Create(&m).Error
}

// PricesByDate 取某交易日全市場日 K。
func (r *Repo) PricesByDate(ctx context.Context, date time.Time) ([]dataDomain.DailyPrice, error) {
	type result struct {
		TradingPair string `gorm:"column:trading_pair"`
		MarketType  string `gorm:"column:market_type"`
		Timeframe   string
		TradeDate   time.Time
		OpenPrice   float64
		HighPrice   float64
		LowPrice    float64
		ClosePrice  float64
		Volume      int64
	}
	var rawResults []result
	err := r.db.WithContext(ctx).Table("daily_prices").
		Select("stocks.trading_pair, stocks.market_type, daily_prices.timeframe, daily_prices.trade_date, daily_prices.open_price, daily_prices.high_price, daily_prices.low_price, daily_prices.close_price, daily_prices.volume").
		Joins("JOIN stocks ON daily_prices.stock_id = stocks.id").
		Where("daily_prices.trade_date = ?", date).
		Order("stocks.trading_pair").
		Scan(&rawResults).Error

	if err != nil {
		return nil, err
	}

	out := make([]dataDomain.DailyPrice, len(rawResults))
	for i, r := range rawResults {
		out[i] = dataDomain.DailyPrice{
			Symbol:    r.TradingPair,
			Market:    dataDomain.Market(r.MarketType),
			Timeframe: r.Timeframe,
			TradeDate: r.TradeDate,
			Open:      r.OpenPrice,
			High:      r.HighPrice,
			Low:       r.LowPrice,
			Close:     r.ClosePrice,
			Volume:    r.Volume,
		}
	}
	return out, nil
}

// PricesByPair 取單檔交易對歷史日 K（遞增日期）。
func (r *Repo) PricesByPair(ctx context.Context, pair string, timeframe string) ([]dataDomain.DailyPrice, error) {
	type result struct {
		TradingPair string
		MarketType  string
		Timeframe   string
		TradeDate   time.Time
		OpenPrice   float64
		HighPrice   float64
		LowPrice    float64
		ClosePrice  float64
		Volume      int64
	}
	var rawResults []result
	err := r.db.WithContext(ctx).Table("daily_prices").
		Select("stocks.trading_pair, stocks.market_type, daily_prices.timeframe, daily_prices.trade_date, daily_prices.open_price, daily_prices.high_price, daily_prices.low_price, daily_prices.close_price, daily_prices.volume").
		Joins("JOIN stocks ON daily_prices.stock_id = stocks.id").
		Where("stocks.trading_pair = ? AND daily_prices.timeframe = ?", pair, timeframe).
		Order("daily_prices.trade_date").
		Scan(&rawResults).Error

	if err != nil {
		return nil, err
	}

	out := make([]dataDomain.DailyPrice, len(rawResults))
	for i, r := range rawResults {
		out[i] = dataDomain.DailyPrice{
			Symbol:    r.TradingPair,
			Market:    dataDomain.Market(r.MarketType),
			Timeframe: r.Timeframe,
			TradeDate: r.TradeDate,
			Open:      r.OpenPrice,
			High:      r.HighPrice,
			Low:       r.LowPrice,
			Close:     r.ClosePrice,
			Volume:    r.Volume,
		}
	}
	return out, nil
}

// InsertAnalysisResult 寫入或更新分析結果。
func (r *Repo) InsertAnalysisResult(ctx context.Context, stockID string, res analysisDomain.DailyAnalysisResult) error {
	m := AnalysisResultModel{
		StockID:          stockID,
		Timeframe:        res.Timeframe,
		TradeDate:        res.TradeDate,
		AnalysisVersion:  res.Version,
		ClosePrice:       res.Close,
		Change:           res.Change,
		ChangePercent:    res.ChangeRate,
		Return5d:         res.Return5,
		Return20d:        res.Return20,
		Return60d:        res.Return60,
		Volume:           res.Volume,
		VolumeRatio:      res.VolumeMultiple,
		Score:            res.Score,
		Ma20:             res.MA20,
		PricePosition20d: res.RangePos20,
		High20d:          res.High20,
		Low20d:           res.Low20,
		Status:           statusValue(res.Success),
		ErrorReason:      nullableString(res.ErrorReason),
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "stock_id"}, {Name: "timeframe"}, {Name: "trade_date"}, {Name: "analysis_version"}},
		DoUpdates: clause.AssignmentColumns([]string{"close_price", "change", "change_percent", "return_5d", "return_20d", "return_60d", "volume", "volume_ratio", "score", "ma_20", "price_position_20d", "high_20d", "low_20d", "status", "error_reason", "updated_at"}),
	}).Create(&m).Error
}

// HasAnalysisForDate 判斷該日是否已有分析結果。
func (r *Repo) HasAnalysisForDate(ctx context.Context, date time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&AnalysisResultModel{}).Where("trade_date = ?", date).Count(&count).Error
	return count > 0, err
}

// LatestAnalysisDate 回傳最新分析日期。
func (r *Repo) LatestAnalysisDate(ctx context.Context) (time.Time, error) {
	var d time.Time
	err := r.db.WithContext(ctx).Model(&AnalysisResultModel{}).Select("MAX(trade_date)").Scan(&d).Error
	if err != nil {
		return time.Time{}, err
	}
	if d.IsZero() || d.Year() <= 1 {
		return time.Time{}, fmt.Errorf("no analysis data")
	}
	return d, nil
}

// GetHistory 取單檔交易對歷史日 K（供 AnalyzeUseCase 使用）。
func (r *Repo) GetHistory(ctx context.Context, symbol string, endDate time.Time, lookback int) ([]dataDomain.DailyPrice, error) {
	type result struct {
		TradingPair string
		MarketType  string
		Timeframe   string
		TradeDate   time.Time
		OpenPrice   float64
		HighPrice   float64
		LowPrice    float64
		ClosePrice  float64
		Volume      int64
	}
	var rawResults []result
	err := r.db.WithContext(ctx).Table("daily_prices").
		Select("stocks.trading_pair, stocks.market_type, daily_prices.timeframe, daily_prices.trade_date, daily_prices.open_price, daily_prices.high_price, daily_prices.low_price, daily_prices.close_price, daily_prices.volume").
		Joins("JOIN stocks ON daily_prices.stock_id = stocks.id").
		Where("stocks.trading_pair = ? AND daily_prices.trade_date <= ?", symbol, endDate).
		Order("daily_prices.trade_date DESC").
		Limit(lookback).
		Scan(&rawResults).Error

	if err != nil {
		return nil, err
	}

	out := make([]dataDomain.DailyPrice, len(rawResults))
	for i, r := range rawResults {
		out[i] = dataDomain.DailyPrice{
			Symbol:    r.TradingPair,
			Market:    dataDomain.Market(r.MarketType),
			Timeframe: r.Timeframe,
			TradeDate: r.TradeDate,
			Open:      r.OpenPrice,
			High:      r.HighPrice,
			Low:       r.LowPrice,
			Close:     r.ClosePrice,
			Volume:    r.Volume,
		}
	}
	// Sort by date ascending for analysis
	sort.Slice(out, func(i, j int) bool {
		return out[i].TradeDate.Before(out[j].TradeDate)
	})
	return out, nil
}

// ListBasicInfo 取得股票基本資料（供 AnalyzeUseCase 使用）。
func (r *Repo) ListBasicInfo(ctx context.Context, symbols []string, date time.Time) ([]analysis.BasicInfo, error) {
	var models []StockModel
	query := r.db.WithContext(ctx).Table("stocks")
	if len(symbols) > 0 {
		query = query.Where("trading_pair IN ?", symbols)
	}
	err := query.Where("status = ?", "active").Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]analysis.BasicInfo, len(models))
	for i, m := range models {
		out[i] = analysis.BasicInfo{
			Symbol:   m.TradingPair,
			Market:   dataDomain.Market(m.MarketType),
			Industry: m.Industry,
		}
	}
	return out, nil
}

// SaveDailyResult 儲存分析結果（供 AnalyzeUseCase 使用）。
func (r *Repo) SaveDailyResult(ctx context.Context, res analysisDomain.DailyAnalysisResult) error {
	var stockID string
	err := r.db.WithContext(ctx).Table("stocks").Select("id").Where("trading_pair = ?", res.Symbol).Scan(&stockID).Error
	if err != nil || stockID == "" {
		return fmt.Errorf("stock not found for %s", res.Symbol)
	}
	return r.InsertAnalysisResult(ctx, stockID, res)
}

// FindByDate 供 QueryUseCase 使用。
func (r *Repo) FindByDate(ctx context.Context, date time.Time, filter analysis.QueryFilter, sort analysis.SortOption, pagination analysis.Pagination) ([]analysisDomain.DailyAnalysisResult, int, error) {
	type result struct {
		TradingPair string
		MarketType  string
		Industry    string
		Timeframe   string
		TradeDate   time.Time
		Version     string `gorm:"column:analysis_version"`
		ClosePrice  float64
		Change      float64
		ChangePercent float64
		Return5d    *float64
		Return20d   *float64
		Return60d   *float64
		Volume      int64
		VolumeRatio *float64
		Score       float64
		Ma20        *float64
		PricePosition20d *float64
		High20d     *float64
		Low20d      *float64
		Status      string
		ErrorReason *string
	}

	var rawResults []result
	query := r.db.WithContext(ctx).Table("analysis_results ar").
		Select("s.trading_pair, s.market_type, s.industry, ar.timeframe, ar.trade_date, ar.analysis_version, ar.close_price, ar.change, ar.change_percent, ar.return_5d, ar.return_20d, ar.return_60d, ar.volume, ar.volume_ratio, ar.score, ar.ma_20, ar.price_position_20d, ar.high_20d, ar.low_20d, ar.status, ar.error_reason").
		Joins("JOIN stocks s ON ar.stock_id = s.id").
		Where("ar.trade_date = ?", date)

	if filter.OnlySuccess {
		query = query.Where("ar.status = ?", "success")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("s.trading_pair").
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Scan(&rawResults).Error

	if err != nil {
		return nil, 0, err
	}

	results := make([]analysisDomain.DailyAnalysisResult, len(rawResults))
	for i, r := range rawResults {
		res := analysisDomain.DailyAnalysisResult{
			Symbol:         r.TradingPair,
			Market:         dataDomain.Market(r.MarketType),
			Industry:       r.Industry,
			Timeframe:      r.Timeframe,
			TradeDate:      r.TradeDate,
			Version:        r.Version,
			Close:          r.ClosePrice,
			Change:         r.Change,
			ChangeRate:     r.ChangePercent,
			Return5:        r.Return5d,
			Return20:       r.Return20d,
			Return60:       r.Return60d,
			Volume:         r.Volume,
			VolumeMultiple: r.VolumeRatio,
			Score:          r.Score,
			MA20:           r.Ma20,
			RangePos20:     r.PricePosition20d,
			High20:         r.High20d,
			Low20:          r.Low20d,
			Success:        r.Status == "success",
		}
		if r.ErrorReason != nil {
			res.ErrorReason = *r.ErrorReason
		}
		if res.MA20 != nil && *res.MA20 > 0 {
			dev := (res.Close - *res.MA20) / *res.MA20
			res.Deviation20 = &dev
		}
		results[i] = res
	}

	return results, int(total), nil
}

// FindHistory 供 QueryUseCase 使用，MVP 版。
func (r *Repo) FindHistory(ctx context.Context, symbol string, timeframe string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error) {
	type result struct {
		TradingPair string
		MarketType  string
		Timeframe   string
		TradeDate   time.Time
		Version     string `gorm:"column:analysis_version"`
		ClosePrice  float64
		Change      float64
		ChangePercent float64
		Return5d    *float64
		Return20d   *float64
		Return60d   *float64
		Volume      int64
		VolumeRatio *float64
		Score       float64
		Ma20        *float64
		PricePosition20d *float64
		High20d     *float64
		Low20d      *float64
		Status      string
		ErrorReason *string
	}

	query := r.db.WithContext(ctx).Table("analysis_results ar").
		Select("s.trading_pair, s.market_type, ar.timeframe, ar.trade_date, ar.analysis_version, ar.close_price, ar.change, ar.change_percent, ar.return_5d, ar.return_20d, ar.return_60d, ar.volume, ar.volume_ratio, ar.score, ar.ma_20, ar.price_position_20d, ar.high_20d, ar.low_20d, ar.status, ar.error_reason").
		Joins("JOIN stocks s ON ar.stock_id = s.id").
		Where("s.trading_pair = ?", symbol)

	if timeframe != "" {
		query = query.Where("ar.timeframe = ?", timeframe)
	}
	if from != nil {
		query = query.Where("ar.trade_date >= ?", *from)
	}
	if to != nil {
		query = query.Where("ar.trade_date <= ?", *to)
	}
	if onlySuccess {
		query = query.Where("ar.status = ?", "success")
	}

	query = query.Order("ar.trade_date DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var rawResults []result
	if err := query.Scan(&rawResults).Error; err != nil {
		return nil, err
	}

	results := make([]analysisDomain.DailyAnalysisResult, len(rawResults))
	for i, r := range rawResults {
		res := analysisDomain.DailyAnalysisResult{
			Symbol:         r.TradingPair,
			Market:         dataDomain.Market(r.MarketType),
			Timeframe:      r.Timeframe,
			TradeDate:      r.TradeDate,
			Version:        r.Version,
			Close:          r.ClosePrice,
			Change:         r.Change,
			ChangeRate:     r.ChangePercent,
			Return5:        r.Return5d,
			Return20:       r.Return20d,
			Return60:       r.Return60d,
			Volume:         r.Volume,
			VolumeMultiple: r.VolumeRatio,
			Score:          r.Score,
			MA20:           r.Ma20,
			RangePos20:     r.PricePosition20d,
			High20:         r.High20d,
			Low20:          r.Low20d,
			Success:        r.Status == "success",
		}
		if r.ErrorReason != nil {
			res.ErrorReason = *r.ErrorReason
		}
		if res.MA20 != nil && *res.MA20 > 0 {
			dev := (res.Close - *res.MA20) / *res.MA20
			res.Deviation20 = &dev
		}
		results[i] = res
	}

	return results, nil
}

// Get 單筆查詢。
func (r *Repo) Get(ctx context.Context, symbol string, date time.Time, timeframe string) (analysisDomain.DailyAnalysisResult, error) {
	type result struct {
		TradingPair string
		MarketType  string
		Timeframe   string
		TradeDate   time.Time
		Version     string `gorm:"column:analysis_version"`
		ClosePrice  float64
		Change      float64
		ChangePercent float64
		Return5d    *float64
		Return20d   *float64
		Return60d   *float64
		Volume      int64
		VolumeRatio *float64
		Score       float64
		Ma20        *float64
		PricePosition20d *float64
		High20d     *float64
		Low20d      *float64
		Status      string
		ErrorReason *string
	}

	var rres result
	err := r.db.WithContext(ctx).Table("analysis_results ar").
		Select("s.trading_pair, s.market_type, ar.timeframe, ar.trade_date, ar.analysis_version, ar.close_price, ar.change, ar.change_percent, ar.return_5d, ar.return_20d, ar.return_60d, ar.volume, ar.volume_ratio, ar.score, ar.ma_20, ar.price_position_20d, ar.high_20d, ar.low_20d, ar.status, ar.error_reason").
		Joins("JOIN stocks s ON ar.stock_id = s.id").
		Where("s.trading_pair = ? AND ar.trade_date = ? AND ar.timeframe = ?", symbol, date, timeframe).
		First(&rres).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return analysisDomain.DailyAnalysisResult{}, fmt.Errorf("analysis result not found")
		}
		return analysisDomain.DailyAnalysisResult{}, err
	}

	res := analysisDomain.DailyAnalysisResult{
		Symbol:         rres.TradingPair,
		Market:         dataDomain.Market(rres.MarketType),
		Timeframe:      rres.Timeframe,
		TradeDate:      rres.TradeDate,
		Version:        rres.Version,
		Close:          rres.ClosePrice,
		Change:         rres.Change,
		ChangeRate:     rres.ChangePercent,
		Return5:        rres.Return5d,
		Return20:       rres.Return20d,
		Return60:       rres.Return60d,
		Volume:         rres.Volume,
		VolumeMultiple: rres.VolumeRatio,
		Score:          rres.Score,
		MA20:           rres.Ma20,
		RangePos20:     rres.PricePosition20d,
		High20:         rres.High20d,
		Low20:          rres.Low20d,
		Success:        rres.Status == "success",
	}
	if rres.ErrorReason != nil {
		res.ErrorReason = *rres.ErrorReason
	}
	if res.MA20 != nil && *res.MA20 > 0 {
		dev := (res.Close - *res.MA20) / *res.MA20
		res.Deviation20 = &dev
	}

	return res, nil
}

func statusValue(success bool) string {
	if success {
		return "success"
	}
	return "failure"
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
