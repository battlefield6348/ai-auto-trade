package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleListLogs(c *gin.Context, strategyID string) {
	// MVP: Fetch recent execution logs for a strategy
	// For now, return empty as placeholder or fetch from a log store if available.
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"logs":    []interface{}{},
	})
}
