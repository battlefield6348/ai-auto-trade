package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ai-auto-trade/internal/application/trading"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
	tradingDomain "ai-auto-trade/internal/domain/trading"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TradingRepo 實作 trading.Repository，使用 Postgres 儲存。
type TradingRepo struct {
	db *gorm.DB
}

// NewTradingRepo 建立新實例。
func NewTradingRepo(db *gorm.DB) *TradingRepo {
	return &TradingRepo{db: db}
}

// CreateStrategy 建立策略。
func (r *TradingRepo) CreateStrategy(ctx context.Context, s tradingDomain.Strategy) (string, error) {
	buyJSON, _ := json.Marshal(s.Buy)
	sellJSON, _ := json.Marshal(s.Sell)
	riskJSON, _ := json.Marshal(s.Risk)

	m := StrategyModel{
		Name:           s.Name,
		Slug:           s.Slug,
		Description:    s.Description,
		BaseSymbol:     s.BaseSymbol,
		Timeframe:      s.Timeframe,
		Env:            string(s.Env),
		Status:         string(s.Status),
		Version:        s.Version,
		BuyConditions:  buyJSON,
		SellConditions: sellJSON,
		RiskSettings:   riskJSON,
		Threshold:      s.Threshold,
		ExitThreshold:  s.ExitThreshold,
		CreatedBy:      s.CreatedBy,
		UpdatedBy:      s.UpdatedBy,
	}

	if m.CreatedBy == "" {
		uid, err := r.fallbackUser(ctx)
		if err == nil {
			m.CreatedBy = uid
			m.UserID = uid
		}
	} else {
		m.UserID = m.CreatedBy
	}

	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return "", err
	}
	return m.ID, nil
}

// UpdateStrategy 更新策略。
func (r *TradingRepo) UpdateStrategy(ctx context.Context, s tradingDomain.Strategy) error {
	buyJSON, _ := json.Marshal(s.Buy)
	sellJSON, _ := json.Marshal(s.Sell)
	riskJSON, _ := json.Marshal(s.Risk)

	updates := map[string]interface{}{
		"name":            s.Name,
		"description":     s.Description,
		"base_symbol":     s.BaseSymbol,
		"timeframe":       s.Timeframe,
		"env":             string(s.Env),
		"status":          string(s.Status),
		"version":         s.Version,
		"buy_conditions":  buyJSON,
		"sell_conditions": sellJSON,
		"risk_settings":   riskJSON,
		"updated_by":      s.UpdatedBy,
		"threshold":       s.Threshold,
		"exit_threshold":  s.ExitThreshold,
		"updated_at":      time.Now(),
	}

	return r.db.WithContext(ctx).Model(&StrategyModel{}).Where("id = ?", s.ID).Updates(updates).Error
}

func (r *TradingRepo) GetStrategy(ctx context.Context, id string) (tradingDomain.Strategy, error) {
	return r.getStrategyByField(ctx, "id", id)
}

func (r *TradingRepo) GetStrategyBySlug(ctx context.Context, slug string) (tradingDomain.Strategy, error) {
	return r.getStrategyByField(ctx, "slug", slug)
}

func (r *TradingRepo) getStrategyByField(ctx context.Context, field, value string) (tradingDomain.Strategy, error) {
	var m StrategyModel
	err := r.db.WithContext(ctx).Where(fmt.Sprintf("%s = ?", field), value).First(&m).Error
	if err != nil {
		return tradingDomain.Strategy{}, err
	}

	var s tradingDomain.Strategy
	s.ID = m.ID
	s.Name = m.Name
	s.Slug = m.Slug
	s.Description = m.Description
	s.BaseSymbol = m.BaseSymbol
	s.Timeframe = m.Timeframe
	s.Env = tradingDomain.Environment(m.Env)
	s.Status = tradingDomain.Status(m.Status)
	s.Version = m.Version
	s.Threshold = m.Threshold
	s.ExitThreshold = m.ExitThreshold
	s.CreatedAt = m.CreatedAt
	s.UpdatedAt = m.UpdatedAt
	s.CreatedBy = m.CreatedBy
	s.UpdatedBy = m.UpdatedBy
	s.LastActivatedAt = m.LastExecutedAt

	_ = json.Unmarshal(m.BuyConditions, &s.Buy)
	_ = json.Unmarshal(m.SellConditions, &s.Sell)
	_ = json.Unmarshal(m.RiskSettings, &s.Risk)

	return s, nil
}

