package trading

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	dataDomain "ai-auto-trade/internal/domain/dataingestion"
	tradingDomain "ai-auto-trade/internal/domain/trading"
)

// Repository 封裝策略、回測、交易等持久化。
type Repository interface {
	CreateStrategy(ctx context.Context, s tradingDomain.Strategy) (string, error)
	UpdateStrategy(ctx context.Context, s tradingDomain.Strategy) error
	GetStrategy(ctx context.Context, id string) (tradingDomain.Strategy, error)
	ListStrategies(ctx context.Context, filter StrategyFilter) ([]tradingDomain.Strategy, error)
	SetStatus(ctx context.Context, id string, status tradingDomain.Status, env tradingDomain.Environment) error

	SaveBacktest(ctx context.Context, rec tradingDomain.BacktestRecord) (string, error)
	ListBacktests(ctx context.Context, strategyID string) ([]tradingDomain.BacktestRecord, error)

	SaveTrade(ctx context.Context, trade tradingDomain.TradeRecord) error
	ListTrades(ctx context.Context, filter tradingDomain.TradeFilter) ([]tradingDomain.TradeRecord, error)
	GetOpenPosition(ctx context.Context, strategyID string, env tradingDomain.Environment) (*tradingDomain.Position, error)
	ListOpenPositions(ctx context.Context) ([]tradingDomain.Position, error)
	UpsertPosition(ctx context.Context, p tradingDomain.Position) error
	ClosePosition(ctx context.Context, id string, exitDate time.Time, exitPrice float64) error

	SaveLog(ctx context.Context, log tradingDomain.LogEntry) error
	ListLogs(ctx context.Context, filter tradingDomain.LogFilter) ([]tradingDomain.LogEntry, error)

	SaveReport(ctx context.Context, rep tradingDomain.Report) (string, error)
	ListReports(ctx context.Context, strategyID string) ([]tradingDomain.Report, error)
}

// MarketDataProvider 取得分析結果與日 K 價格。
type MarketDataProvider interface {
	FindHistory(ctx context.Context, symbol string, from, to *time.Time, limit int, onlySuccess bool) ([]analysisDomain.DailyAnalysisResult, error)
	PricesByPair(ctx context.Context, pair string) ([]dataDomain.DailyPrice, error)
}

// StrategyFilter 供列表查詢使用。
type StrategyFilter struct {
	Status tradingDomain.Status
	Env    tradingDomain.Environment
	Name   string
}

// Service 聚合策略 CRUD、回測與執行。
type Service struct {
	repo Repository
	data MarketDataProvider
	now  func() time.Time
}

// NewService 建立服務。
func NewService(repo Repository, data MarketDataProvider) *Service {
	return &Service{
		repo: repo,
		data: data,
		now:  time.Now,
	}
}

// CreateStrategy 建立新策略並設定預設值。
func (s *Service) CreateStrategy(ctx context.Context, input tradingDomain.Strategy) (tradingDomain.Strategy, error) {
	if input.BaseSymbol == "" {
		input.BaseSymbol = "BTCUSDT"
	}
	if input.Timeframe == "" {
		input.Timeframe = "1d"
	}
	if input.Env == "" {
		input.Env = tradingDomain.EnvBoth
	}
	if input.Status == "" {
		input.Status = tradingDomain.StatusDraft
	}
	if input.Version == 0 {
		input.Version = 1
	}
	if input.CreatedBy == "" && input.UpdatedBy != "" {
		input.CreatedBy = input.UpdatedBy
	}
	input.Risk = applyRiskDefaults(input.Risk)
	if err := input.Validate(); err != nil {
		return input, err
	}
	if err := validateStrategyContent(input); err != nil {
		return input, err
	}
	id, err := s.repo.CreateStrategy(ctx, input)
	if err != nil {
		return input, err
	}
	input.ID = id
	return input, nil
}

