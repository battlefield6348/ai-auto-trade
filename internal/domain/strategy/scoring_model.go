package strategy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ScoringStrategy represents the new three-layer strategy design.
// We use a different name to avoid conflict with the existing legacy Strategy struct if needed,
// but here we align it with the DB schema requested.
type ScoringStrategy struct {
	ID        string         `json:"id" db:"id"`
	Name      string         `json:"name" db:"name"`
	Slug      string         `json:"slug" db:"slug"`
	Threshold float64        `json:"threshold" db:"threshold"`
	Rules     []StrategyRule `json:"rules"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// Condition represents a reusable logic component.
type Condition struct {
	ID        string          `json:"id" db:"id"`
	Name      string          `json:"name" db:"name"`
	Type      string          `json:"type" db:"type"`
	ParamsRaw json.RawMessage `json:"params" db:"params"` // Using json.RawMessage to store params as requested
}

// ParseParams parses ParamsRaw into a map[string]interface{}.
func (c *Condition) ParseParams() (map[string]interface{}, error) {
	var p map[string]interface{}
	if len(c.ParamsRaw) == 0 {
		return p, nil
	}
	err := json.Unmarshal(c.ParamsRaw, &p)
	return p, err
}

// ParseParamsInto parses ParamsRaw into a specific target struct.
func (c *Condition) ParseParamsInto(target interface{}) error {
	return json.Unmarshal(c.ParamsRaw, target)
}

// StrategyRule links a Strategy to a Condition with a specific weight.
type StrategyRule struct {
	StrategyID  string    `json:"strategy_id" db:"strategy_id"`
	ConditionID string    `json:"condition_id" db:"condition_id"`
	Weight      float64   `json:"weight" db:"weight"`
	Condition   Condition `json:"condition"`
}

// DBQueryer defines the interface for database operations.
type DBQueryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// LoadScoringStrategy fetches a strategy and all its associated rules/conditions by slug.
func LoadScoringStrategy(ctx context.Context, db DBQueryer, slug string) (*ScoringStrategy, error) {
	// 1. Fetch the base Strategy
	s := &ScoringStrategy{}
	strategyQuery := `
		SELECT id, name, slug, threshold, created_at, updated_at
		FROM strategies
		WHERE slug = $1
	`
	err := db.QueryRowContext(ctx, strategyQuery, slug).Scan(
		&s.ID, &s.Name, &s.Slug, &s.Threshold, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("strategy not found with slug: %s", slug)
		}
		return nil, fmt.Errorf("failed to fetch strategy: %w", err)
	}

	// 2. Fetch Rules and Conditions via JOIN
	rulesQuery := `
		SELECT 
			sr.strategy_id, sr.weight,
			c.id, c.name, c.type, c.params
		FROM strategy_rules sr
		JOIN conditions c ON sr.condition_id = c.id
		WHERE sr.strategy_id = $1
	`

	rows, err := db.QueryContext(ctx, rulesQuery, s.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch strategy rules: %w", err)
	}
	defer rows.Close()

	var rules []StrategyRule
	for rows.Next() {
		var r StrategyRule
		var c Condition
		var paramsBytes []byte

		err := rows.Scan(
			&r.StrategyID, &r.Weight,
			&c.ID, &c.Name, &c.Type, &paramsBytes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule row: %w", err)
		}

		c.ParamsRaw = json.RawMessage(paramsBytes)
		r.ConditionID = c.ID
		r.Condition = c

		rules = append(rules, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	s.Rules = rules
	return s, nil
}