// DeleteStrategy 刪除策略。
func (r *TradingRepo) DeleteStrategy(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&StrategyModel{}).Error
}

// ListStrategies 列出策略。
func (r *TradingRepo) ListStrategies(ctx context.Context, filter trading.StrategyFilter) ([]tradingDomain.Strategy, error) {
	query := r.db.WithContext(ctx).Model(&StrategyModel{})

	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
	}
	if filter.Env != "" {
		query = query.Where("env = ?", string(filter.Env))
	}
	if filter.Name != "" {
		query = query.Where("name ILIKE ?", "%"+filter.Name+"%")
	}

	var models []StrategyModel
	err := query.Order("updated_at DESC").Limit(200).Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]tradingDomain.Strategy, len(models))
	for i, m := range models {
		s := tradingDomain.Strategy{
			ID:              m.ID,
			Name:            m.Name,
			Slug:            m.Slug,
			Description:     m.Description,
			BaseSymbol:      m.BaseSymbol,
			Timeframe:       m.Timeframe,
			Env:             tradingDomain.Environment(m.Env),
			Status:          tradingDomain.Status(m.Status),
			Version:         m.Version,
			Threshold:       m.Threshold,
			ExitThreshold:   m.ExitThreshold,
			CreatedAt:       m.CreatedAt,
			UpdatedAt:       m.UpdatedAt,
			CreatedBy:       m.CreatedBy,
			UpdatedBy:       m.UpdatedBy,
			LastActivatedAt: m.LastExecutedAt,
		}
		_ = json.Unmarshal(m.BuyConditions, &s.Buy)
		_ = json.Unmarshal(m.SellConditions, &s.Sell)
		_ = json.Unmarshal(m.RiskSettings, &s.Risk)
		out[i] = s
	}
	return out, nil
}

func (r *TradingRepo) SetStatus(ctx context.Context, id string, status tradingDomain.Status, env tradingDomain.Environment) error {
	isActive := status == tradingDomain.StatusActive
	updates := map[string]interface{}{
		"status":           string(status),
		"is_active":        isActive,
		"last_executed_at": time.Now(),
		"updated_at":       time.Now(),
	}
	if env != "" {
		updates["env"] = string(env)
	}
	return r.db.WithContext(ctx).Model(&StrategyModel{}).Where("id = ?", id).Updates(updates).Error
}

func (r *TradingRepo) UpdateLastActivatedAt(ctx context.Context, id string, t time.Time) error {
	return r.db.WithContext(ctx).Model(&StrategyModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_executed_at": t,
		"updated_at":       time.Now(),
	}).Error
}

func (r *TradingRepo) UpdateRiskSettings(ctx context.Context, id string, risk tradingDomain.RiskSettings) error {
	riskJSON, _ := json.Marshal(risk)
	return r.db.WithContext(ctx).Model(&StrategyModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"risk_settings": riskJSON,
		"updated_at":    time.Now(),
	}).Error
}

// SaveBacktest 儲存回測紀錄。
func (r *TradingRepo) SaveBacktest(ctx context.Context, rec tradingDomain.BacktestRecord) (string, error) {
	paramsJSON, _ := json.Marshal(rec.Params)
	statsJSON, _ := json.Marshal(rec.Result.Stats)
	equityJSON, _ := json.Marshal(rec.Result.EquityCurve)
	tradesJSON, _ := json.Marshal(rec.Result.Trades)

	m := StrategyBacktest{
		StrategyID:      rec.StrategyID,
		StrategyVersion: rec.StrategyVersion,
		StartDate:       rec.Params.StartDate,
		EndDate:         rec.Params.EndDate,
		Params:          paramsJSON,
		Stats:           statsJSON,
		EquityCurve:     equityJSON,
		Trades:          tradesJSON,
		CreatedBy:       rec.CreatedBy,
	}

	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return "", err
	}
	return m.ID, nil
}

