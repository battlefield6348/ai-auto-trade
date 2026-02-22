package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ai-auto-trade/internal/application/strategy"
	"ai-auto-trade/internal/application/trading"
	strategyDomain "ai-auto-trade/internal/domain/strategy"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleAnalysisBacktest(c *gin.Context) {
	var body analysisBacktestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	
	start, err := time.Parse("2006-01-02", body.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid start_date", "error_code": errCodeBadRequest})
		return
	}
	end, err := time.Parse("2006-01-02", body.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid end_date", "error_code": errCodeBadRequest})
		return
	}

	// This handler uses the scoringBtUC which works on Scaling Strategy rules
	// But it expects a 'slug'. If we want arbitrary rules, we might need a different execute method.
	// For MVP, we assume we want to backtest an existing strategy by slug.
	slug := c.Query("slug")
	var res *strategy.BacktestResult
	if slug != "" {
		res, err = s.scoringBtUC.Execute(c.Request.Context(), slug, body.Symbol, start, end)
	} else {
		// Build dynamic strategy from inline params
		strat := buildDynamicStrategy(body)
		res, err = s.scoringBtUC.ExecuteWithStrategy(c.Request.Context(), strat, body.Symbol, start, end)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res,
	})
}

func (s *Server) handleSlugBacktest(c *gin.Context) {
	var body struct {
		Slug      string `json:"slug"`
		Symbol    string `json:"symbol"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	start, _ := time.Parse("2006-01-02", body.StartDate)
	end, _ := time.Parse("2006-01-02", body.EndDate)

	res, err := s.scoringBtUC.Execute(c.Request.Context(), body.Slug, body.Symbol, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res,
	})
}

func (s *Server) handleGetBacktestPreset(c *gin.Context) {
	// MVP: Fetch preset for a user/slug if implemented
	c.JSON(http.StatusOK, gin.H{"success": true, "preset": nil})
}

func (s *Server) handleSaveBacktestPreset(c *gin.Context) {
	// MVP: Save preset
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func normalizeBacktestRequest(req analysisBacktestRequest) (trading.BacktestInput, error) {
	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return trading.BacktestInput{}, fmt.Errorf("invalid start_date")
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return trading.BacktestInput{}, fmt.Errorf("invalid end_date")
	}
	
	return trading.BacktestInput{
		// ... mapping ...
		StartDate: start,
		EndDate:   end,
	}, nil
}

func buildDynamicStrategy(req analysisBacktestRequest) *strategyDomain.ScoringStrategy {
	s := &strategyDomain.ScoringStrategy{
		Name:          "Inline Test",
		Timeframe:     req.Timeframe,
		BaseSymbol:    req.Symbol,
		Threshold:     req.Entry.TotalMin,
		ExitThreshold: req.Exit.TotalMin,
	}
	if s.Timeframe == "" {
		s.Timeframe = "1d"
	}

	s.EntryRules = buildRules(req.Entry, "entry")
	s.ExitRules = buildRules(req.Exit, "exit")

	return s
}

func buildRules(params backtestSideParams, ruleType string) []strategyDomain.StrategyRule {
	var rules []strategyDomain.StrategyRule

	// 1. Base Score
	if params.Weights.Score > 0 {
		rules = append(rules, strategyDomain.StrategyRule{
			Weight:   params.Weights.Score,
			RuleType: ruleType,
			Condition: strategyDomain.Condition{
				Type: "BASE_SCORE",
			},
		})
	}

	// 2. Price Return
	if params.Flags.UseChange {
		p, _ := json.Marshal(map[string]interface{}{"days": 1, "min": params.Thresholds.ChangeMin})
		rules = append(rules, strategyDomain.StrategyRule{
			Weight:   params.Weights.ChangeBonus,
			RuleType: ruleType,
			Condition: strategyDomain.Condition{
				Type:      "PRICE_RETURN",
				ParamsRaw: p,
			},
		})
	}

	// 3. Volume Surge
	if params.Flags.UseVolume {
		p, _ := json.Marshal(map[string]interface{}{"min": params.Thresholds.VolumeRatioMin})
		rules = append(rules, strategyDomain.StrategyRule{
			Weight:   params.Weights.VolumeBonus,
			RuleType: ruleType,
			Condition: strategyDomain.Condition{
				Type:      "VOLUME_SURGE",
				ParamsRaw: p,
			},
		})
	}

	// 4. MA Deviation
	if params.Flags.UseMa {
		p, _ := json.Marshal(map[string]interface{}{"ma": 20, "min": params.Thresholds.MaGapMin})
		rules = append(rules, strategyDomain.StrategyRule{
			Weight:   params.Weights.MaBonus,
			RuleType: ruleType,
			Condition: strategyDomain.Condition{
				Type:      "MA_DEVIATION",
				ParamsRaw: p,
			},
		})
	}

	// 5. Range Pos
	if params.Flags.UseRange {
		p, _ := json.Marshal(map[string]interface{}{"days": 20, "min": params.Thresholds.RangeMin})
		rules = append(rules, strategyDomain.StrategyRule{
			Weight:   params.Weights.RangeBonus,
			RuleType: ruleType,
			Condition: strategyDomain.Condition{
				Type:      "RANGE_POS",
				ParamsRaw: p,
			},
		})
	}

	return rules
}
