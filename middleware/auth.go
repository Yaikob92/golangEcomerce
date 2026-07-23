package middleware

import (
	"net/http"
	"strings"

	"backend/config"
	"backend/dto"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware validates the access token from the HttpOnly cookie
// and injects user claims into the Gin context.
func JWTAuthMiddleware(cfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// 1. Try to extract from HttpOnly cookie first
		cookie, err := c.Cookie("access_token")
		if err == nil && cookie != "" {
			tokenStr = cookie
		}

		// 2. Fallback to Authorization header (for API clients/testing)
		if tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
					tokenStr = parts[1]
				}
			}
		}

		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Message: "Authentication required",
				Error:   dto.ErrorDetail{Code: "UNAUTHORIZED"},
			})
			c.Abort()
			return
		}

		claims, err := utils.ValidateAccessToken(tokenStr, cfg.Secret, cfg.Issuer, cfg.Audience)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Message: "Invalid or expired token",
				Error:   dto.ErrorDetail{Code: "INVALID_TOKEN"},
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("userID", claims.UserID)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)

		c.Next()
	}
}
