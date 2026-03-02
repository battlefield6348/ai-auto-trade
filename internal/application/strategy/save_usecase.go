package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SaveScoringStrategyInput struct {
	UserID        string          `json:"user_id"`
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	BaseSymbol    string          `json:"base_symbol"`
	Timeframe     string          `json:"timeframe"`
	Threshold     float64         `json:"threshold"`
	ExitThreshold float64         `json:"exit_threshold"`
	Rules         []SaveRuleInput `json:"rules"`
}

type SaveRuleInput struct {
	ConditionName string                 `json:"condition_name"`
	Type          string                 `json:"type"`
	Params        map[string]interface{} `json:"params"`
	Weight        float64                `json:"weight"`
	RuleType      string                 `json:"rule_type"` // 'entry' or 'exit'
}

type SaveScoringStrategyUseCase struct {
	db *gorm.DB
}

func NewSaveScoringStrategyUseCase(db *gorm.DB) *SaveScoringStrategyUseCase {
	return &SaveScoringStrategyUseCase{db: db}
}

func (u *SaveScoringStrategyUseCase) Execute(ctx context.Context, input SaveScoringStrategyInput) error {
	if u.db == nil {
		return fmt.Errorf("database not available")
	}
	if reflect.ValueOf(u.db).IsNil() {
		return fmt.Errorf("database storage not initialized")
	}

	hasEntry := false
	hasExit := false
	for _, r := range input.Rules {
		if r.RuleType == "entry" || r.RuleType == "" || r.RuleType == "both" {
			hasEntry = true
		}
		if r.RuleType == "exit" || r.RuleType == "both" {
			hasExit = true
		}
	}
	if !hasEntry {
		return fmt.Errorf("策略必須包含至少一個進場規則 (entry)")
	}
	if !hasExit {
		return fmt.Errorf("策略必須包含至少一個出場規則 (exit)")
	}

	var strategyID string
	err := u.db.Transaction(func(tx *gorm.DB) error {
		type Strategy struct {
			ID            string `gorm:"primaryKey;default:gen_random_uuid()"`
			UserID        string
			Name          string
			Slug          string
			Threshold     float64
			ExitThreshold float64
			BaseSymbol    string
			Timeframe     string
			Env           string
			IsActive      bool
			UpdatedAt     time.Time
		}
		s := Strategy{
			UserID:        input.UserID,
			Name:          input.Name,
			Slug:          input.Slug,
			Threshold:     input.Threshold,
			ExitThreshold: input.ExitThreshold,
			BaseSymbol:    input.BaseSymbol,
			Timeframe:     input.Timeframe,
			Env:           "both",
			IsActive:      true,
			UpdatedAt:     time.Now(),
		}

		err := tx.Table("strategies").Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "threshold", "exit_threshold", "base_symbol", "timeframe", "updated_at"}),
		}).Create(&s).Error
		if err != nil {
			return err
		}
		strategyID = s.ID

		if err := tx.Table("strategy_rules").Where("strategy_id = ?", strategyID).Delete(nil).Error; err != nil {
			return err
		}

		for _, r := range input.Rules {
			paramsJSON, _ := json.Marshal(r.Params)
			
			var conditionID string
			tx.Table("conditions").Select("id").Where("type = ? AND params::jsonb = ?::jsonb", r.Type, string(paramsJSON)).Scan(&conditionID)
			
			if conditionID == "" {
				type Condition struct {
					ID     string `gorm:"primaryKey;default:gen_random_uuid()"`
					Name   string
					Type   string
					Params []byte
				}
				c := Condition{Name: r.ConditionName, Type: r.Type, Params: paramsJSON}
				if err := tx.Table("conditions").Create(&c).Error; err != nil {
					return err
				}
				conditionID = c.ID
			}

			ruleType := r.RuleType
			if ruleType == "" {
				ruleType = "entry"
			}
			
			rule := struct {
				StrategyID  string
				ConditionID string
				Weight      float64
				RuleType    string
			}{
				StrategyID:  strategyID,
				ConditionID: conditionID,
				Weight:      r.Weight,
				RuleType:    ruleType,
			}
			if err := tx.Table("strategy_rules").Create(&rule).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("儲存策略失敗: %w", err)
	}

	fmt.Printf("[SaveStrategy] Successfully upserted strategy %s (ID: %s)\n", input.Slug, strategyID)
	return nil
}
