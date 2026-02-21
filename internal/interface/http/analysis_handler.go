package httpapi

import (
	"log"
	"net/http"
	"time"

	"ai-auto-trade/internal/application/analysis"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleAnalysisDaily(c *gin.Context) {
	var body struct {
		TradeDate string `json:"trade_date"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	tradeDate, err := time.Parse("2006-01-02", body.TradeDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid trade_date", "error_code": errCodeBadRequest})
		return
	}

	summary, err := s.runAnalysisForDate(c.Request.Context(), tradeDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"summary": gin.H{
			"total":   summary.total,
			"success": summary.success,
			"failure": summary.failure,
		},
	})
}

func (s *Server) handleAnalysisQuery(c *gin.Context) {
	tradeDateStr := c.Query("trade_date")
	if tradeDateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "trade_date required", "error_code": errCodeBadRequest})
		return
	}
	tradeDate, err := time.Parse("2006-01-02", tradeDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid trade_date", "error_code": errCodeBadRequest})
		return
	}

	timeframe := s.getTimeframe(c, "1d")
	limit := parseIntDefault(c.Query("limit"), 100)
	offset := parseIntDefault(c.Query("offset"), 0)

	out, err := s.queryUC.QueryByDate(c.Request.Context(), analysis.QueryByDateInput{
		Date: tradeDate,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
			Timeframe:   timeframe,
		},
		Sort: analysis.SortOption{
			Field: analysis.SortScore,
			Desc:  true,
		},
		Pagination: analysis.Pagination{
			Offset: offset,
			Limit:  limit,
		},
	})
	if err != nil {
		log.Printf("[Analysis] Query failed for %s: %v", tradeDateStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "query failed", "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"trade_date":  tradeDateStr,
		"total_count": out.Total,
		"items":       out.Results,
	})
}

func (s *Server) handleAnalysisHistory(c *gin.Context) {
	symbol := s.getSymbol(c)
	start, end, err := s.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}

	timeframe := s.getTimeframe(c, "1d")
	limit := parseIntDefault(c.Query("limit"), 100)

	results, err := s.queryUC.QueryHistory(c.Request.Context(), analysis.QueryHistoryInput{
		Symbol:      symbol,
		From:        &start,
		To:          &end,
		Limit:       limit,
		OnlySuccess: true,
		Timeframe:   timeframe,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"symbol":  symbol,
		"items":   results,
	})
}

func (s *Server) handleAnalysisSummary(c *gin.Context) {
	latestDate, err := s.dataRepo.LatestAnalysisDate(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "no data", "error_code": errCodeAnalysisNotReady})
		return
	}

	out, err := s.queryUC.QueryByDate(c.Request.Context(), analysis.QueryByDateInput{
		Date: latestDate,
		Filter: analysis.QueryFilter{
			OnlySuccess: true,
		},
		Sort: analysis.SortOption{
			Field: analysis.SortScore,
			Desc:  true,
		},
		Pagination: analysis.Pagination{
			Limit: 3,
		},
	})
	if err != nil || len(out.Results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "no data", "error_code": errCodeAnalysisNotReady})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"trade_date": latestDate.Format("2006-01-02"),
		"top_picks":  out.Results,
	})
}