// UpdateStrategy 更新策略並自動 bump 版本。
func (s *Service) UpdateStrategy(ctx context.Context, id string, input tradingDomain.Strategy) (tradingDomain.Strategy, error) {
	current, err := s.repo.GetStrategy(ctx, id)
	if err != nil {
		return tradingDomain.Strategy{}, err
	}
	current.Name = input.Name
	current.Description = input.Description
	if input.BaseSymbol != "" {
		current.BaseSymbol = input.BaseSymbol
	}
	if input.Timeframe != "" {
		current.Timeframe = input.Timeframe
	}
	if input.Env != "" {
		current.Env = input.Env
	}
	if input.Status != "" {
		current.Status = input.Status
	}
	current.Buy = input.Buy
	current.Sell = input.Sell
	current.Risk = applyRiskDefaults(input.Risk)
	current.Version++
	current.UpdatedBy = input.UpdatedBy
	current.UpdatedAt = s.now()

	if err := current.Validate(); err != nil {
		return current, err
	}
	if err := validateStrategyContent(current); err != nil {
		return current, err
	}
	if err := s.repo.UpdateStrategy(ctx, current); err != nil {
		return current, err
	}
	return current, nil
}

// SetStatus 切換策略狀態。
func (s *Service) SetStatus(ctx context.Context, id string, status tradingDomain.Status, env tradingDomain.Environment) error {
	return s.repo.SetStatus(ctx, id, status, env)
}

// GetStrategy 取得單筆策略。
func (s *Service) GetStrategy(ctx context.Context, id string) (tradingDomain.Strategy, error) {
	return s.repo.GetStrategy(ctx, id)
}

// ListStrategies 查詢策略列表。
func (s *Service) ListStrategies(ctx context.Context, filter StrategyFilter) ([]tradingDomain.Strategy, error) {
	return s.repo.ListStrategies(ctx, filter)
}

// BacktestInput 定義回測請求。
type BacktestInput struct {
	StrategyID      string
	Inline          *tradingDomain.Strategy
	StartDate       time.Time
	EndDate         time.Time
	InitialEquity   float64
	FeesPct         *float64
	SlippagePct     *float64
	PriceMode       *tradingDomain.PriceMode
	StopLossPct     *float64
	TakeProfitPct   *float64
	MaxDailyLossPct *float64
	CoolDownDays    *int
	MinHoldDays     *int
	MaxPositions    *int
	CreatedBy       string
	Save            bool
}

// Backtest 執行回測，必要時保存結果。
func (s *Service) Backtest(ctx context.Context, input BacktestInput) (tradingDomain.BacktestRecord, error) {
	var rec tradingDomain.BacktestRecord
	if input.StartDate.IsZero() || input.EndDate.IsZero() {
		return rec, fmt.Errorf("start_date and end_date required")
	}
	if input.EndDate.Before(input.StartDate) {
		return rec, fmt.Errorf("end_date must not be before start_date")
	}
	var strategy tradingDomain.Strategy
	var err error
	if input.Inline != nil {
		strategy = *input.Inline
	} else if input.StrategyID != "" {
		strategy, err = s.repo.GetStrategy(ctx, input.StrategyID)
		if err != nil {
			return rec, err
		}
	} else {
		return rec, fmt.Errorf("strategy required")
	}

	strategy.Risk = applyRiskDefaults(strategy.Risk)
	params := mergeParams(strategy, input)

	history, prices, err := s.loadData(ctx, strategy.BaseSymbol, params.StartDate, params.EndDate)
	if err != nil {
		return rec, err
	}
	engine := backtestEngine{
		params:  params,
		history: history,
		prices:  prices,
	}
	result := engine.Run()

	rec = tradingDomain.BacktestRecord{
		StrategyID:      strategy.ID,
		StrategyVersion: strategy.Version,
		Params:          params,
		Result:          result,
		CreatedBy:       input.CreatedBy,
		CreatedAt:       s.now(),
	}

	if input.Save && strategy.ID != "" {
		id, err := s.repo.SaveBacktest(ctx, rec)
		if err != nil {
			return rec, err
		}
		rec.ID = id
	}
	return rec, nil
}

// ListBacktests 查詢策略回測紀錄。
func (s *Service) ListBacktests(ctx context.Context, strategyID string) ([]tradingDomain.BacktestRecord, error) {
	return s.repo.ListBacktests(ctx, strategyID)
}

