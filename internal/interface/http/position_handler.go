package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleListPositions(c *gin.Context) {
	positions, err := s.tradingSvc.ListPositions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"positions": positions,
	})
}

func (s *Server) handlePositionClose(c *gin.Context, id string) {
	if err := s.tradingSvc.ClosePositionManually(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}