// ListBacktests 取回測清單。
func (r *TradingRepo) ListBacktests(ctx context.Context, strategyID string) ([]tradingDomain.BacktestRecord, error) {
	var models []StrategyBacktest
	err := r.db.WithContext(ctx).Where("strategy_id = ?", strategyID).Order("created_at DESC").Limit(50).Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]tradingDomain.BacktestRecord, len(models))
	for i, m := range models {
		rec := tradingDomain.BacktestRecord{
			ID:              m.ID,
			StrategyID:      m.StrategyID,
			StrategyVersion: m.StrategyVersion,
			CreatedBy:       m.CreatedBy,
			CreatedAt:       m.CreatedAt,
		}
		_ = json.Unmarshal(m.Params, &rec.Params)
		_ = json.Unmarshal(m.Stats, &rec.Result.Stats)
		_ = json.Unmarshal(m.EquityCurve, &rec.Result.EquityCurve)
		_ = json.Unmarshal(m.Trades, &rec.Result.Trades)
		out[i] = rec
	}
	return out, nil
}

// SaveTrade 寫入交易紀錄。
func (r *TradingRepo) SaveTrade(ctx context.Context, trade tradingDomain.TradeRecord) error {
	var sid *string
	if trade.StrategyID != "" && trade.StrategyID != "manual" {
		s := trade.StrategyID
		sid = &s
	}

	m := StrategyTrade{
		StrategyID:      sid,
		StrategyVersion: trade.StrategyVersion,
		Env:             string(trade.Env),
		Symbol:          trade.Symbol,
		Side:            trade.Side,
		EntryDate:       trade.EntryDate,
		EntryPrice:      trade.EntryPrice,
		ExitDate:        trade.ExitDate,
		ExitPrice:       trade.ExitPrice,
		PNL:             trade.PNL,
		PNLPct:          trade.PNLPct,
		HoldDays:        trade.HoldDays,
		Reason:          trade.Reason,
	}

	return r.db.WithContext(ctx).Create(&m).Error
}

// ListTrades 查詢交易紀錄。
func (r *TradingRepo) ListTrades(ctx context.Context, filter tradingDomain.TradeFilter) ([]tradingDomain.TradeRecord, error) {
	query := r.db.WithContext(ctx).Model(&StrategyTrade{})

	if filter.StrategyID != "" {
		if filter.StrategyID == "manual" {
			query = query.Where("strategy_id IS NULL")
		} else {
			query = query.Where("strategy_id = ?", filter.StrategyID)
		}
	}
	if filter.Env != "" {
		query = query.Where("env = ?", string(filter.Env))
	}
	if filter.StartDate != nil {
		query = query.Where("entry_date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("entry_date <= ?", *filter.EndDate)
	}

	var models []StrategyTrade
	err := query.Order("entry_date DESC").Limit(200).Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]tradingDomain.TradeRecord, len(models))
	for i, m := range models {
		rec := tradingDomain.TradeRecord{
			ID:              m.ID,
			StrategyVersion: m.StrategyVersion,
			Env:             tradingDomain.Environment(m.Env),
			Symbol:          m.Symbol,
			Side:            m.Side,
			EntryDate:       m.EntryDate,
			EntryPrice:      m.EntryPrice,
			ExitDate:        m.ExitDate,
			ExitPrice:       m.ExitPrice,
			PNL:             m.PNL,
			PNLPct:          m.PNLPct,
			HoldDays:        m.HoldDays,
			Reason:          m.Reason,
			CreatedAt:       m.CreatedAt,
		}
		if m.StrategyID != nil {
			rec.StrategyID = *m.StrategyID
		} else {
			rec.StrategyID = "manual"
		}
		out[i] = rec
	}
	return out, nil
}

