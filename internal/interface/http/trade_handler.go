package httpapi

import (
	"log"
	"net/http"

	"ai-auto-trade/internal/domain/trading"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleListTrades(c *gin.Context) {
	trades, err := s.tradingSvc.ListTrades(c.Request.Context(), trading.TradeFilter{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"trades":  trades,
	})
}

func (s *Server) handleManualBuy(c *gin.Context) {
	var body struct {
		Symbol    string  `json:"symbol"`
		Amount    float64 `json:"amount"` // USDT amount
		Env       string  `json:"env"`
		Strategy  string  `json:"strategy_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	env := s.defaultEnv
	if body.Env != "" {
		env = trading.Environment(body.Env)
	}

	log.Printf("[ManualBuy] Triggering buy for %s amount=%f env=%s", body.Symbol, body.Amount, env)
	
	err := s.tradingSvc.ExecuteManualBuy(c.Request.Context(), body.Symbol, body.Amount, env, currentUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Buy order placed",
	})
}
