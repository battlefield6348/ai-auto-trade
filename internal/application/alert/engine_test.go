package alert

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-auto-trade/internal/application/analysis"
	alertDomain "ai-auto-trade/internal/domain/alert"
	analysisDomain "ai-auto-trade/internal/domain/analysis"
	"ai-auto-trade/internal/domain/dataingestion"
)

type fakeSubsRepo struct {
	list []alertDomain.Subscription
	err  error
}

func (f fakeSubsRepo) ListActive(context.Context) ([]alertDomain.Subscription, error) {
	return f.list, f.err
}

type fakeScreener struct {
	out analysis.ScreenerOutput
	err error
}

func (f fakeScreener) Run(context.Context, analysis.ScreenerInput) (analysis.ScreenerOutput, error) {
	return f.out, f.err
}

type fakeAnalysisQuery struct {
	result analysisDomain.DailyAnalysisResult
	err    error
}

func (f fakeAnalysisQuery) QueryDetail(context.Context, analysis.QueryDetailInput) (analysisDomain.DailyAnalysisResult, error) {
	return f.result, f.err
}

type fakeHealth struct {
	metrics []alertDomain.SystemMetric
	err     error
}

func (f fakeHealth) Check(context.Context, time.Time) ([]alertDomain.SystemMetric, error) {
	return f.metrics, f.err
}

type fakeNotifier struct {
	sent []alertDomain.Notification
	err  error
}

func (f *fakeNotifier) Send(_ context.Context, n alertDomain.Notification) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, n)
	return nil
}

func TestEngine_ScreenerSubscription(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	screener := fakeScreener{
		out: analysis.ScreenerOutput{
			Results: []analysisDomain.DailyAnalysisResult{
				{Symbol: "2330", Market: dataingestion.MarketTWSE, TradeDate: date, Score: 80, Close: 600, Tags: []analysisDomain.Tag{analysisDomain.TagShortTermStrong}},
			},
			Total: 1,
		},
	}
	notifier := &fakeNotifier{}
	engine := NewEngine(
		fakeSubsRepo{list: []alertDomain.Subscription{{
			ID:         "sub1",
			Name:       "強勢股",
			Type:       alertDomain.SubscriptionScreener,
			Enabled:    true,
			Logic:      analysis.LogicAND,
			Conditions: []analysis.Condition{numericCondForTest(analysis.FieldScore, analysis.OpGTE, 50)},
			Channels:   []alertDomain.Channel{alertDomain.ChannelEmail},
		}}},
		screener,
		fakeAnalysisQuery{},
		fakeHealth{},
		notifier,
	)

	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.sent))
	}
	if notifier.sent[0].SubscriptionID != "sub1" {
		t.Fatalf("unexpected subscription id")
	}
}

func TestEngine_StockSubscription_NoMatch(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	engine := NewEngine(
		fakeSubsRepo{list: []alertDomain.Subscription{{
			ID:        "stock1",
			Name:      "單股監控",
			Type:      alertDomain.SubscriptionStock,
			Enabled:   true,
			Logic:     analysis.LogicAND,
			Symbol:    "2330",
			Conditions: []analysis.Condition{numericCondForTest(analysis.FieldScore, analysis.OpGTE, 90)},
			Channels:  []alertDomain.Channel{alertDomain.ChannelEmail},
		}}},
		fakeScreener{},
		fakeAnalysisQuery{
			result: analysisDomain.DailyAnalysisResult{
				Symbol:    "2330",
				Market:    dataingestion.MarketTWSE,
				TradeDate: date,
				Score:     50,
				Success:   true,
			},
		},
		fakeHealth{},
		&fakeNotifier{},
	)

	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEngine_SystemAlert(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	notifier := &fakeNotifier{}
	engine := NewEngine(
		fakeSubsRepo{list: []alertDomain.Subscription{{
			ID:       "sys1",
			Name:     "系統警報",
			Type:     alertDomain.SubscriptionSystem,
			Enabled:  true,
			Channels: []alertDomain.Channel{alertDomain.ChannelWebhook},
		}}},
		fakeScreener{},
		fakeAnalysisQuery{},
		fakeHealth{metrics: []alertDomain.SystemMetric{{Metric: "ingestion_fail_rate", Value: 0.1, Detail: "10% fail"}}},
		notifier,
	)

	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifier.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.sent))
	}
	if notifier.sent[0].SystemMetric == nil {
		t.Fatalf("expected system metric in notification")
	}
}

func TestEngine_InvalidSubscriptionIgnored(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	engine := NewEngine(
		fakeSubsRepo{list: []alertDomain.Subscription{{
			ID:       "",
			Name:     "bad",
			Type:     alertDomain.SubscriptionScreener,
			Enabled:  true,
			Channels: []alertDomain.Channel{},
		}}},
		fakeScreener{},
		fakeAnalysisQuery{},
		fakeHealth{},
		&fakeNotifier{},
	)
	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEngine_NotifierErrorContinue(t *testing.T) {
	date := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	notifier := &fakeNotifier{err: errors.New("send fail")}
	engine := NewEngine(
		fakeSubsRepo{list: []alertDomain.Subscription{{
			ID:       "sub1",
			Name:     "強勢股",
			Type:     alertDomain.SubscriptionScreener,
			Enabled:  true,
			Channels: []alertDomain.Channel{alertDomain.ChannelEmail},
		}}},
		fakeScreener{out: analysis.ScreenerOutput{Total: 1}},
		fakeAnalysisQuery{},
		fakeHealth{},
		notifier,
	)
	if err := engine.Run(context.Background(), date); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func numericCondForTest(field analysis.NumericField, op analysis.NumericOp, value float64) analysis.Condition {
	return analysis.Condition{
		Type: analysis.ConditionNumeric,
		Numeric: &analysis.NumericCondition{
			Field: field,
			Op:    op,
			Value: value,
		},
	}
}
