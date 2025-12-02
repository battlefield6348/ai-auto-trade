package alert

import (
	"time"

	analysisDomain "ai-auto-trade/internal/domain/analysis"
)

// Notification 封裝通知內容摘要。
type Notification struct {
	SubscriptionID   string
	SubscriptionName string
	Type             SubscriptionType
	Date             time.Time
	Message          string
	Stocks           []StockSummary
	SystemMetric     *SystemMetric
	Channel          Channel
}

// StockSummary 提供通知中顯示的股票摘要。
type StockSummary struct {
	Symbol string
	Name   string
	Market string
	Close  float64
	Score  float64
	Tags   []analysisDomain.Tag
}

// SystemMetric 用於系統警報通知。
type SystemMetric struct {
	Metric string
	Value  float64
	Detail string
}