// GetOpenPosition 取得當前持倉。
func (r *TradingRepo) GetOpenPosition(ctx context.Context, strategyID string, env tradingDomain.Environment) (*tradingDomain.Position, error) {
	query := r.db.WithContext(ctx).Model(&StrategyPosition{}).Where("env = ? AND status = 'open'", string(env))

	if strategyID == "manual" {
		query = query.Where("strategy_id IS NULL")
	} else {
		query = query.Where("strategy_id = ?", strategyID)
	}

	var m StrategyPosition
	err := query.First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	p := &tradingDomain.Position{
		ID:         m.ID,
		Symbol:     m.Symbol,
		Env:        tradingDomain.Environment(m.Env),
		EntryDate:  m.EntryDate,
		EntryPrice: m.EntryPrice,
		Size:       m.Size,
		StopLoss:   m.StopLoss,
		TakeProfit: m.TakeProfit,
		Status:     m.Status,
		UpdatedAt:  m.UpdatedAt,
	}
	if m.StrategyID != nil {
		p.StrategyID = *m.StrategyID
	} else {
		p.StrategyID = "manual"
	}
	return p, nil
}

// ListOpenPositions 列出所有未平倉。
func (r *TradingRepo) ListOpenPositions(ctx context.Context) ([]tradingDomain.Position, error) {
	var models []StrategyPosition
	err := r.db.WithContext(ctx).Where("status = 'open'").Order("updated_at DESC").Limit(200).Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]tradingDomain.Position, len(models))
	for i, m := range models {
		p := tradingDomain.Position{
			ID:         m.ID,
			Symbol:     m.Symbol,
			Env:        tradingDomain.Environment(m.Env),
			EntryDate:  m.EntryDate,
			EntryPrice: m.EntryPrice,
			Size:       m.Size,
			StopLoss:   m.StopLoss,
			TakeProfit: m.TakeProfit,
			Status:     m.Status,
			UpdatedAt:  m.UpdatedAt,
		}
		if m.StrategyID != nil {
			p.StrategyID = *m.StrategyID
		} else {
			p.StrategyID = "manual"
		}
		out[i] = p
	}
	return out, nil
}

func (r *TradingRepo) GetPosition(ctx context.Context, id string) (*tradingDomain.Position, error) {
	var m StrategyPosition
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		return nil, err
	}

	p := &tradingDomain.Position{
		ID:         m.ID,
		Symbol:     m.Symbol,
		Env:        tradingDomain.Environment(m.Env),
		EntryDate:  m.EntryDate,
		EntryPrice: m.EntryPrice,
		Size:       m.Size,
		StopLoss:   m.StopLoss,
		TakeProfit: m.TakeProfit,
		Status:     m.Status,
		UpdatedAt:  m.UpdatedAt,
	}
	if m.StrategyID != nil {
		p.StrategyID = *m.StrategyID
	} else {
		p.StrategyID = "manual"
	}
	return p, nil
}

// UpsertPosition 新增或更新持倉。
func (r *TradingRepo) UpsertPosition(ctx context.Context, p tradingDomain.Position) error {
	var sid *string
	if p.StrategyID != "" && p.StrategyID != "manual" {
		s := p.StrategyID
		sid = &s
	}

	m := StrategyPosition{
		ID:         p.ID,
		StrategyID: sid,
		Env:        string(p.Env),
		Symbol:     p.Symbol,
		EntryDate:  p.EntryDate,
		EntryPrice: p.EntryPrice,
		Size:       p.Size,
		StopLoss:   p.StopLoss,
		TakeProfit: p.TakeProfit,
		Status:     p.Status,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"entry_date", "entry_price", "size", "stop_loss", "take_profit", "status", "updated_at"}),
	}).Create(&m).Error
}

// ClosePosition 將持倉標記為結束。
func (r *TradingRepo) ClosePosition(ctx context.Context, id string, exitDate time.Time, exitPrice float64) error {
	return r.db.WithContext(ctx).Model(&StrategyPosition{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     "closed",
		"exit_date":  exitDate,
		"exit_price": exitPrice,
		"updated_at": time.Now(),
	}).Error
}

// SaveLog 寫入日誌。
func (r *TradingRepo) SaveLog(ctx context.Context, log tradingDomain.LogEntry) error {
	payload, _ := json.Marshal(log.Payload)
	m := StrategyLog{
		StrategyID:      log.StrategyID,
		StrategyVersion: log.StrategyVersion,
		Env:             string(log.Env),
		Date:            log.Date,
		Phase:           log.Phase,
		Message:         log.Message,
		Payload:         payload,
	}
	return r.db.WithContext(ctx).Create(&m).Error
}

