package httpapi

import (
	"log"
	"net/http"
	"time"

	"ai-auto-trade/internal/application/auth"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleLogin(c *gin.Context) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid body", "error_code": errCodeBadRequest})
		return
	}

	res, err := s.loginUC.Execute(c.Request.Context(), auth.LoginInput{
		Email:     body.Email,
		Password:  body.Password,
		UserAgent: c.GetHeader("User-Agent"),
		IP:        c.ClientIP(),
	})
	if err != nil {
		log.Printf("[Auth] login failure for %s: %v", body.Email, err)
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid email or password", "error_code": errCodeInvalidCredentials})
		return
	}

	s.setRefreshCookie(c, res.Token.RefreshToken, res.Token.RefreshExpiry)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":    res.User.ID,
			"email": res.User.Email,
			"role":  res.User.Role,
		},
		"access_token": res.Token.AccessToken,
		"token_type":   "Bearer",
		"expiry":       res.Token.AccessExpiry.Format(time.RFC3339),
	})
}

func (s *Server) handleRefresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "refresh token missing", "error_code": errCodeUnauthorized})
		return
	}

	res, err := s.tokenSvc.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid refresh token", "error_code": errCodeUnauthorized})
		return
	}

	s.setRefreshCookie(c, res.RefreshToken, res.RefreshExpiry)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"access_token": res.AccessToken,
		"token_type":   "Bearer",
		"expiry":       res.AccessExpiry.Format(time.RFC3339),
	})
}

func (s *Server) handleLogout(c *gin.Context) {
	refreshToken, _ := c.Cookie(refreshCookieName)
	if refreshToken != "" {
		_ = s.logoutUC.Execute(c.Request.Context(), refreshToken)
	}

	c.SetCookie(
		refreshCookieName,
		"",
		-1,
		"/",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"success": true})
}
