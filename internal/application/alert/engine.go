package alert

import (
	"context"
	"fmt"
	"time"

	"ai-auto-trade/internal/application/analysis"
	alertDomain "ai-auto-trade/internal/domain/alert"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
)

// SubscriptionRepository 管理訂閱存取。
type SubscriptionRepository interface {
	ListActive(ctx context.Context) ([]alertDomain.Subscription, error)
}

// ScreenerExecutor 封裝選股器執行。
type ScreenerExecutor interface {
	Run(ctx context.Context, input analysis.ScreenerInput) (analysis.ScreenerOutput, error)
}

// AnalysisQuery 封裝單股查詢。
type AnalysisQuery interface {
	QueryDetail(ctx context.Context, input analysis.QueryDetailInput) (analysisDomain.DailyAnalysisResult, error)
}

// SystemHealthChecker 回傳系統健康度檢查結果。
type SystemHealthChecker interface {
	Check(ctx context.Context, date time.Time) ([]alertDomain.SystemMetric, error)
}

// Notifier 寄送通知。
type Notifier interface {
	Send(ctx context.Context, notification alertDomain.Notification) error
}

// Engine 執行所有訂閱，產生並送出通知。
type Engine struct {
	subsRepo   SubscriptionRepository
	screener   ScreenerExecutor
	analysis   AnalysisQuery
	health     SystemHealthChecker
	notifier   Notifier
	now        func() time.Time
	resultSize int
}

// NewEngine 建立通知引擎。
func NewEngine(subs SubscriptionRepository, screener ScreenerExecutor, analysis AnalysisQuery, health SystemHealthChecker, notifier Notifier) *Engine {
	return &Engine{
		subsRepo:   subs,
		screener:   screener,
		analysis:   analysis,
		health:     health,
		notifier:   notifier,
		now:        time.Now,
		resultSize: 10,
	}
}

// Run 執行當日所有訂閱與通知。
func (e *Engine) Run(ctx context.Context, date time.Time) error {
	subs, err := e.subsRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list subscriptions: %w", err)
	}

	for _, sub := range subs {
		if err := sub.Validate(); err != nil {
			continue // 跳過無效訂閱，避免中斷
		}

		switch sub.Type {
		case alertDomain.SubscriptionScreener:
			if err := e.handleScreener(ctx, sub, date); err != nil {
				continue
			}
		case alertDomain.SubscriptionStock:
			if err := e.handleStock(ctx, sub, date); err != nil {
				continue
			}
		case alertDomain.SubscriptionSystem:
			if err := e.handleSystem(ctx, sub, date); err != nil {
				continue
			}
		default:
			continue
		}
	}
	return nil
}

func (e *Engine) handleScreener(ctx context.Context, sub alertDomain.Subscription, date time.Time) error {
	out, err := e.screener.Run(ctx, analysis.ScreenerInput{
		Date:       date,
		Logic:      sub.Logic,
		Conditions: sub.Conditions,
		Pagination: analysis.Pagination{Offset: 0, Limit: e.resultSize},
	})
	if err != nil || out.Total == 0 || out.Total < sub.Threshold {
		return err
	}

	notification := alertDomain.Notification{
		SubscriptionID:   sub.ID,
		SubscriptionName: sub.Name,
		Type:             sub.Type,
		Date:             date,
		Message:          fmt.Sprintf("%s 命中 %d 檔", sub.Name, out.Total),
		Stocks:           mapStocks(out.Results),
	}

	return e.sendAll(ctx, sub, notification)
}

func (e *Engine) handleStock(ctx context.Context, sub alertDomain.Subscription, date time.Time) error {
	if sub.Symbol == "" {
		return fmt.Errorf("symbol required")
	}
	result, err := e.analysis.QueryDetail(ctx, analysis.QueryDetailInput{
		Symbol: sub.Symbol,
		Date:   date,
	})
	if err != nil {
		return err
	}
	if !matchConditions(result, sub.Conditions, sub.Logic) {
		return nil
	}

	notification := alertDomain.Notification{
		SubscriptionID:   sub.ID,
		SubscriptionName: sub.Name,
		Type:             sub.Type,
		Date:             date,
		Message:          fmt.Sprintf("%s 命中條件", sub.Symbol),
		Stocks:           mapStocks([]analysisDomain.DailyAnalysisResult{result}),
	}
	return e.sendAll(ctx, sub, notification)
}

func (e *Engine) handleSystem(ctx context.Context, sub alertDomain.Subscription, date time.Time) error {
	metrics, err := e.health.Check(ctx, date)
	if err != nil {
		return err
	}
	if len(metrics) == 0 {
		return nil
	}

	for _, m := range metrics {
		notification := alertDomain.Notification{
			SubscriptionID:   sub.ID,
			SubscriptionName: sub.Name,
			Type:             sub.Type,
			Date:             date,
			Message:          fmt.Sprintf("系統警報: %s %.2f", m.Metric, m.Value),
			SystemMetric:     &m,
		}
		if err := e.sendAll(ctx, sub, notification); err != nil {
			continue
		}
	}
	return nil
}

func (e *Engine) sendAll(ctx context.Context, sub alertDomain.Subscription, notif alertDomain.Notification) error {
	for _, ch := range sub.Channels {
		n := notif
		n.Channel = ch
		if err := e.notifier.Send(ctx, n); err != nil {
			continue
		}
	}
	return nil
}

func matchConditions(r analysisDomain.DailyAnalysisResult, conditions []analysis.Condition, logic analysis.BoolLogic) bool {
	if len(conditions) == 0 {
		return true
	}
	return analysis.MatchConditions(r, conditions, logic)
}

func mapStocks(results []analysisDomain.DailyAnalysisResult) []alertDomain.StockSummary {
	max := 10
	if len(results) < max {
		max = len(results)
	}
	out := make([]alertDomain.StockSummary, 0, max)
	for i := 0; i < max; i++ {
		r := results[i]
		out = append(out, alertDomain.StockSummary{
			Symbol: r.Symbol,
			Name:   "", // 未來接入基本資料
			Market: string(r.Market),
			Close:  r.Close,
			Score:  r.Score,
			Tags:   r.Tags,
		})
	}
	return out
}

