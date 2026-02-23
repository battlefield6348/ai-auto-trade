package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// seedScoringStrategies 預設建立幾套最強的計分策略。
func seedScoringStrategies(ctx context.Context, db *sql.DB) error {
	// 1. 取得管理員 ID
	var adminID string
	err := db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = 'admin@example.com'").Scan(&adminID)
	if err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	// 2. 定義預設策略 - Nexus 高效系列
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
			Name:          "Nexus Apex 最強收益版 (Nexus Apex Optimized)",
			Slug:          "nexus-apex-high-profit",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     65.0,
			ExitThreshold: 90.0, // 極速止損：只要任一健康條件不滿足即退出
			IsActive:      true,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				// 進場邏輯：AI + 動能 + 趨勢
				{Type: "BASE_SCORE", Name: "AI 核心預測支撐", Params: map[string]interface{}{}, Weight: 45.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "短線發動 (> 1.2%)", Params: map[string]interface{}{"days": 1.0, "min": 0.012}, Weight: 30.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "量能確認 (> 1.5倍)", Params: map[string]interface{}{"min": 1.5}, Weight: 15.0, RuleType: "entry"},
				{Type: "MA_DEVIATION", Name: "均線上方安全區 (MA20 > 1%)", Params: map[string]interface{}{"ma": 20.0, "min": 0.01}, Weight: 10.0, RuleType: "entry"},
				
				// 出場邏輯 (必須全部滿足才留著，任一失敗即退出)
				{Type: "PRICE_RETURN", Name: "持倉安全 (漲跌 > -1.2%)", Params: map[string]interface{}{"days": 1.0, "min": -0.012}, Weight: 60.0, RuleType: "exit"},
				{Type: "MA_DEVIATION", Name: "趨勢未破 (MA20 > -0.5%)", Params: map[string]interface{}{"ma": 20.0, "min": -0.005}, Weight: 40.0, RuleType: "exit"},
			},
		},
		{
			Name:          "Nexus Quantum v2 高頻修正版 (Quantum Scalper)",
			Slug:          "nexus-quantum-v2",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     55.0,
			ExitThreshold: 85.0, // 修正止損遲鈍問題
			IsActive:      true,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "BASE_SCORE", Name: "AI 預測得分", Params: map[string]interface{}{}, Weight: 40.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "極速脈衝 (> 0.8%)", Params: map[string]interface{}{"days": 1.0, "min": 0.008}, Weight: 30.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "溫量支撐 (> 1.2倍)", Params: map[string]interface{}{"min": 1.2}, Weight: 20.0, RuleType: "entry"},
				{Type: "MA_DEVIATION", Name: "多頭位階 (MA20 > 0%)", Params: map[string]interface{}{"ma": 20.0, "min": 0.0}, Weight: 10.0, RuleType: "entry"},
				
				{Type: "PRICE_RETURN", Name: "移動止損 (-1.0%)", Params: map[string]interface{}{"days": 1.0, "min": -0.01}, Weight: 70.0, RuleType: "exit"},
				{Type: "MA_DEVIATION", Name: "趨勢過濾 (MA20 > -1%)", Params: map[string]interface{}{"ma": 20.0, "min": -0.01}, Weight: 30.0, RuleType: "exit"},
			},
		},
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
			err = db.QueryRowContext(ctx, `
				SELECT id FROM conditions WHERE name = $1 AND type = $2 AND params::jsonb = $3::jsonb
			`, r.Name, r.Type, paramsBytes).Scan(&cid)

			if err == sql.ErrNoRows {
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
				VALUES ($1, $2, $3, $4)
				ON CONFLICT DO NOTHING
			`, sid, cid, r.Weight, r.RuleType)

			if err != nil {
				log.Printf("[Seed] Link rule %s to %s failed: %v", r.Name, s.Slug, err)
			}
		}
	}

	// 額外清理：刪除表現不佳或舊的策略，僅保留最新最強版本
	_, err = db.ExecContext(ctx, "DELETE FROM strategies WHERE slug NOT IN ('nexus-apex-high-profit', 'nexus-quantum-v2', 'ai-optimized')")
	if err != nil {
		log.Printf("[Seed] Cleanup old strategies failed: %v", err)
	}

	log.Printf("[Seed] Default scoring strategies UPDATED with high-profit focus (Nexus Apex)")
	return nil
}
