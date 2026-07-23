package middleware

import (
	"net/http"
	"time"

	"backend/dto"
	"backend/services"

	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware creates a rate limiting middleware for a specific endpoint category.
// key is derived from the client IP address + the action name.
func RateLimitMiddleware(rateLimiter services.RateLimitService, action string, maxAttempts int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := action + ":" + ip

		allowed, err := rateLimiter.Allow(c.Request.Context(), key, maxAttempts, window)
		if err != nil {
			// If Redis is down, allow the request (fail open for availability)
			c.Next()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, dto.ErrorResponse{
				Success: false,
				Message: "Too many requests. Please try again later.",
				Error:   dto.ErrorDetail{Code: "RATE_LIMITED"},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
