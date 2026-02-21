package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "pong",
		"timestamp": time.Now().Unix(),
		"status":    "alive",
	})
}

func (s *Server) handleHealth(c *gin.Context) {
	// Basic health check (DB connectivity, etc.)
	dbStatus := "ok"
	if s.db != nil {
		if err := s.db.PingContext(c.Request.Context()); err != nil {
			dbStatus = "error: " + err.Error()
		}
	} else {
		dbStatus = "using_memory"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"health":  "ok",
		"db":      dbStatus,
		"time":    time.Now().Format(time.RFC3339),
	})
}
