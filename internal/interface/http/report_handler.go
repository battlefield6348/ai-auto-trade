package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type reportRequest struct {
	Env         string      `json:"env"`
	PeriodStart string      `json:"period_start"`
	PeriodEnd   string      `json:"period_end"`
	Summary     interface{} `json:"summary"`
	TradesRef   interface{} `json:"trades_ref"`
}

func (s *Server) handleCreateReport(c *gin.Context, strategyID string) {
	var body reportRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	// MVP: Skip saving report to specific table for now if not needed,
	// or just acknowledge. In MVP, we might just be logging or echoing.
	// But let's assume tradingSvc has a SaveReport or similar if we want persistence.
	
	// if err := s.tradingSvc.SaveReport(c.Request.Context(), strategyID, ...); err != nil { ... }

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Report created (mock persistence)",
	})
}

func (s *Server) handleListReports(c *gin.Context, strategyID string) {
	// MVP: Echo empty or list from DB if implemented
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"reports": []interface{}{},
	})
}
