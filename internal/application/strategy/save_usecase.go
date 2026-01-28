package strategy

import (
	strategyDomain "ai-auto-trade/internal/domain/strategy"
	"context"
	"encoding/json"
	"fmt"
)

type SaveScoringStrategyInput struct {
	Name      string             `json:"name"`
	Slug      string             `json:"slug"`
	Threshold float64            `json:"threshold"`
	Rules     []SaveRuleInput    `json:"rules"`
}

type SaveRuleInput struct {
	ConditionName string                 `json:"condition_name"`
	Type          string                 `json:"type"`
	Params        map[string]interface{} `json:"params"`
	Weight        float64                `json:"weight"`
}

type SaveScoringStrategyUseCase struct {
	db strategyDomain.DBQueryer
}

func NewSaveScoringStrategyUseCase(db strategyDomain.DBQueryer) *SaveScoringStrategyUseCase {
	return &SaveScoringStrategyUseCase{db: db}
}

// Execute performs the save operation into the database.
func (u *SaveScoringStrategyUseCase) Execute(ctx context.Context, input SaveScoringStrategyInput) error {
	// 1. Insert or Update Strategy
	var strategyID string
	err := u.db.QueryRowContext(ctx, `
		INSERT INTO strategies (name, slug, threshold, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, threshold = EXCLUDED.threshold, updated_at = NOW()
		RETURNING id
	`, input.Name, input.Slug, input.Threshold).Scan(&strategyID)
	if err != nil {
		return fmt.Errorf("upsert strategy failed: %w", err)
	}

	// 2. Clear old rules for this strategy
	_, err = u.db.ExecContext(ctx, "DELETE FROM strategy_rules WHERE strategy_id = $1", strategyID)
	if err != nil {
		return fmt.Errorf("clear old rules failed: %w", err)
	}

	// 3. Process Rules and Conditions
	for _, r := range input.Rules {
		paramsJSON, _ := json.Marshal(r.Params)
		
		var conditionID string
		err = u.db.QueryRowContext(ctx, `
			INSERT INTO conditions (name, type, params)
			VALUES ($1, $2, $3)
			RETURNING id
		`, r.ConditionName, r.Type, paramsJSON).Scan(&conditionID)
		if err != nil {
			return fmt.Errorf("create condition failed: %w", err)
		}

		// 4. Link Rule
		_, err = u.db.ExecContext(ctx, `
			INSERT INTO strategy_rules (strategy_id, condition_id, weight)
			VALUES ($1, $2, $3)
		`, strategyID, conditionID, r.Weight)
		if err != nil {
			return fmt.Errorf("link rule failed: %w", err)
		}
	}

	return nil
}
