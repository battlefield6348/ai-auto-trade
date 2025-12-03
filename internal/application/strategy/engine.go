package strategy

import (
	"context"
	"fmt"
	"time"

	"ai-auto-trade/internal/application/analysis"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	strategyDomain "ai-auto-trade/internal/domain/strategy"
)

// StrategyRepository 負責存取策略定義與執行紀錄。
type StrategyRepository interface {
	ListActive(ctx context.Context) ([]strategyDomain.Strategy, error)
	SaveRun(ctx context.Context, record RunRecord) error
	UpdateState(ctx context.Context, strategyID string, lastRun time.Time, triggered bool) error
}

// ScreenerExecutor 供策略套用當日條件。
type ScreenerExecutor interface {
	Run(ctx context.Context, input analysis.ScreenerInput) (analysis.ScreenerOutput, error)
}

// AnalysisHistoryProvider 查詢歷史分析結果。
type AnalysisHistoryProvider interface {
	QueryHistory(ctx context.Context, input analysis.QueryHistoryInput) ([]analysisDomain.DailyAnalysisResult, error)
}

// ActionDispatcher 執行策略動作（通知、匯出等）。
type ActionDispatcher interface {
	Dispatch(ctx context.Context, action strategyDomain.StrategyAction, strategy strategyDomain.Strategy, stocks []analysisDomain.DailyAnalysisResult) error
}

// RunRecord 紀錄策略執行結果摘要。
type RunRecord struct {
	StrategyID string
	Date       time.Time
	Triggered  bool
	Matched    int
	Err        string
}

// Engine 執行策略引擎。
type Engine struct {
	repo      StrategyRepository
	screener  ScreenerExecutor
	history   AnalysisHistoryProvider
	dispatch  ActionDispatcher
	now       func() time.Time
	maxResult int
}

// NewEngine 建立策略引擎，負責讀取啟用策略並執行對應篩選與動作。
func NewEngine(repo StrategyRepository, screener ScreenerExecutor, history AnalysisHistoryProvider, dispatch ActionDispatcher) *Engine {
	return &Engine{
		repo:      repo,
		screener:  screener,
		history:   history,
		dispatch:  dispatch,
		now:       time.Now,
		maxResult: 50,
	}
}

// Run 執行當日策略。
func (e *Engine) Run(ctx context.Context, date time.Time) error {
	strategies, err := e.repo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list strategies: %w", err)
	}

	for _, s := range strategies {
		e.runOne(ctx, s, date)
	}
	return nil
}

func (e *Engine) runOne(ctx context.Context, s strategyDomain.Strategy, date time.Time) {
	record := RunRecord{StrategyID: s.ID, Date: date}

	if err := s.Validate(); err != nil {
		record.Err = err.Error()
		_ = e.repo.SaveRun(ctx, record)
		return
	}

	if !shouldRun(s, date) {
		return
	}

	results, err := e.screener.Run(ctx, analysis.ScreenerInput{
		Date:       date,
		Logic:      s.Condition.Logic,
		Conditions: s.Condition.Conditions,
		Pagination: analysis.Pagination{Offset: 0, Limit: e.maxResult},
	})
	if err != nil {
		record.Err = fmt.Sprintf("screener: %v", err)
		_ = e.repo.SaveRun(ctx, record)
		return
	}

	filtered := results.Results
	if s.Condition.MultiDay != nil {
		filtered = e.filterMultiDay(ctx, filtered, date, *s.Condition.MultiDay)
	}

	if len(filtered) == 0 {
		_ = e.repo.SaveRun(ctx, record)
		_ = e.repo.UpdateState(ctx, s.ID, date, false)
		return
	}

	record.Triggered = true
	record.Matched = len(filtered)
	_ = e.repo.SaveRun(ctx, record)

	for _, action := range s.Actions {
		_ = e.dispatch.Dispatch(ctx, action, s, filtered)
	}

	_ = e.repo.UpdateState(ctx, s.ID, date, true)
}

func (e *Engine) filterMultiDay(ctx context.Context, results []analysisDomain.DailyAnalysisResult, date time.Time, cond strategyDomain.MultiDayCondition) []analysisDomain.DailyAnalysisResult {
	if cond.Days <= 1 {
		return results
	}
	out := make([]analysisDomain.DailyAnalysisResult, 0, len(results))
	for _, r := range results {
		ok := e.checkMultiDay(ctx, r.Symbol, date, cond)
		if ok {
			out = append(out, r)
		}
	}
	return out
}

func (e *Engine) checkMultiDay(ctx context.Context, symbol string, date time.Time, cond strategyDomain.MultiDayCondition) bool {
	history, err := e.history.QueryHistory(ctx, analysis.QueryHistoryInput{
		Symbol:      symbol,
		To:          &date,
		Limit:       cond.Days,
		OnlySuccess: true,
	})
	if err != nil || len(history) < cond.Days {
		return false
	}

	// 確保日期包含當日
	last := history[len(history)-1]
	if !sameDate(last.TradeDate, date) {
		return false
	}

	// 連續天數條件
	for i := len(history) - cond.Days; i < len(history); i++ {
		if !analysis.MatchConditions(history[i], []analysis.Condition{cond.Condition}, analysis.LogicAND) {
			return false
		}
	}
	return true
}

func shouldRun(s strategyDomain.Strategy, date time.Time) bool {
	if s.LastRun == nil {
		return true
	}
	delta := int(date.Sub(*s.LastRun).Hours() / 24)
	return delta >= s.FrequencyDays
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
