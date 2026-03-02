package strategy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	tradingDomain "ai-auto-trade/internal/domain/trading"

	"gorm.io/gorm"
)

// LoadScoringStrategyBySlugGORM fetches a strategy and all its associated rules/conditions by slug using GORM.
func LoadScoringStrategyBySlugGORM(ctx context.Context, db *gorm.DB, slug string) (*ScoringStrategy, error) {
	return loadScoringStrategyGORM(ctx, db, "slug", slug)
}

// LoadScoringStrategyIDGORM fetches a strategy and all its associated rules/conditions by ID using GORM.
func LoadScoringStrategyIDGORM(ctx context.Context, db *gorm.DB, id string) (*ScoringStrategy, error) {
	return loadScoringStrategyGORM(ctx, db, "id", id)
}

func loadScoringStrategyGORM(ctx context.Context, db *gorm.DB, field, value string) (*ScoringStrategy, error) {
	// 1. Fetch the base Strategy
	s := &ScoringStrategy{}
	type strategyResult struct {
		ID            string
		UserID        string
		Name          string
		Slug          string
		Description   string
		Timeframe     string
		BaseSymbol    string
		Threshold     float64
		ExitThreshold float64
		IsActive      bool
		Env           string
		RiskSettings  []byte
		CreatedAt     time.Time
		UpdatedAt     time.Time
	}

	var res strategyResult
	err := db.WithContext(ctx).Table("strategies").Where(fmt.Sprintf("%s = ?", field), value).First(&res).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("strategy not found with %s: %s", field, value)
		}
		return nil, fmt.Errorf("failed to fetch strategy: %w", err)
	}

	s.ID = res.ID
	s.UserID = res.UserID
	s.Name = res.Name
	s.Slug = res.Slug
	s.Description = res.Description
	s.Timeframe = res.Timeframe
	s.BaseSymbol = res.BaseSymbol
	s.Threshold = res.Threshold
	s.ExitThreshold = res.ExitThreshold
	s.IsActive = res.IsActive
	s.Env = res.Env
	s.CreatedAt = res.CreatedAt
	s.UpdatedAt = res.UpdatedAt

	if len(res.RiskSettings) > 0 {
		_ = json.Unmarshal(res.RiskSettings, &s.Risk)
	}

	// 2. Fetch Rules and Conditions via JOIN
	type ruleResult struct {
		StrategyID  string
		Weight      float64
		RuleType    string
		ID          string
		Name        string
		Type        string
		Params      []byte
	}

	var rawResults []ruleResult
	err = db.WithContext(ctx).Table("strategy_rules sr").
		Select("sr.strategy_id, sr.weight, sr.rule_type, c.id, c.name, c.type, c.params").
		Joins("JOIN conditions c ON sr.condition_id = c.id").
		Where("sr.strategy_id = ?", s.ID).
		Scan(&rawResults).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch strategy rules: %w", err)
	}

	for _, r := range rawResults {
		rule := StrategyRule{
			StrategyID:  r.StrategyID,
			ConditionID: r.ID,
			Weight:      r.Weight,
			RuleType:    r.RuleType,
			Condition: Condition{
				ID:        r.ID,
				Name:      r.Name,
				Type:      r.Type,
				ParamsRaw: r.Params,
			},
		}

		s.Rules = append(s.Rules, rule)

		isEntry := rule.RuleType == "entry" || rule.RuleType == "both" || rule.RuleType == ""
		isExit := rule.RuleType == "exit" || rule.RuleType == "both"

		if isEntry {
			s.EntryRules = append(s.EntryRules, rule)
		}
		if isExit {
			s.ExitRules = append(s.ExitRules, rule)
		}
	}

	return s, nil
}

