package trading

import (
	"fmt"
	"time"

	"ai-auto-trade/internal/application/analysis"
)

// Environment 區分策略運行環境。
type Environment string

const (
	EnvTest  Environment = "test"
	EnvProd  Environment = "prod"
	EnvReal  Environment = "real"
	EnvPaper Environment = "paper"
	EnvBoth  Environment = "both"
)

// Status 表示策略狀態。
type Status string

const (
	StatusDraft    Status = "draft"
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

// PriceMode 決定成交價採樣方式。
type PriceMode string

const (
	PriceNextOpen     PriceMode = "next_open"
	PriceNextClose    PriceMode = "next_close"
	PriceCurrentClose PriceMode = "current_close"
)

// OrderSizeMode 決定下單金額計算方式。
type OrderSizeMode string

const (
	OrderFixedUSDT     OrderSizeMode = "fixed_usdt"
	OrderPercentEquity OrderSizeMode = "percent_of_equity"
)

// ConditionSet 代表一組條件與邏輯。
type ConditionSet struct {
	Logic      analysis.BoolLogic   `json:"logic"`
	Conditions []analysis.Condition `json:"conditions"`
}

// RiskSettings 風控與下單配置。
type RiskSettings struct {
	OrderSizeMode   OrderSizeMode `json:"order_size_mode"`
	OrderSizeValue  float64       `json:"order_size_value"`
	FeesPct         float64       `json:"fees_pct"`
	SlippagePct     float64       `json:"slippage_pct"`
	StopLossPct     *float64      `json:"stop_loss_pct,omitempty"`
	TakeProfitPct   *float64      `json:"take_profit_pct,omitempty"`
	MaxDailyLossPct *float64      `json:"max_daily_loss_pct,omitempty"`
	CoolDownDays    int           `json:"cool_down_days"`
	MinHoldDays     int           `json:"min_hold_days"`
	MaxPositions    int           `json:"max_positions"`
	PriceMode          PriceMode     `json:"price_mode"`
	AutoStopMinBalance float64       `json:"auto_stop_min_balance"` // 當可用餘額低於此值時自動停止交易監控
}

// Strategy 定義買賣條件與風控設定。
type Strategy struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	BaseSymbol  string      `json:"base_symbol"`
	Timeframe   string      `json:"timeframe"`
	Env         Environment `json:"env"`
	Status      Status      `json:"status"`
	Version     int         `json:"version"`

	Buy  ConditionSet `json:"buy_conditions"`
	Sell ConditionSet `json:"sell_conditions"`
	Risk RiskSettings `json:"risk_settings"`

	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate 檢查策略基本合理性。
func (s Strategy) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("strategy name is required")
	}
	if s.BaseSymbol == "" {
		return fmt.Errorf("base_symbol is required")
	}
	switch s.Env {
	case EnvTest, EnvProd, EnvReal, EnvPaper, EnvBoth, "":
	default:
		return fmt.Errorf("unsupported env")
	}
	switch s.Status {
	case StatusDraft, StatusActive, StatusArchived, "":
	default:
		return fmt.Errorf("unsupported status")
	}
	if len(s.Buy.Conditions) == 0 || len(s.Sell.Conditions) == 0 {
		return fmt.Errorf("buy/sell conditions required")
	}
	return nil
}

// BacktestParams 回測輸入參數。
type BacktestParams struct {
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	InitialEquity   float64   `json:"initial_equity"`
	PriceMode       PriceMode `json:"price_mode"`
	FeesPct         float64   `json:"fees_pct"`
	SlippagePct     float64   `json:"slippage_pct"`
	StopLossPct     *float64  `json:"stop_loss_pct,omitempty"`
	TakeProfitPct   *float64  `json:"take_profit_pct,omitempty"`
	MaxDailyLossPct *float64  `json:"max_daily_loss_pct,omitempty"`
	CoolDownDays    int       `json:"cool_down_days"`
	MinHoldDays     int       `json:"min_hold_days"`
	MaxPositions    int       `json:"max_positions"`
	Strategy        Strategy  `json:"strategy"`
}

