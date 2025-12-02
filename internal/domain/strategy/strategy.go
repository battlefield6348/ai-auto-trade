package strategy

import (
	"fmt"
	"time"

	"ai-auto-trade/internal/application/analysis"
	alertDomain "ai-auto-trade/internal/domain/alert"
)

// Strategy 定義策略內容與狀態。
type Strategy struct {
	ID          string
	Name        string
	Description string
	Enabled     bool

	// 執行頻率：每 N 天執行一次，預設 1。
	FrequencyDays int

	Condition StrategyCondition
	Actions   []StrategyAction

	LastRun       *time.Time
	LastTriggered *time.Time
	TriggerCount  int

	CreatedAt time.Time
	UpdatedAt time.Time
}

// StrategyCondition 支援單日與跨日條件。
type StrategyCondition struct {
	Logic      analysis.BoolLogic
	Conditions []analysis.Condition
	MultiDay   *MultiDayCondition
}

// MultiDayCondition 要求條件連續成立。
type MultiDayCondition struct {
	Days          int                 // 包含當日
	Condition     analysis.Condition  // 需成立的條件（可再搭配 StrategyCondition.Conditions）
	RequireAll    bool                // 若為 true，所有天都需成立；false 則為至少首尾滿足
	AllowPartial  bool                // 保留未來彈性
	Description   string              // 文字描述
}

// StrategyAction 定義觸發後的動作。
type StrategyAction struct {
	Type       StrategyActionType
	Channel    alertDomain.Channel // for notify/webhook
	WebhookURL string
	Export     bool // 是否需要匯出清單
}

// StrategyActionType 支援的動作類型。
type StrategyActionType string

const (
	ActionNotify StrategyActionType = "notify"
	ActionExport StrategyActionType = "export"
)

// Validate 確保策略定義基本合理。
func (s Strategy) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("strategy id is required")
	}
	if s.Name == "" {
		return fmt.Errorf("strategy name is required")
	}
	if s.FrequencyDays <= 0 {
		return fmt.Errorf("frequency_days must be >= 1")
	}
	if len(s.Actions) == 0 {
		return fmt.Errorf("strategy actions required")
	}
	for _, a := range s.Actions {
		switch a.Type {
		case ActionNotify, ActionExport:
		default:
			return fmt.Errorf("unsupported action type: %s", a.Type)
		}
	}
	return nil
}
