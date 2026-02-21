package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"ai-auto-trade/internal/application/trading"

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
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "slug query param required", "error_code": errCodeBadRequest})
		return
	}

	res, err := s.scoringBtUC.Execute(c.Request.Context(), slug, body.Symbol, start, end)
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
