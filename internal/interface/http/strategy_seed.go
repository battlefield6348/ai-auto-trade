package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// seedScoringStrategies 預設建立幾套最強的計分策略。
func seedScoringStrategies(ctx context.Context, db *gorm.DB) error {
	// 1. 取得管理員 ID
	var adminID string
	err := db.Table("users").Select("id").Where("email = ?", "admin@example.com").Scan(&adminID).Error
	if err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}
	if adminID == "" {
		return fmt.Errorf("admin user not found")
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
			ExitThreshold: 90.0,
			IsActive:      true,
			Rules: []struct {
				Type     string
				Name     string
				Params   map[string]interface{}
				Weight   float64
				RuleType string
			}{
				{Type: "BASE_SCORE", Name: "AI 核心預測支撐", Params: map[string]interface{}{}, Weight: 45.0, RuleType: "entry"},
				{Type: "PRICE_RETURN", Name: "短線發動 (> 1.2%)", Params: map[string]interface{}{"days": 1.0, "min": 0.012}, Weight: 30.0, RuleType: "entry"},
				{Type: "VOLUME_SURGE", Name: "量能確認 (> 1.5倍)", Params: map[string]interface{}{"min": 1.5}, Weight: 15.0, RuleType: "entry"},
				{Type: "MA_DEVIATION", Name: "均線上方安全區 (MA20 > 1%)", Params: map[string]interface{}{"ma": 20.0, "min": 0.01}, Weight: 10.0, RuleType: "entry"},
				
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
			ExitThreshold: 85.0,
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
		err = db.Table("strategies").Select("id").Where("slug = ?", s.Slug).Scan(&sid).Error
		if err == nil && sid != "" {
			log.Printf("[Seed] Strategy %s already exists, skipping re-seed.", s.Slug)
			continue
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			newStrat := struct {
				ID            string `gorm:"primaryKey;default:gen_random_uuid()"`
				UserID        string
				Name          string
				Slug          string
				Timeframe     string
				BaseSymbol    string
				Threshold     float64
				ExitThreshold float64
				IsActive      bool
				Env           string
			}{
				UserID:        adminID,
				Name:          s.Name,
				Slug:          s.Slug,
				Timeframe:     s.Timeframe,
				BaseSymbol:    s.BaseSymbol,
				Threshold:     s.Threshold,
				ExitThreshold: s.ExitThreshold,
				IsActive:      s.IsActive,
				Env:           "both",
			}
			if err := tx.Table("strategies").Create(&newStrat).Error; err != nil {
				return err
			}
			sid = newStrat.ID

			for _, r := range s.Rules {
				paramsBytes, _ := json.Marshal(r.Params)
				var cid string
				tx.Table("conditions").Select("id").Where("name = ? AND type = ? AND params::jsonb = ?::jsonb", r.Name, r.Type, string(paramsBytes)).Scan(&cid)

				if cid == "" {
					newCond := struct {
						ID     string `gorm:"primaryKey;default:gen_random_uuid()"`
						Name   string
						Type   string
						Params []byte
					}{
						Name:   r.Name,
						Type:   r.Type,
						Params: paramsBytes,
					}
					if err := tx.Table("conditions").Create(&newCond).Error; err != nil {
						return err
					}
					cid = newCond.ID
				}

				link := struct {
					StrategyID  string
					ConditionID string
					Weight      float64
					RuleType    string
				}{
					StrategyID:  sid,
					ConditionID: cid,
					Weight:      r.Weight,
					RuleType:    r.RuleType,
				}
				if err := tx.Table("strategy_rules").Create(&link).Error; err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			log.Printf("[Seed] Strategy %s seed failed: %v", s.Slug, err)
		}
	}

	log.Printf("[Seed] Default scoring strategies checked.")
	return nil
}