// ListLogs 查詢日誌。
func (r *TradingRepo) ListLogs(ctx context.Context, filter tradingDomain.LogFilter) ([]tradingDomain.LogEntry, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	var models []StrategyLog
	query := r.db.WithContext(ctx).Where("strategy_id = ?", filter.StrategyID)
	if filter.Env != "" {
		query = query.Where("env = ?", string(filter.Env))
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]tradingDomain.LogEntry, len(models))
	for i, m := range models {
		l := tradingDomain.LogEntry{
			ID:              m.ID,
			StrategyID:      m.StrategyID,
			StrategyVersion: m.StrategyVersion,
			Env:             tradingDomain.Environment(m.Env),
			Date:            m.Date,
			Phase:           m.Phase,
			Message:         m.Message,
			CreatedAt:       m.CreatedAt,
		}
		if len(m.Payload) > 0 {
			_ = json.Unmarshal(m.Payload, &l.Payload)
		}
		out[i] = l
	}
	return out, nil
}

// SaveReport 儲存報告。
func (r *TradingRepo) SaveReport(ctx context.Context, rep tradingDomain.Report) (string, error) {
	summary, _ := json.Marshal(rep.Summary)
	trades, _ := json.Marshal(rep.TradesRef)

	m := StrategyReport{
		StrategyID:      rep.StrategyID,
		StrategyVersion: rep.StrategyVersion,
		Env:             string(rep.Env),
		PeriodStart:     rep.PeriodStart,
		PeriodEnd:       rep.PeriodEnd,
		Summary:         summary,
		TradesRef:       trades,
		CreatedBy:       rep.CreatedBy,
	}

	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return "", err
	}
	return m.ID, nil
}

// ListReports 查詢報告列表。
func (r *TradingRepo) ListReports(ctx context.Context, strategyID string) ([]tradingDomain.Report, error) {
	var models []StrategyReport
	err := r.db.WithContext(ctx).Where("strategy_id = ?", strategyID).Order("created_at DESC").Limit(100).Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]tradingDomain.Report, len(models))
	for i, m := range models {
		rep := tradingDomain.Report{
			ID:              m.ID,
			StrategyID:      m.StrategyID,
			StrategyVersion: m.StrategyVersion,
			Env:             tradingDomain.Environment(m.Env),
			PeriodStart:     m.PeriodStart,
			PeriodEnd:       m.PeriodEnd,
			CreatedBy:       m.CreatedBy,
			CreatedAt:       m.CreatedAt,
		}
		_ = json.Unmarshal(m.Summary, &rep.Summary)
		_ = json.Unmarshal(m.TradesRef, &rep.TradesRef)
		out[i] = rep
	}
	return out, nil
}

func (r *TradingRepo) fallbackUser(ctx context.Context) (string, error) {
	var u User
	if err := r.db.WithContext(ctx).Order("created_at ASC").First(&u).Error; err != nil {
		return "", err
	}
	return u.ID, nil
}

func (r *TradingRepo) LoadScoringStrategyBySlug(ctx context.Context, slug string) (*strategyDomain.ScoringStrategy, error) {
	// 這裡保持原本的 LoadScoringStrategyBySlug 呼叫
	// 但原本的內部實現是手寫 SQL，這裡我們也應該重構它。
	// 為了保持進度，我們先重構主要的 Repo 方法。
	return strategyDomain.LoadScoringStrategyBySlugGORM(ctx, r.db, slug)
}

func (r *TradingRepo) LoadScoringStrategyByID(ctx context.Context, id string) (*strategyDomain.ScoringStrategy, error) {
	return strategyDomain.LoadScoringStrategyIDGORM(ctx, r.db, id)
}

func (r *TradingRepo) ListActiveScoringStrategies(ctx context.Context) ([]*strategyDomain.ScoringStrategy, error) {
	var slugs []string
	err := r.db.WithContext(ctx).Table("strategies").Where("is_active = ? AND slug IS NOT NULL", true).Pluck("slug", &slugs).Error
	if err != nil {
		return nil, err
	}

	var out []*strategyDomain.ScoringStrategy
	for _, slug := range slugs {
		s, err := r.LoadScoringStrategyBySlug(ctx, slug)
		if err != nil {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}
