package postgres

import (
	"encoding/json"
	"time"
)

// User 映射到 users 表
type User struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email        string `gorm:"uniqueIndex;not null"`
	DisplayName  string
	PasswordHash string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Role 映射到 roles 表
type Role struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name         string `gorm:"uniqueIndex;not null"`
	Description  string
	IsSystemRole bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserRole 映射到 user_roles 表
type UserRole struct {
	UserID string `gorm:"primaryKey"`
	RoleID string `gorm:"primaryKey"`
}

// Permission 映射到 permissions 表
type Permission struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string `gorm:"uniqueIndex;not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RolePermission 映射到 role_permissions 表
type RolePermission struct {
	RoleID       string `gorm:"primaryKey"`
	PermissionID string `gorm:"primaryKey"`
}

// StrategyModel 映射到 strategies 表
type StrategyModel struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         string `gorm:"index"`
	Name           string `gorm:"not null"`
	Slug           string `gorm:"uniqueIndex"`
	Description    string
	BaseSymbol     string
	Timeframe      string
	Env            string
	Status         string
	Version        int
	IsActive       bool
	BuyConditions  json.RawMessage `gorm:"type:jsonb"`
	SellConditions json.RawMessage `gorm:"type:jsonb"`
	RiskSettings   json.RawMessage `gorm:"type:jsonb"`
	Threshold      float64
	ExitThreshold  float64
	LastExecutedAt *time.Time
	CreatedBy      string
	UpdatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TableName 指定策略資料表名稱
func (StrategyModel) TableName() string {
	return "strategies"
}

// ConditionModel 映射到 conditions 表
type ConditionModel struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name      string          `gorm:"not null"`
	Type      string          `gorm:"not null"`
	Params    json.RawMessage `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ConditionModel) TableName() string {
	return "conditions"
}

// StrategyRuleModel 映射到 strategy_rules 表
type StrategyRuleModel struct {
	StrategyID  string  `gorm:"primaryKey"`
	ConditionID string  `gorm:"primaryKey"`
	Weight      float64 `gorm:"not null"`
	RuleType    string  `gorm:"not null"` // entry, exit, both
}

func (StrategyRuleModel) TableName() string {
	return "strategy_rules"
}

// StrategyBacktest 映射到 strategy_backtests 表
type StrategyBacktest struct {
	ID              string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	StrategyID      string          `gorm:"index"`
	StrategyVersion int             `gorm:"not null"`
	StartDate       time.Time       `gorm:"not null"`
	EndDate         time.Time       `gorm:"not null"`
	Params          json.RawMessage `gorm:"type:jsonb"`
	Stats           json.RawMessage `gorm:"type:jsonb"`
	EquityCurve     json.RawMessage `gorm:"type:jsonb"`
	Trades          json.RawMessage `gorm:"type:jsonb"`
	CreatedBy       string
	CreatedAt       time.Time
}

// StrategyTrade 映射到 strategy_trades 表
type StrategyTrade struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	StrategyID      *string `gorm:"index"` // NULL for manual
	StrategyVersion int
	Env             string
	Symbol          string
	Side            string
	EntryDate       time.Time
	EntryPrice      float64
	ExitDate        *time.Time
	ExitPrice       *float64
	PNL             *float64 `gorm:"column:pnl_usdt"`
	PNLPct          *float64 `gorm:"column:pnl_pct"`
	HoldDays        *int
	Reason          string
	ParamsSnapshot  json.RawMessage `gorm:"type:jsonb"`
	CreatedAt       time.Time
}

// StrategyPosition 映射到 strategy_positions 表
type StrategyPosition struct {
	ID         string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	StrategyID *string `gorm:"index"` // NULL for manual
	Env        string
	Symbol     string
	EntryDate  time.Time
	EntryPrice float64
	Size       float64
	StopLoss   *float64
	TakeProfit *float64
	ExitDate   *time.Time
	ExitPrice  float64
	Status     string
	UpdatedAt  time.Time
}

func (StrategyPosition) TableName() string {
	return "strategy_positions"
}

// StrategyLog 映射到 strategy_logs 表
type StrategyLog struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	StrategyID      string `gorm:"index"`
	StrategyVersion int
	Env             string
	Date            time.Time `gorm:"index"`
	Phase           string
	Message         string
	Payload         json.RawMessage `gorm:"type:jsonb"`
	CreatedAt       time.Time
}

// StrategyReport 映射到 strategy_reports 表
type StrategyReport struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	StrategyID      string `gorm:"index"`
	StrategyVersion int
	Env             string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	Summary         json.RawMessage `gorm:"type:jsonb"`
	TradesRef       json.RawMessage `gorm:"type:jsonb"`
	CreatedBy       string
	CreatedAt       time.Time
}

// AuthSession 映射到 auth_sessions 表
type AuthSession struct {
	ID              string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID          string `gorm:"index"`
	RefreshTokenID  string `gorm:"uniqueIndex"`
	ExpiresAt       time.Time
	RevokedAt       *time.Time
	UserAgent       string
	IPAddress       string
	CreatedAt       time.Time
}

// BacktestPresetModel 映射到 backtest_presets 表
type BacktestPresetModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string `gorm:"index:idx_user_name,unique"`
	Name      string `gorm:"index:idx_user_name,unique"`
	Config    json.RawMessage `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (BacktestPresetModel) TableName() string {
	return "backtest_presets"
}

// DailyPriceModel 映射到 daily_prices 表
type DailyPriceModel struct {
	ID             string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	StockID        string `gorm:"uniqueIndex:idx_stock_timeframe_date"`
	Timeframe      string `gorm:"uniqueIndex:idx_stock_timeframe_date"`
	TradeDate      time.Time `gorm:"uniqueIndex:idx_stock_timeframe_date"`
	OpenPrice      float64
	HighPrice      float64
	LowPrice       float64
	ClosePrice     float64
	Volume         int64
	Turnover       int64
	Change         float64
	ChangePercent  float64
	IsDividendDate bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (DailyPriceModel) TableName() string {
	return "daily_prices"
}

// AnalysisResultModel 映射到 analysis_results 表
type AnalysisResultModel struct {
	ID               string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	StockID          string `gorm:"uniqueIndex:idx_stock_timeframe_date_version"`
	Timeframe        string `gorm:"uniqueIndex:idx_stock_timeframe_date_version"`
	TradeDate        time.Time `gorm:"uniqueIndex:idx_stock_timeframe_date_version"`
	AnalysisVersion  string `gorm:"uniqueIndex:idx_stock_timeframe_date_version"`
	ClosePrice       float64
	Change           float64
	ChangePercent    float64
	Return5d         *float64
	Return20d        *float64
	Return60d        *float64
	Volume           int64
	VolumeRatio      *float64
	Score            float64
	Ma20             *float64
	PricePosition20d *float64
	High20d          *float64
	Low20d           *float64
	Status           string
	ErrorReason      *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (AnalysisResultModel) TableName() string {
	return "analysis_results"
}

// StockModel 映射到 stocks 表
type StockModel struct {
	ID          string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TradingPair string `gorm:"index:idx_pair_market,unique"`
	MarketType  string `gorm:"index:idx_pair_market,unique"`
	NameZh      string
	Industry    string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (StockModel) TableName() string {
	return "stocks"
}
