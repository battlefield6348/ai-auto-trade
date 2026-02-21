package httpapi

import (
	"log"
	"net/http"
	"time"

	"ai-auto-trade/internal/application/auth"

	"github.com/gin-gonic/gin"
)

func (s *Server) requireAuth(perm auth.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := parseBearer(c.GetHeader("Authorization"))
		if token == "" {
			if t, err := c.Cookie("access_token"); err == nil {
				token = t
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized", "error_code": errCodeUnauthorized})
			c.Abort()
			return
		}

		claims, err := s.tokenSvc.ParseAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid token", "error_code": errCodeUnauthorized})
			c.Abort()
			return
		}

		// Check permission if needed
		if perm != "" {
			res, err := s.authz.Authorize(c.Request.Context(), auth.AuthorizeInput{
				UserID:   claims.UserID,
				Required: []auth.Permission{perm},
			})
			if err != nil || !res.Allowed {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "forbidden", "error_code": errCodeForbidden})
				c.Abort()
				return
			}
		}

		c.Set("userID", claims.UserID)
		c.Next()
	}
}

func (s *Server) ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[GIN] %v | %3d | %13v | %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			status,
			latency,
			c.Request.Method,
			path,
		)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