// RunOnce 立即評估並（paper/real）下單，僅單日。
func (s *Service) RunOnce(ctx context.Context, id string, env tradingDomain.Environment, createdBy string) ([]tradingDomain.TradeRecord, error) {
	strategy, err := s.repo.GetStrategy(ctx, id)
	if err != nil {
		return nil, err
	}
	strategy.Risk = applyRiskDefaults(strategy.Risk)

	today := s.now().In(time.UTC)
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	history, prices, err := s.loadData(ctx, strategy.BaseSymbol, start.AddDate(0, 0, -5), start)
	if err != nil {
		return nil, err
	}
	engine := backtestEngine{
		params: tradingDomain.BacktestParams{
			StartDate:       start,
			EndDate:         start,
			InitialEquity:   strategy.Risk.OrderSizeValue,
			PriceMode:       strategy.Risk.PriceMode,
			FeesPct:         strategy.Risk.FeesPct,
			SlippagePct:     strategy.Risk.SlippagePct,
			StopLossPct:     strategy.Risk.StopLossPct,
			TakeProfitPct:   strategy.Risk.TakeProfitPct,
			MaxDailyLossPct: strategy.Risk.MaxDailyLossPct,
			CoolDownDays:    strategy.Risk.CoolDownDays,
			MinHoldDays:     strategy.Risk.MinHoldDays,
			MaxPositions:    1,
			Strategy:        strategy,
		},
		history: history,
		prices:  prices,
	}
	result := engine.Run()
	trades := make([]tradingDomain.TradeRecord, 0, len(result.Trades))
	for _, t := range result.Trades {
		pnl := t.PNL
		pnlPct := t.PNLPct
		hold := t.HoldDays
		trades = append(trades, tradingDomain.TradeRecord{
			StrategyID:      strategy.ID,
			StrategyVersion: strategy.Version,
			Env:             env,
			Side:            "buy",
			EntryDate:       t.EntryDate,
			EntryPrice:      t.EntryPrice,
			ExitDate:        &t.ExitDate,
			ExitPrice:       &t.ExitPrice,
			PNL:             &pnl,
			PNLPct:          &pnlPct,
			HoldDays:        &hold,
			Reason:          t.Reason,
			CreatedAt:       s.now(),
		})
	}
	for _, tr := range trades {
		_ = s.repo.SaveTrade(ctx, tr)
	}
	return trades, nil
}

// ListTrades 查詢交易紀錄。
func (s *Service) ListTrades(ctx context.Context, filter tradingDomain.TradeFilter) ([]tradingDomain.TradeRecord, error) {
	return s.repo.ListTrades(ctx, filter)
}

// ListPositions 查詢所有未平倉。
func (s *Service) ListPositions(ctx context.Context) ([]tradingDomain.Position, error) {
	return s.repo.ListOpenPositions(ctx)
}

// SaveReport 保存報告。
func (s *Service) SaveReport(ctx context.Context, rep tradingDomain.Report) (string, error) {
	if rep.CreatedAt.IsZero() {
		rep.CreatedAt = s.now()
	}
	return s.repo.SaveReport(ctx, rep)
}

// ListReports 查詢報告。
func (s *Service) ListReports(ctx context.Context, strategyID string) ([]tradingDomain.Report, error) {
	return s.repo.ListReports(ctx, strategyID)
}

// ListLogs 查詢策略日誌。
func (s *Service) ListLogs(ctx context.Context, filter tradingDomain.LogFilter) ([]tradingDomain.LogEntry, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	return s.repo.ListLogs(ctx, filter)
}

// loadData 抽取歷史分析與日 K 價格。
func (s *Service) loadData(ctx context.Context, symbol string, start, end time.Time) ([]analysisDomain.DailyAnalysisResult, []dataDomain.DailyPrice, error) {
	limit := 4000
	history, err := s.data.FindHistory(ctx, symbol, &start, &end, limit, true)
	if err != nil {
		return nil, nil, err
	}
	if len(history) == 0 {
		return nil, nil, errors.New("analysis history not found")
	}
	// 由新到舊排序，改為舊到新
	slices.SortFunc(history, func(a, b analysisDomain.DailyAnalysisResult) int {
		if a.TradeDate.Before(b.TradeDate) {
			return -1
		}
		if a.TradeDate.After(b.TradeDate) {
			return 1
		}
		return 0
	})

	prices, err := s.data.PricesByPair(ctx, symbol)
	if err != nil {
		return nil, nil, err
	}
	filteredPrices := make([]dataDomain.DailyPrice, 0, len(prices))
	for _, p := range prices {
		if (p.TradeDate.Equal(start) || p.TradeDate.After(start)) && (p.TradeDate.Equal(end) || p.TradeDate.Before(end.AddDate(0, 0, 2))) {
			filteredPrices = append(filteredPrices, p)
		}
	}
	return history, filteredPrices, nil
}

// --- Backtest Engine ---

type backtestEngine struct {
	params  tradingDomain.BacktestParams
	history []analysisDomain.DailyAnalysisResult
	prices  []dataDomain.DailyPrice
}

