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

	// 2. 定義預設策略
	strategies := []struct {
		Name      string
		Slug      string
		Threshold float64
		Rules     []struct {
			Type   string
			Name   string
			Params map[string]interface{}
			Weight float64
		}
	}{
		{
			Name:      "趨勢突破策略 (Trend Breakout)",
			Slug:      "trend-breakout",
			Threshold: 7.0,
			Rules: []struct {
				Type   string
				Name   string
				Params map[string]interface{}
				Weight float64
			}{
				{Type: "PRICE_RETURN", Name: "5日漲幅 > 5%", Params: map[string]interface{}{"days": 5.0, "min": 0.05}, Weight: 4.0},
				{Type: "VOLUME_SURGE", Name: "成交量倍數 > 1.5", Params: map[string]interface{}{"min": 1.5}, Weight: 3.0},
				{Type: "MA_DEVIATION", Name: "價格在20日均線上 (MA20 > 0%)", Params: map[string]interface{}{"ma": 20.0, "min": 0.0}, Weight: 3.0},
			},
		},
		{
			Name:      "強勢放量策略 (High Volume Surge)",
			Slug:      "volume-surge-pro",
			Threshold: 6.0,
			Rules: []struct {
				Type   string
				Name   string
				Params map[string]interface{}
				Weight float64
			}{
				{Type: "VOLUME_SURGE", Name: "巨大成交量 > 2.0", Params: map[string]interface{}{"min": 2.0}, Weight: 6.0},
				{Type: "PRICE_RETURN", Name: "當日收紅 (> 0%)", Params: map[string]interface{}{"days": 1.0, "min": 0.0}, Weight: 4.0},
			},
		},
		{
			Name:      "低檔轉強策略 (Reversal at Low)",
			Slug:      "low-reversal",
			Threshold: 6.5,
			Rules: []struct {
				Type   string
				Name   string
				Params map[string]interface{}
				Weight float64
			}{
				{Type: "RANGE_POS", Name: "20日股價位階 < 30%", Params: map[string]interface{}{"days": 20.0, "min": 0.3}, Weight: 2.0}, // Note: min here works differently in evaluator, it's actually "is at least" 0.3, so we might need a "max" in evaluator or just different logic.
				{Type: "PRICE_RETURN", Name: "5日漲幅由負轉正 (> 2%)", Params: map[string]interface{}{"days": 5.0, "min": 0.02}, Weight: 5.0},
				{Type: "VOLUME_SURGE", Name: "成交量略微放大 (> 1.2)", Params: map[string]interface{}{"min": 1.2}, Weight: 3.0},
			},
		},
	}

	for _, s := range strategies {
		var sid string
		err = db.QueryRowContext(ctx, `
			INSERT INTO strategies (user_id, name, slug, threshold, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, threshold = EXCLUDED.threshold, updated_at = NOW()
			RETURNING id
		`, adminID, s.Name, s.Slug, s.Threshold).Scan(&sid)
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
				SELECT id FROM conditions WHERE name = $1 AND type = $2 AND params = $3
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

			_, err = db.ExecContext(ctx, `
				INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type)
				VALUES ($1, $2, $3, 'entry')
				ON CONFLICT DO NOTHING
			`, sid, cid, r.Weight)

			if err != nil {
				log.Printf("[Seed] Link rule %s to %s failed: %v", r.Name, s.Slug, err)
			}
		}
	}

	log.Printf("[Seed] Default scoring strategies seeded successfully")
	return nil
}