// BacktestTrade 模擬交易結果。
type BacktestTrade struct {
	EntryDate  time.Time `json:"entry_date"`
	EntryPrice float64   `json:"entry_price"`
	ExitDate   time.Time `json:"exit_date"`
	ExitPrice  float64   `json:"exit_price"`
	Reason     string    `json:"reason"`
	PNL        float64   `json:"pnl_usdt"`
	PNLPct     float64   `json:"pnl_pct"`
	HoldDays   int       `json:"hold_days"`
}

// EquityPoint 代表每日淨值。
type EquityPoint struct {
	Date   time.Time `json:"date"`
	Equity float64   `json:"equity"`
}

// BacktestStats 總覽統計。
type BacktestStats struct {
	TotalReturn  float64 `json:"total_return"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	WinRate      float64 `json:"win_rate"`
	TradeCount   int     `json:"trade_count"`
	AvgGain      float64 `json:"avg_gain"`
	AvgLoss      float64 `json:"avg_loss"`
	ProfitFactor float64 `json:"profit_factor"`
}

// BacktestResult 回測結果。
type BacktestResult struct {
	Trades      []BacktestTrade `json:"trades"`
	EquityCurve []EquityPoint   `json:"equity_curve"`
	Stats       BacktestStats   `json:"stats"`
}

// BacktestRecord 供儲存回測結果。
type BacktestRecord struct {
	ID              string         `json:"id"`
	StrategyID      string         `json:"strategy_id"`
	StrategyVersion int            `json:"strategy_version"`
	Params          BacktestParams `json:"params"`
	Result          BacktestResult `json:"result"`
	CreatedBy       string         `json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
}

// TradeRecord 實際或紙本交易紀錄。
type TradeRecord struct {
	ID              string      `json:"id"`
	StrategyID      string      `json:"strategy_id"`
	StrategyVersion int         `json:"strategy_version"`
	Env             Environment `json:"env"`
	Side            string      `json:"side"`
	EntryDate       time.Time   `json:"entry_date"`
	EntryPrice      float64     `json:"entry_price"`
	ExitDate        *time.Time  `json:"exit_date,omitempty"`
	ExitPrice       *float64    `json:"exit_price,omitempty"`
	PNL             *float64    `json:"pnl_usdt,omitempty"`
	PNLPct          *float64    `json:"pnl_pct,omitempty"`
	HoldDays        *int        `json:"hold_days,omitempty"`
	Reason          string      `json:"reason"`
	CreatedAt       time.Time   `json:"created_at"`
}

// TradeFilter 提供查詢交易紀錄用。
type TradeFilter struct {
	StrategyID string
	Env        Environment
	StartDate  *time.Time
	EndDate    *time.Time
}

// Position 表示當前持倉。
type Position struct {
	ID         string      `json:"id"`
	StrategyID string      `json:"strategy_id"`
	Env        Environment `json:"env"`
	EntryDate  time.Time   `json:"entry_date"`
	EntryPrice float64     `json:"entry_price"`
	Size       float64     `json:"size"`
	StopLoss   *float64    `json:"stop_loss,omitempty"`
	TakeProfit *float64    `json:"take_profit,omitempty"`
	Status     string      `json:"status"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// LogEntry 紀錄策略執行訊息。
type LogEntry struct {
	ID              string      `json:"id"`
	StrategyID      string      `json:"strategy_id"`
	StrategyVersion int         `json:"strategy_version"`
	Env             Environment `json:"env"`
	Date            time.Time   `json:"date"`
	Phase           string      `json:"phase"`
	Message         string      `json:"message"`
	Payload         interface{} `json:"payload"`
	CreatedAt       time.Time   `json:"created_at"`
}

// LogFilter 用於查詢策略日誌。
type LogFilter struct {
	StrategyID string
	Env        Environment
	Limit      int
}

// Report 代表一次報告摘要。
type Report struct {
	ID              string      `json:"id"`
	StrategyID      string      `json:"strategy_id"`
	StrategyVersion int         `json:"strategy_version"`
	Env             Environment `json:"env"`
	PeriodStart     time.Time   `json:"period_start"`
	PeriodEnd       time.Time   `json:"period_end"`
	Summary         interface{} `json:"summary"`
	TradesRef       interface{} `json:"trades_ref"`
	CreatedBy       string      `json:"created_by"`
	CreatedAt       time.Time   `json:"created_at"`
}
