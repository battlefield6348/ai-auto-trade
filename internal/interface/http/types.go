package httpapi

import (
	"time"

	tradingDomain "ai-auto-trade/internal/domain/trading"
)

type analysisRunSummary struct {
	total   int
	success int
	failure int
}

type backfillFailure struct {
	TradeDate string `json:"trade_date"`
	Stage     string `json:"stage"`
	Reason    string `json:"reason"`
}

type jobRun struct {
	Kind          string
	TriggeredBy   string
	Start         time.Time
	End           time.Time
	IngestionOK   bool
	IngestionErr  string
	AnalysisOn    bool
	AnalysisOK    bool
	AnalysisTotal int
	AnalysisSucc  int
	AnalysisFail  int
	AnalysisErr   string
	Failures      []backfillFailure
	DataSource    string
}

type strategyBacktestRequest struct {
	StartDate       string                  `json:"start_date"`
	EndDate         string                  `json:"end_date"`
	InitialEquity   float64                 `json:"initial_equity"`
	FeesPct         float64                 `json:"fees_pct"`
	SlippagePct     float64                 `json:"slippage_pct"`
	PriceMode       string                  `json:"price_mode"`
	StopLossPct     *float64                `json:"stop_loss_pct"`
	TakeProfitPct   *float64                `json:"take_profit_pct"`
	MaxDailyLossPct *float64                `json:"max_daily_loss_pct"`
	CoolDownDays    int                     `json:"cool_down_days"`
	MinHoldDays     int                     `json:"min_hold_days"`
	MaxPositions    int                     `json:"max_positions"`
	Strategy        *tradingDomain.Strategy `json:"strategy,omitempty"`
}

type analysisBacktestRequest struct {
	Symbol    string             `json:"symbol"`
	StartDate string             `json:"start_date"`
	EndDate   string             `json:"end_date"`
	Entry     backtestSideParams `json:"entry"`
	Exit      backtestSideParams `json:"exit"`
	Horizons  []int              `json:"horizons"`
	Timeframe string             `json:"timeframe"`
}

type backtestSideParams struct {
	Weights    backtestWeights    `json:"weights"`
	Thresholds backtestThresholds `json:"thresholds"`
	Flags      backtestFlags      `json:"flags"`
	TotalMin   float64            `json:"total_min"`
}

type backtestWeights struct {
	Score       float64 `json:"score"`
	ChangeBonus float64 `json:"change_bonus"`
	VolumeBonus float64 `json:"volume_bonus"`
	ReturnBonus float64 `json:"return_bonus"`
	MaBonus     float64 `json:"ma_bonus"`
	AmpBonus    float64 `json:"amp_bonus"`
	RangeBonus  float64 `json:"range_bonus"`
}

type backtestThresholds struct {
	ChangeMin      float64 `json:"change_min"`
	VolumeRatioMin float64 `json:"volume_ratio_min"`
	Return5Min     float64 `json:"return5_min"`
	MaGapMin       float64 `json:"ma_gap_min"`
	AmpMin         float64 `json:"amp_min"`
	RangeMin       float64 `json:"range_min"`
}

type backtestFlags struct {
	UseChange bool `json:"use_change"`
	UseVolume bool `json:"use_volume"`
	UseReturn bool `json:"use_return"`
	UseMa     bool `json:"use_ma"`
	UseAmp    bool `json:"use_amp"`
	UseRange  bool `json:"use_range"`
}