type positionState struct {
	entryDate  time.Time
	entryPrice float64
	qty        float64
	cost       float64
}

// Run 執行回測。
func (b backtestEngine) Run() tradingDomain.BacktestResult {
	equity := b.params.InitialEquity
	cash := b.params.InitialEquity
	var peak float64 = equity
	var trades []tradingDomain.BacktestTrade
	equityCurve := make([]tradingDomain.EquityPoint, 0, len(b.history))

	priceMap := map[string]dataDomain.DailyPrice{}
	for _, p := range b.prices {
		priceMap[p.TradeDate.Format("2006-01-02")] = p
	}

	var pos *positionState
	var lastExitDate time.Time

	for _, r := range b.history {
		if r.TradeDate.Before(b.params.StartDate) || r.TradeDate.After(b.params.EndDate) {
			continue
		}
		dateKey := r.TradeDate.Format("2006-01-02")
		price, ok := priceMap[dateKey]
		if !ok {
			continue
		}

		dayStartEquity := equity

		// 判斷賣出
		if pos != nil {
			holdDays := int(r.TradeDate.Sub(pos.entryDate).Hours() / 24)
			shouldCheckSell := holdDays >= b.params.MinHoldDays
			exitReason := ""
			closePrice := price.Close
			changePct := (closePrice - pos.entryPrice) / pos.entryPrice
			if b.params.StopLossPct != nil && changePct <= -(*b.params.StopLossPct) {
				exitReason = "stop_loss"
				shouldCheckSell = true
			}
			if b.params.TakeProfitPct != nil && changePct >= *b.params.TakeProfitPct {
				exitReason = "take_profit"
				shouldCheckSell = true
			}
			if shouldCheckSell && (exitReason != "" || analysis.MatchConditions(r, b.params.Strategy.Sell.Conditions, b.params.Strategy.Sell.Logic)) {
				exitPrice, ok := b.pickPrice(r.TradeDate, priceMap, b.params.PriceMode, price.Close)
				if !ok {
					exitPrice = price.Close
				}
				pnl, pnlPct := b.calcPnL(pos.entryPrice, exitPrice, pos.qty)
				cash += pos.qty*exitPrice + pnl
				equity = cash
				trades = append(trades, tradingDomain.BacktestTrade{
					EntryDate:  pos.entryDate,
					EntryPrice: pos.entryPrice,
					ExitDate:   r.TradeDate,
					ExitPrice:  exitPrice,
					Reason: func() string {
						if exitReason != "" {
							return exitReason
						}
						return "sell_condition"
					}(),
					PNL:      pnl,
					PNLPct:   pnlPct,
					HoldDays: holdDays,
				})
				lastExitDate = r.TradeDate
				pos = nil
			}
		}

		// 判斷買入
		if pos == nil {
			cooldown := b.params.CoolDownDays > 0 && !lastExitDate.IsZero() && int(r.TradeDate.Sub(lastExitDate).Hours()/24) <= b.params.CoolDownDays
			if b.params.MaxDailyLossPct != nil && dayStartEquity > 0 && (dayStartEquity-equity)/dayStartEquity >= *b.params.MaxDailyLossPct {
				cooldown = true
			}
			if !cooldown && analysis.MatchConditions(r, b.params.Strategy.Buy.Conditions, b.params.Strategy.Buy.Logic) {
				entryPrice, ok := b.pickPrice(r.TradeDate, priceMap, b.params.PriceMode, price.Close)
				if !ok {
					continue
				}
				orderSize := b.orderSize(equity)
				if orderSize <= 0 || entryPrice <= 0 {
					continue
				}
				qty := orderSize / entryPrice
				entryCost := qty * entryPrice
				cash -= entryCost
				pos = &positionState{
					entryDate:  r.TradeDate,
					entryPrice: entryPrice,
					qty:        qty,
					cost:       entryCost,
				}
			}
		}

		// 當日淨值
		positionValue := 0.0
		if pos != nil {
			positionValue = pos.qty * price.Close
		}
		equity = cash + positionValue
		if equity > peak {
			peak = equity
		}
		equityCurve = append(equityCurve, tradingDomain.EquityPoint{
			Date:   r.TradeDate,
			Equity: equity,
		})
	}

	stats := computeStats(trades, equityCurve, b.params.InitialEquity)
	return tradingDomain.BacktestResult{
		Trades:      trades,
		EquityCurve: equityCurve,
		Stats:       stats,
	}
}

