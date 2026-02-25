package strategy

import (
	strategyDomain "ai-auto-trade/internal/domain/strategy"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
)

type SaveScoringStrategyInput struct {
	UserID    string             `json:"user_id"`
	Name      string             `json:"name"`
	Slug      string             `json:"slug"`
	BaseSymbol string            `json:"base_symbol"`
	Timeframe string             `json:"timeframe"`
	Threshold     float64            `json:"threshold"`
	ExitThreshold float64            `json:"exit_threshold"`
	Rules         []SaveRuleInput    `json:"rules"`
}

type SaveRuleInput struct {
	ConditionName string                 `json:"condition_name"`
	Type          string                 `json:"type"`
	Params        map[string]interface{} `json:"params"`
	Weight        float64                `json:"weight"`
	RuleType      string                 `json:"rule_type"` // 'entry' or 'exit'
}

type SaveScoringStrategyUseCase struct {
	db strategyDomain.DBQueryer
}

func NewSaveScoringStrategyUseCase(db strategyDomain.DBQueryer) *SaveScoringStrategyUseCase {
	return &SaveScoringStrategyUseCase{db: db}
}

// Execute performs the save operation into the database.
func (u *SaveScoringStrategyUseCase) Execute(ctx context.Context, input SaveScoringStrategyInput) error {
	if u.db == nil {
		return fmt.Errorf("database not available")
	}
	// Defensive check for typed nil
	if reflect.ValueOf(u.db).IsNil() {
		return fmt.Errorf("database storage not initialized")
	}

	// 0. Validate rules: Must have at least one entry and one exit
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

	// 1. Start Transaction if possible
	var tx *sql.Tx
	db, ok := u.db.(*sql.DB)
	if ok {
		var err error
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		defer tx.Rollback()
	}

	queryer := u.db
	if tx != nil {
		queryer = tx
	}

	// 2. Insert or Update Strategy
	var strategyID string
	err := queryer.QueryRowContext(ctx, `
		INSERT INTO strategies (user_id, name, slug, threshold, exit_threshold, base_symbol, timeframe, env, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'both', true, NOW())
		ON CONFLICT (slug) DO UPDATE SET 
			name = EXCLUDED.name, 
			threshold = EXCLUDED.threshold, 
			exit_threshold = EXCLUDED.exit_threshold,
			base_symbol = EXCLUDED.base_symbol,
			timeframe = EXCLUDED.timeframe,
			updated_at = NOW()
		RETURNING id
	`, input.UserID, input.Name, input.Slug, input.Threshold, input.ExitThreshold, input.BaseSymbol, input.Timeframe).Scan(&strategyID)
	if err != nil {
		return fmt.Errorf("儲存策略主檔失敗: %w", err)
	}

	// 3. Clear old rules
	_, err = queryer.ExecContext(ctx, "DELETE FROM strategy_rules WHERE strategy_id = $1", strategyID)
	if err != nil {
		return fmt.Errorf("清除舊規則失敗: %w", err)
	}

	// 4. Process Rules
	for _, r := range input.Rules {
		paramsJSON, _ := json.Marshal(r.Params)
		
		var conditionID string
		err = queryer.QueryRowContext(ctx, `
			SELECT id FROM conditions WHERE type = $1 AND params::jsonb = $2::jsonb
		`, r.Type, paramsJSON).Scan(&conditionID)
		
		if err == sql.ErrNoRows {
			err = queryer.QueryRowContext(ctx, `
				INSERT INTO conditions (name, type, params)
				VALUES ($1, $2, $3)
				RETURNING id
			`, r.ConditionName, r.Type, paramsJSON).Scan(&conditionID)
		}
		
		if err != nil {
			return fmt.Errorf("處理條件 [%s] 失敗: %w", r.Type, err)
		}

		ruleType := r.RuleType
		if ruleType == "" {
			ruleType = "entry"
		}
		_, err = queryer.ExecContext(ctx, `
			INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type)
			VALUES ($1, $2, $3, $4)
		`, strategyID, conditionID, r.Weight, ruleType)
		if err != nil {
			return fmt.Errorf("建立規則連結失敗: %w", err)
		}
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	fmt.Printf("[SaveStrategy] Successfully upserted strategy %s (ID: %s)\n", input.Slug, strategyID)
	return nil
}
