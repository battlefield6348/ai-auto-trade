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
			Name:          "趨勢突破策略 (Trend Breakout)",
			Slug:          "trend-breakout",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     7.0,
			ExitThreshold: 5.0,
			IsActive:      false,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "PRICE_RETURN", Name: "5日漲幅 > 5%", Params: map[string]interface{}{"days": 5.0, "min": 0.05}, Weight: 3.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "成交量倍數 > 1.5", Params: map[string]interface{}{"min": 1.5}, Weight: 2.0, RuleType: "entry"},
				{Type: "RANGE_POS", Name: "位階在高位 (> 80%)", Params: map[string]interface{}{"days": 20.0, "min": 0.8}, Weight: 2.0, RuleType: "entry"},
				{Type: "AMPLITUDE_SURGE", Name: "波動放大 (> 1.2)", Params: map[string]interface{}{"min": 1.2}, Weight: 2.0, RuleType: "entry"},
				{Type: "MA_DEVIATION", Name: "價格在20日均線上", Params: map[string]interface{}{"ma": 20.0, "min": 0.0}, Weight: 1.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "跌破5日低點 (止損)", Params: map[string]interface{}{"days": 1.0, "min": -0.03}, Weight: 1.0, RuleType: "exit"},
			},
		},
		{
			Name:          "強勢放量策略 (High Volume Surge)",
			Slug:          "volume-surge-pro",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     6.0,
			ExitThreshold: 4.0,
			IsActive:      false,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "VOLUME_SURGE", Name: "巨大成交量 > 2.0", Params: map[string]interface{}{"min": 2.0}, Weight: 5.0, RuleType: "entry"},
				{Type: "AMPLITUDE_SURGE", Name: "波動度倍數 > 1.5", Params: map[string]interface{}{"min": 1.5}, Weight: 3.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "當日收紅 (> 0%)", Params: map[string]interface{}{"days": 1.0, "min": 0.0}, Weight: 2.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "成交量萎縮 (< 0.8)", Params: map[string]interface{}{"min": -0.8}, Weight: 1.0, RuleType: "exit"},
			},
		},
		{
			Name:          "低檔轉強策略 (Reversal at Low)",
			Slug:          "low-reversal",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     6.5,
			ExitThreshold: 4.0,
			IsActive:      false,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "RANGE_POS", Name: "20日股價位階 < 30%", Params: map[string]interface{}{"days": 20.0, "min": 0.3}, Weight: 2.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "5日漲幅由負轉正 (> 2%)", Params: map[string]interface{}{"days": 5.0, "min": 0.02}, Weight: 5.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "成交量略微放大 (> 1.2)", Params: map[string]interface{}{"min": 1.2}, Weight: 3.0, RuleType: "entry"},
				{Type: "RANGE_POS", Name: "股價位階回升至高位 (> 80%)", Params: map[string]interface{}{"days": 20.0, "min": 0.8}, Weight: 1.0, RuleType: "exit"},
			},
		},
		{
			Name:          "AI Max-Return Strategy (PROD)",
			Slug:          "ai-max-profit-2026",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     70.0,
			ExitThreshold: 42.0,
			IsActive:      true,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "BASE_SCORE", Name: "AI Core Score", Params: map[string]interface{}{}, Weight: 100.0, RuleType: "both"},
				{Type: "AMPLITUDE_SURGE", Name: "高效動能獎勵", Params: map[string]interface{}{"min": 1.5}, Weight: 20.0, RuleType: "entry"},
				{Type: "RANGE_POS", Name: "趨勢延續獎勵", Params: map[string]interface{}{"days": 20.0, "min": 0.7}, Weight: 10.0, RuleType: "entry"},
			},
		},
		{
			Name:          "AI Opt Strategy 2026-02-08 (RSI < 40)",
			Slug:          "opt-080324-rsi40",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     80.0,
			ExitThreshold: 50.0,
			IsActive:      false,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "BASE_SCORE", Name: "AI Core Score", Params: map[string]interface{}{}, Weight: 100.0, RuleType: "entry"},
				{Type: "MA_DEVIATION", Name: "Trend Follow", Params: map[string]interface{}{"ma": 200.0, "min": 0.0}, Weight: 30.0, RuleType: "entry"},
			},
		},
		{
			Name:          "極速阿爾法動能策略 (Alpha High-Freq Momentum) v2",
			Slug:          "alpha-hf-momentum",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     55.0,
			ExitThreshold: 35.0,
			IsActive:      true,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "PRICE_RETURN", Name: "當日漲幅 > 1.5%", Params: map[string]interface{}{"days": 1.0, "min": 0.015}, Weight: 4.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "成交量倍數 > 1.5", Params: map[string]interface{}{"min": 1.5}, Weight: 3.0, RuleType: "entry"},
				{Type: "MA_DEVIATION", Name: "站在20日均線上", Params: map[string]interface{}{"ma": 20.0, "min": 0.0}, Weight: 2.0, RuleType: "entry"},
				{Type: "AMPLITUDE_SURGE", Name: "波動度 > 1.2", Params: map[string]interface{}{"min": 1.2}, Weight: 1.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "當日不跌超過 1.5% (止損)", Params: map[string]interface{}{"days": 1.0, "min": -0.015}, Weight: 6.0, RuleType: "exit"},
				{Type: "VOLUME_SURGE", Name: "量能維持 (> 0.7)", Params: map[string]interface{}{"min": 0.7}, Weight: 4.0, RuleType: "exit"},
			},
		},
		{
			Name:          "黃金彈頭策略 (Golden Bullet Scalper)",
			Slug:          "golden-bullet",
			Timeframe:     "1d",
			BaseSymbol:    "BTCUSDT",
			Threshold:     60.0,
			ExitThreshold: 30.0,
			IsActive:      true,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "PRICE_RETURN", Name: "脈衝漲幅 > 2.0%", Params: map[string]interface{}{"days": 1.0, "min": 0.02}, Weight: 30.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "強勢放量 > 1.8倍", Params: map[string]interface{}{"min": 1.8}, Weight: 30.0, RuleType: "entry"},
				{Type: "RANGE_POS", Name: "近期突破 (> 85%)", Params: map[string]interface{}{"days": 20.0, "min": 0.85}, Weight: 20.0, RuleType: "entry"},
				{Type: "MA_DEVIATION", Name: "趨勢向上 (MA20 > 1%)", Params: map[string]interface{}{"ma": 20.0, "min": 0.01}, Weight: 20.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "趨勢反轉 (跌破 -2%)", Params: map[string]interface{}{"days": 1.0, "min": -0.02}, Weight: 50.0, RuleType: "exit"},
				{Type: "MA_DEVIATION", Name: "跌破均線 (MA20)", Params: map[string]interface{}{"ma": 20.0, "min": 0.0}, Weight: 50.0, RuleType: "exit"},
			},
		},
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

	log.Printf("[Seed] Default scoring strategies seeded successfully")
	return nil
}
