package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/inbound"
	"github.com/b1ll1eee/flowdo-api/pkg/response"
)

const userIDKey = "user_id"

// Auth returns a Gin middleware that validates Bearer tokens using AuthService.
func Auth(authSvc inbound.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Unauthorized(c, "authorization header must be Bearer <token>")
			c.Abort()
			return
		}

		userID, err := authSvc.ValidateToken(c.Request.Context(), parts[1])
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(userIDKey, userID)
		c.Next()
	}
}

// GetUserID retrieves the authenticated user's ID from the Gin context.
// It panics if the middleware was not applied, which indicates a programming error.
func GetUserID(c *gin.Context) (interface{}, bool) {
	return c.Get(userIDKey)
}
