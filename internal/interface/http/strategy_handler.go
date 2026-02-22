package httpapi

import (
	"database/sql"
	"fmt"
	"net/http"

	"ai-auto-trade/internal/application/strategy"
	"ai-auto-trade/internal/application/trading"
	tradingDomain "ai-auto-trade/internal/domain/trading"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleListStrategies(c *gin.Context) {
	strategies, err := s.tradingSvc.ListStrategies(c.Request.Context(), trading.StrategyFilter{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"strategies": strategies,
	})
}

func (s *Server) handleCreateStrategy(c *gin.Context) {
	var body strategy.SaveScoringStrategyInput
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	body.UserID = currentUserID(c)
	if body.UserID == "" {
		body.UserID = "00000000-0000-0000-0000-000000000001" // admin fallback
	}

	if err := s.saveScoringBtUC.Execute(c.Request.Context(), body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

func (s *Server) handleGetStrategyByQuery(c *gin.Context) {
	id := c.Query("id")
	slug := c.Query("slug")

	var st tradingDomain.Strategy
	var err error

	if id != "" {
		st, err = s.tradingSvc.GetStrategy(c.Request.Context(), id)
	} else if slug != "" {
		st, err = s.tradingSvc.GetStrategyBySlug(c.Request.Context(), slug)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "id or slug required", "error_code": errCodeBadRequest})
		return
	}

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": fmt.Sprintf("strategy not found (id: %s, slug: %s)", id, slug), "error_code": errCodeNotFound})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "database error: " + err.Error(), "error_code": errCodeInternal})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"strategy": st,
	})
}

func (s *Server) handleStrategyGetOrUpdate(c *gin.Context, id string) {
	switch c.Request.Method {
	case http.MethodGet:
		st, err := s.tradingSvc.GetStrategy(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "strategy not found", "error_code": errCodeNotFound})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "strategy": st})
	case http.MethodPut:
		var st tradingDomain.Strategy
		if err := c.ShouldBindJSON(&st); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
			return
		}
		// id should be from path
		updated, err := s.tradingSvc.UpdateStrategy(c.Request.Context(), id, st)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "strategy": updated})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"success": false, "error": "method not allowed", "error_code": errCodeMethodNotAllowed})
	}
}

func (s *Server) handleStrategyBacktest(c *gin.Context, strategyID string) {
	var body strategyBacktestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	input, err := buildBacktestInput(body, strategyID, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}
	input.Save = true
	input.CreatedBy = currentUserID(c)
	rec, err := s.tradingSvc.Backtest(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  rec,
	})
}

func (s *Server) handleInlineBacktest(c *gin.Context) {
	var body strategyBacktestRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}
	if body.Strategy == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "strategy object missing", "error_code": errCodeBadRequest})
		return
	}
	input, err := buildBacktestInput(body, "", body.Strategy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}
	input.Save = false
	rec, err := s.tradingSvc.Backtest(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result":  rec,
	})
}

func (s *Server) handleListStrategyBacktests(c *gin.Context, strategyID string) {
	recs, err := s.tradingSvc.ListBacktests(c.Request.Context(), strategyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": recs,
	})
}

func (s *Server) handleRunStrategy(c *gin.Context, strategyID string) {
	st, err := s.tradingSvc.GetStrategy(c.Request.Context(), strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "strategy not found", "error_code": errCodeNotFound})
		return
	}
	env := s.defaultEnv
	if qenv := c.Query("env"); qenv != "" {
		env = tradingDomain.Environment(qenv)
	}
	if err := s.tradingSvc.ExecuteScoringAutoTrade(c.Request.Context(), st.Slug, env, currentUserID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleActivateStrategy(c *gin.Context, strategyID string) {
	// SetStatus in tradingSvc
	if err := s.tradingSvc.SetStatus(c.Request.Context(), strategyID, tradingDomain.StatusActive, s.defaultEnv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleDeactivateStrategy(c *gin.Context, strategyID string) {
	if err := s.tradingSvc.SetStatus(c.Request.Context(), strategyID, tradingDomain.StatusDraft, s.defaultEnv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleStrategyExecute(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "slug is required", "error_code": errCodeBadRequest})
		return
	}

	env := s.defaultEnv
	if qenv := c.Query("env"); qenv != "" {
		env = tradingDomain.Environment(qenv)
	}

	userID := currentUserID(c)
	if userID == "" {
		userID = "00000000-0000-0000-0000-000000000001"
	}

	err := s.tradingSvc.ExecuteScoringAutoTrade(c.Request.Context(), slug, env, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Strategy check executed",
	})
}

func (s *Server) handleGenerateReport(c *gin.Context, strategyID string) {
	st, err := s.tradingSvc.GetStrategy(c.Request.Context(), strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "strategy not found", "error_code": errCodeNotFound})
		return
	}

	env := s.defaultEnv
	if qenv := c.Query("env"); qenv != "" {
		env = tradingDomain.Environment(qenv)
	}

	start, end, err := s.parseDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error(), "error_code": errCodeBadRequest})
		return
	}

	// 如果沒有指定開始時間，預設從上次啟動時間開始
	if c.Query("start_date") == "" && st.LastActivatedAt != nil {
		start = *st.LastActivatedAt
	}

	rep, err := s.tradingSvc.GenerateReport(c.Request.Context(), strategyID, env, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error(), "error_code": errCodeInternal})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"report":  rep,
	})
}
