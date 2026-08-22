package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/service"
	"github.com/supernand/docubot/backend/internal/util"
)

const userIDKey = "userID"

// Auth returns Gin middleware that verifies Authorization: Bearer <jwt>.
func Auth(jwtSecret string) gin.HandlerFunc {
	svc := service.NewAuthService(nil, jwtSecret)
	return AuthWithParser(svc)
}

// TokenParser parses a JWT and returns the user ID.
type TokenParser interface {
	ParseToken(tokenStr string) (int64, error)
}

// AuthWithParser is like Auth but accepts a parser (useful for tests).
func AuthWithParser(parser TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			util.AbortError(c, 401, "UNAUTHORIZED", "missing authorization header")
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			util.AbortError(c, 401, "UNAUTHORIZED", "invalid or expired token")
			return
		}
		token := strings.TrimSpace(header[len(prefix):])
		if token == "" {
			util.AbortError(c, 401, "UNAUTHORIZED", "invalid or expired token")
			return
		}

		id, err := parser.ParseToken(token)
		if err != nil {
			util.AbortError(c, 401, "UNAUTHORIZED", "invalid or expired token")
			return
		}

		c.Set(userIDKey, id)
		c.Next()
	}
}

// UserID returns the authenticated user ID from context, or 0 if missing.
func UserID(c *gin.Context) int64 {
	v, ok := c.Get(userIDKey)
	if !ok {
		return 0
	}
	id, ok := v.(int64)
	if !ok {
		return 0
	}
	return id
}
