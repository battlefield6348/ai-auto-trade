package httpapi

import (
	"fmt"
	"net/http"

	tradingDomain "ai-auto-trade/internal/domain/trading"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleBinanceAccount(c *gin.Context) {
	if s.binanceClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "binance client not initialized", "error_code": errCodeInternal})
		return
	}
	info, err := s.binanceClient.GetAccountInfo()
	if err != nil {
		// If we are in Paper mode, don't return an error even if key is invalid.
		// Return a mock balance instead.
		if s.defaultEnv == tradingDomain.EnvPaper {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"is_mock": true,
				"account": gin.H{
					"accountType": "SPOT",
					"balances": []gin.H{
						{"asset": "USDT", "free": "0.00", "locked": "0.00"},
						{"asset": "BTC", "free": "0.000000", "locked": "0.000000"},
					},
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"account": info,
	})
}

func (s *Server) handleBinancePrice(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	price, err := s.tradingSvc.GetExchangePrice(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"symbol":  symbol,
		"price":   price,
	})
}

func (s *Server) handleGetBinanceConfig(c *gin.Context) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"active_env": s.defaultEnv,
	})
}

func (s *Server) handleUpdateBinanceConfig(c *gin.Context) {
	var body struct {
		ActiveEnv string `json:"active_env"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	newEnv := tradingDomain.Environment(body.ActiveEnv)
	// Simple validation
	switch newEnv {
	case tradingDomain.EnvProd, tradingDomain.EnvPaper, tradingDomain.EnvTest:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "unsupported environment", "error_code": errCodeBadRequest})
		return
	}

	s.configMu.Lock()
	defer s.configMu.Unlock()

	if s.binanceClient != nil {
		s.binanceClient.SetBaseURL(newEnv == tradingDomain.EnvTest)
	}
	s.defaultEnv = newEnv

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("System environment switched to %s", newEnv),
	})
}
