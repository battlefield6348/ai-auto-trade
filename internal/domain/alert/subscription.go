package alert

import (
	"fmt"
	"time"

	"ai-auto-trade/internal/application/analysis"
)

// SubscriptionType 列舉訂閱類型。
type SubscriptionType string

const (
	SubscriptionScreener SubscriptionType = "screener"
	SubscriptionStock    SubscriptionType = "stock"
	SubscriptionSystem   SubscriptionType = "system"
)

// Channel 支援的通知通道。
type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelWebhook Channel = "webhook"
	ChannelApp     Channel = "app"
)

// Subscription 定義訂閱條件與通道。
type Subscription struct {
	ID          string
	Name        string
	Type        SubscriptionType
	Enabled     bool
	Logic       analysis.BoolLogic
	Conditions  []analysis.Condition
	Symbol      string // 針對單股監控
	Threshold   int    // 選股命中門檻，0 代表只要有結果就通知
	Channels    []Channel
	WebhookURL  string
	Version     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastRunDate *time.Time
}

// Validate 基本欄位檢查。
func (s Subscription) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("id is required")
	}
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}
	switch s.Type {
	case SubscriptionScreener, SubscriptionStock, SubscriptionSystem:
	default:
		return fmt.Errorf("unsupported subscription type")
	}
	if len(s.Channels) == 0 {
		return fmt.Errorf("channels is required")
	}
	for _, ch := range s.Channels {
		switch ch {
		case ChannelEmail, ChannelWebhook, ChannelApp:
		default:
			return fmt.Errorf("unsupported channel: %s", ch)
		}
	}
	if s.Type == SubscriptionStock && s.Symbol == "" {
		return fmt.Errorf("symbol required for stock subscription")
	}
	return nil
}