// ScoringStrategy represents the new three-layer strategy design.
type ScoringStrategy struct {
	ID        string         `json:"id" db:"id"`
	UserID    string         `json:"user_id" db:"user_id"`
	Name      string         `json:"name" db:"name"`
	Slug      string         `json:"slug" db:"slug"`
	Description string       `json:"description" db:"description"`
	Timeframe string         `json:"timeframe" db:"timeframe"`
	BaseSymbol string        `json:"base_symbol" db:"base_symbol"`
	Threshold     float64        `json:"threshold" db:"threshold"`
	ExitThreshold float64        `json:"exit_threshold" db:"exit_threshold"`
	IsActive      bool           `json:"is_active" db:"is_active"`
	Env           string         `json:"env" gorm:"column:env"`
	Risk          tradingDomain.RiskSettings `json:"risk_settings" gorm:"-"`
	Rules         []StrategyRule `json:"rules" gorm:"-"` 
	EntryRules    []StrategyRule `json:"entry_rules" gorm:"-"`
	ExitRules     []StrategyRule `json:"exit_rules" gorm:"-"`
	CreatedAt     time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"column:updated_at"`
}

// Condition represents a reusable logic component.
type Condition struct {
	ID        string          `json:"id" db:"id"`
	Name      string          `json:"name" db:"name"`
	Type      string          `json:"type" db:"type"`
	ParamsRaw json.RawMessage `json:"params" db:"params"` 
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
	RuleType    string    `json:"rule_type" db:"rule_type"` // 'entry' or 'exit'
	Condition   Condition `json:"condition"`
}

// LoadScoringStrategyBySlug fetches a strategy and all its associated rules/conditions by slug using sql.DB (Legacy).
func LoadScoringStrategyBySlug(ctx context.Context, db *sql.DB, slug string) (*ScoringStrategy, error) {
	return loadScoringStrategyLegacy(ctx, db, "slug", slug)
}

// LoadScoringStrategyByID fetches a strategy and all its associated rules/conditions by ID using sql.DB (Legacy).
func LoadScoringStrategyByID(ctx context.Context, db *sql.DB, id string) (*ScoringStrategy, error) {
	return loadScoringStrategyLegacy(ctx, db, "id", id)
}

func loadScoringStrategyLegacy(ctx context.Context, db *sql.DB, field, value string) (*ScoringStrategy, error) {
	// 1. Fetch the base Strategy
	s := &ScoringStrategy{}
	strategyQuery := fmt.Sprintf(`
		SELECT id, user_id, name, slug, description, timeframe, base_symbol, threshold, exit_threshold, is_active, env, risk_settings, created_at, updated_at
		FROM strategies
		WHERE %s = $1
	`, field)
	var riskRaw []byte
	var slugNull, descNull sql.NullString
	err := db.QueryRowContext(ctx, strategyQuery, value).Scan(
		&s.ID, &s.UserID, &s.Name, &slugNull, &descNull, &s.Timeframe, &s.BaseSymbol, &s.Threshold, &s.ExitThreshold, &s.IsActive, &s.Env, &riskRaw, &s.CreatedAt, &s.UpdatedAt,
	)
	s.Slug = slugNull.String
	s.Description = descNull.String

	if err == nil && len(riskRaw) > 0 {
		_ = json.Unmarshal(riskRaw, &s.Risk)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("strategy not found with %s: %s", field, value)
		}
		return nil, fmt.Errorf("failed to fetch strategy: %w", err)
	}

	// 2. Fetch Rules and Conditions via JOIN
	rulesQuery := `
		SELECT 
			sr.strategy_id, sr.weight, sr.rule_type,
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
			&r.StrategyID, &r.Weight, &r.RuleType,
			&c.ID, &c.Name, &c.Type, &paramsBytes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule row: %w", err)
		}

		c.ParamsRaw = json.RawMessage(paramsBytes)
		r.ConditionID = c.ID
		r.Condition = c

		rules = append(rules, r)
		
		isEntry := r.RuleType == "entry" || r.RuleType == "both" || r.RuleType == ""
		isExit := r.RuleType == "exit" || r.RuleType == "both"

		if isEntry {
			s.EntryRules = append(s.EntryRules, r)
		}
		if isExit {
			s.ExitRules = append(s.ExitRules, r)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	s.Rules = rules
	return s, nil
}