func (b backtestEngine) orderSize(equity float64) float64 {
	switch b.params.Strategy.Risk.OrderSizeMode {
	case tradingDomain.OrderPercentEquity:
		return equity * b.params.Strategy.Risk.OrderSizeValue
	default:
		return b.params.Strategy.Risk.OrderSizeValue
	}
}

func (b backtestEngine) pickPrice(date time.Time, priceMap map[string]dataDomain.DailyPrice, mode tradingDomain.PriceMode, fallback float64) (float64, bool) {
	switch mode {
	case tradingDomain.PriceCurrentClose, "":
		return fallback, true
	case tradingDomain.PriceNextOpen:
		next := date.AddDate(0, 0, 1).Format("2006-01-02")
		if p, ok := priceMap[next]; ok && p.Open > 0 {
			return p.Open, true
		}
	case tradingDomain.PriceNextClose:
		next := date.AddDate(0, 0, 1).Format("2006-01-02")
		if p, ok := priceMap[next]; ok && p.Close > 0 {
			return p.Close, true
		}
	}
	return fallback, false
}

func (b backtestEngine) calcPnL(entry, exit, qty float64) (float64, float64) {
	if entry <= 0 || qty <= 0 {
		return 0, 0
	}
	entryPrice := entry * (1 + b.params.SlippagePct)
	exitPrice := exit * (1 - b.params.SlippagePct)
	gross := (exitPrice - entryPrice) * qty
	fee := (entryPrice*qty + exitPrice*qty) * b.params.FeesPct
	pnl := gross - fee
	pnlPct := pnl / (entryPrice * qty)
	return pnl, pnlPct
}

func computeStats(trades []tradingDomain.BacktestTrade, equity []tradingDomain.EquityPoint, initial float64) tradingDomain.BacktestStats {
	stats := tradingDomain.BacktestStats{}
	if len(equity) > 0 {
		last := equity[len(equity)-1].Equity
		stats.TotalReturn = (last / initial) - 1
		peak := initial
		maxDD := 0.0
		for _, p := range equity {
			if p.Equity > peak {
				peak = p.Equity
			}
			if peak > 0 {
				dd := (peak - p.Equity) / peak
				if dd > maxDD {
					maxDD = dd
				}
			}
		}
		stats.MaxDrawdown = maxDD
	}
	if len(trades) == 0 {
		return stats
	}
	stats.TradeCount = len(trades)
	win := 0
	gainSum := 0.0
	lossSum := 0.0
	gainCount := 0
	lossCount := 0
	for _, t := range trades {
		if t.PNL > 0 {
			win++
			gainSum += t.PNL
			gainCount++
		} else if t.PNL < 0 {
			lossSum += -t.PNL
			lossCount++
		}
	}
	stats.WinRate = float64(win) / float64(len(trades))
	if gainCount > 0 {
		stats.AvgGain = gainSum / float64(gainCount)
	}
	if lossCount > 0 {
		stats.AvgLoss = lossSum / float64(lossCount) * -1
	}
	if lossSum > 0 {
		stats.ProfitFactor = gainSum / lossSum
	}
	return stats
}

// mergeParams 將策略風控與回測輸入合併。
func mergeParams(strategy tradingDomain.Strategy, input BacktestInput) tradingDomain.BacktestParams {
	params := tradingDomain.BacktestParams{
		StartDate:       input.StartDate,
		EndDate:         input.EndDate,
		InitialEquity:   input.InitialEquity,
		PriceMode:       strategy.Risk.PriceMode,
		FeesPct:         strategy.Risk.FeesPct,
		SlippagePct:     strategy.Risk.SlippagePct,
		StopLossPct:     strategy.Risk.StopLossPct,
		TakeProfitPct:   strategy.Risk.TakeProfitPct,
		MaxDailyLossPct: strategy.Risk.MaxDailyLossPct,
		CoolDownDays:    strategy.Risk.CoolDownDays,
		MinHoldDays:     strategy.Risk.MinHoldDays,
		MaxPositions:    strategy.Risk.MaxPositions,
		Strategy:        strategy,
	}
	if params.InitialEquity == 0 {
		params.InitialEquity = 10000
	}
	if input.PriceMode != nil {
		params.PriceMode = *input.PriceMode
	}
	if input.FeesPct != nil {
		params.FeesPct = *input.FeesPct
	}
	if input.SlippagePct != nil {
		params.SlippagePct = *input.SlippagePct
	}
	if input.StopLossPct != nil {
		params.StopLossPct = input.StopLossPct
	}
	if input.TakeProfitPct != nil {
		params.TakeProfitPct = input.TakeProfitPct
	}
	if input.MaxDailyLossPct != nil {
		params.MaxDailyLossPct = input.MaxDailyLossPct
	}
	if input.CoolDownDays != nil {
		params.CoolDownDays = *input.CoolDownDays
	}
	if input.MinHoldDays != nil {
		params.MinHoldDays = *input.MinHoldDays
	}
	if input.MaxPositions != nil && *input.MaxPositions > 0 {
		params.MaxPositions = *input.MaxPositions
	}
	return params
}

