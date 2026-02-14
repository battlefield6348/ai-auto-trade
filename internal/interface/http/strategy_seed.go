package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// seedScoringStrategies 預設建立幾個常用的計分策略。
func seedScoringStrategies(ctx context.Context, db *sql.DB) error {
	// 1. 取得管理員 ID
	var adminID string
	err := db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = 'admin@example.com'").Scan(&adminID)
	if err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	// 2. 定義預設策略 - 目前僅保留 Nexus Prime 高勝率策略
	strategies := []struct {
		Name          string
		Slug          string
		Timeframe     string
		BaseSymbol    string
		Threshold     float64
		ExitThreshold float64
		IsActive      bool
		Rules         []struct {
			Type     string
			Name     string
			Params   map[string]interface{}
			Weight   float64
			RuleType string
		}
	}{
		{
			Name:          "Nexus Prime 核心動能策略 (Nexus Prime Momentum)",
			Slug:          "nexus-prime-momentum",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     70.0,
			ExitThreshold: 45.0,
			IsActive:      true,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "BASE_SCORE", Name: "AI 核心評分", Params: map[string]interface{}{}, Weight: 50.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "短線動能漲幅 > 1.5%", Params: map[string]interface{}{"days": 1.0, "min": 0.015}, Weight: 20.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "量能爆發 > 1.8倍", Params: map[string]interface{}{"min": 1.8}, Weight: 15.0, RuleType: "entry"},
				{Type: "MA_DEVIATION", Name: "趨勢延續 (MA20 > 1%)", Params: map[string]interface{}{"ma": 20.0, "min": 0.01}, Weight: 15.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "緊急止損 (-2%)", Params: map[string]interface{}{"days": 1.0, "min": -0.02}, Weight: 50.0, RuleType: "exit"},
				{Type: "MA_DEVIATION", Name: "趨勢破壞 (MA20 < 0%)", Params: map[string]interface{}{"ma": 20.0, "min": 0.0}, Weight: 50.0, RuleType: "exit"},
			},
		},
	}

	// 取得當前所有要保留的 Slugs
	var keptSlugs []string
	for _, s := range strategies {
		keptSlugs = append(keptSlugs, s.Slug)
	}

	for _, s := range strategies {
		var sid string
		err = db.QueryRowContext(ctx, `
			INSERT INTO strategies (user_id, name, slug, timeframe, base_symbol, threshold, exit_threshold, is_active, env, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'both', NOW())
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, threshold = EXCLUDED.threshold, exit_threshold = EXCLUDED.exit_threshold, timeframe = EXCLUDED.timeframe, base_symbol = EXCLUDED.base_symbol, updated_at = NOW()
			RETURNING id
		`, adminID, s.Name, s.Slug, s.Timeframe, s.BaseSymbol, s.Threshold, s.ExitThreshold, s.IsActive).Scan(&sid)

		if err != nil {
			log.Printf("[Seed] Strategy %s insert failed: %v", s.Slug, err)
			continue
		}

		// 清除舊規則以便重新載入
		_, _ = db.ExecContext(ctx, "DELETE FROM strategy_rules WHERE strategy_id = $1", sid)

		for _, r := range s.Rules {
			paramsBytes, _ := json.Marshal(r.Params)
			var cid string
			// 1. Check if condition exists
			err = db.QueryRowContext(ctx, `
				SELECT id FROM conditions WHERE name = $1 AND type = $2 AND params::jsonb = $3::jsonb
			`, r.Name, r.Type, paramsBytes).Scan(&cid)

			if err == sql.ErrNoRows {
				// 2. Insert if not exists
				err = db.QueryRowContext(ctx, `
					INSERT INTO conditions (name, type, params)
					VALUES ($1, $2, $3)
					RETURNING id
				`, r.Name, r.Type, paramsBytes).Scan(&cid)
			}

			if err != nil {
				log.Printf("[Seed] Condition %s failed: %v", r.Name, err)
				continue
			}

			ruleType := r.RuleType
			if ruleType == "" {
				ruleType = "entry"
			}
			_, err = db.ExecContext(ctx, `
				INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT DO NOTHING
			`, sid, cid, r.Weight, ruleType)

			if err != nil {
				log.Printf("[Seed] Link rule %s to %s failed: %v", r.Name, s.Slug, err)
			}
		}
	}

	// 額外清理：刪除不在 strategies 列表中的其他策略
	_, err = db.ExecContext(ctx, "DELETE FROM strategies WHERE slug NOT IN ('nexus-prime-momentum')")
	if err != nil {
		log.Printf("[Seed] Cleanup old strategies failed: %v", err)
	}

	log.Printf("[Seed] Default scoring strategies seeded successfully (Nexus Prime only)")
	return nil
}