func applyRiskDefaults(r tradingDomain.RiskSettings) tradingDomain.RiskSettings {
	if r.OrderSizeMode == "" {
		r.OrderSizeMode = tradingDomain.OrderFixedUSDT
	}
	if r.OrderSizeValue == 0 {
		r.OrderSizeValue = 1000
	}
	if r.PriceMode == "" {
		r.PriceMode = tradingDomain.PriceNextOpen
	}
	if r.FeesPct == 0 {
		r.FeesPct = 0.001
	}
	if r.SlippagePct == 0 {
		r.SlippagePct = 0.001
	}
	if r.MaxPositions == 0 {
		r.MaxPositions = 1
	}
	return r
}

// ToJSON helper for debugging or persistence.
func ToJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func validateStrategyContent(s tradingDomain.Strategy) error {
	if len(s.Buy.Conditions) == 0 {
		return fmt.Errorf("buy_conditions 至少需一個條件")
	}
	if len(s.Sell.Conditions) == 0 {
		return fmt.Errorf("sell_conditions 至少需一個條件")
	}
	if err := validateConditionSet(s.Buy); err != nil {
		return fmt.Errorf("buy_conditions 無效: %w", err)
	}
	if err := validateConditionSet(s.Sell); err != nil {
		return fmt.Errorf("sell_conditions 無效: %w", err)
	}
	if s.Risk.OrderSizeValue <= 0 {
		return fmt.Errorf("risk_settings.order_size_value 必須 > 0")
	}
	switch s.Risk.OrderSizeMode {
	case tradingDomain.OrderFixedUSDT, tradingDomain.OrderPercentEquity:
	default:
		return fmt.Errorf("risk_settings.order_size_mode 無效")
	}
	switch s.Risk.PriceMode {
	case tradingDomain.PriceCurrentClose, tradingDomain.PriceNextOpen, tradingDomain.PriceNextClose:
	default:
		return fmt.Errorf("risk_settings.price_mode 無效")
	}
	return nil
}

func validateConditionSet(set tradingDomain.ConditionSet) error {
	if set.Logic != analysis.LogicAND && set.Logic != analysis.LogicOR {
		return fmt.Errorf("logic 必須為 AND 或 OR")
	}
	if len(set.Conditions) == 0 {
		return fmt.Errorf("conditions 不可為空")
	}
	if len(set.Conditions) > 1 {
		return fmt.Errorf("MVP 限制僅允許 1 條條件，請精簡條件")
	}
	for i, c := range set.Conditions {
		switch c.Type {
		case analysis.ConditionNumeric:
			if c.Numeric == nil {
				return fmt.Errorf("conditions[%d] numeric 條件缺少內容", i)
			}
			if c.Numeric.Field == "" || c.Numeric.Op == "" {
				return fmt.Errorf("conditions[%d] numeric 條件缺少欄位或運算子", i)
			}
		case analysis.ConditionCategory:
			if c.Category == nil || c.Category.Field == "" || len(c.Category.Values) == 0 {
				return fmt.Errorf("conditions[%d] category 條件缺少欄位或值", i)
			}
		case analysis.ConditionTags:
			if c.Tags == nil {
				return fmt.Errorf("conditions[%d] tags 條件缺少內容", i)
			}
		case analysis.ConditionSymbols:
			if c.Symbols == nil || (len(c.Symbols.Include) == 0 && len(c.Symbols.Exclude) == 0) {
				return fmt.Errorf("conditions[%d] symbols 條件缺少代碼", i)
			}
		default:
			return fmt.Errorf("conditions[%d] 類型不支援: %s", i, c.Type)
		}
	}
	return nil
}
