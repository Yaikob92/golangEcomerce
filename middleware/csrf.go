package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"backend/dto"

	"github.com/gin-gonic/gin"
)

// csrfExcludedPaths lists paths that should skip CSRF validation.
// These are public endpoints where CSRF protection is not needed.
var csrfExcludedPaths = []string{
	"/api/auth/register",
	"/api/auth/login",
	"/api/auth/verify-email",
	"/api/auth/resend-verification",
	"/api/auth/forgot-password",
	"/api/auth/reset-password",
	"/api/auth/refresh",
	"/ping",
	"/swagger",
}

// CSRFMiddleware implements the Double-Submit Cookie pattern.
// On state-changing requests (POST, PUT, PATCH, DELETE), it validates
// that the X-CSRF-Token header matches the csrf_token cookie.
// Public routes listed in csrfExcludedPaths are skipped.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip CSRF for excluded public paths
		for _, excluded := range csrfExcludedPaths {
			if strings.HasPrefix(path, excluded) {
				c.Next()
				return
			}
		}

		// Only apply to state-changing methods
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			// Ensure a CSRF cookie exists for the client to read
			ensureCSRFCookie(c)
			c.Next()
			return
		}

		// Validate CSRF token
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil || cookieToken == "" {
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Message: "CSRF token missing",
				Error:   dto.ErrorDetail{Code: "CSRF_MISSING"},
			})
			c.Abort()
			return
		}

		headerToken := c.GetHeader("X-CSRF-Token")
		if headerToken == "" || headerToken != cookieToken {
			c.JSON(http.StatusForbidden, dto.ErrorResponse{
				Success: false,
				Message: "CSRF token invalid",
				Error:   dto.ErrorDetail{Code: "CSRF_INVALID"},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func ensureCSRFCookie(c *gin.Context) {
	_, err := c.Cookie("csrf_token")
	if err != nil {
		// Generate a new CSRF token
		bytes := make([]byte, 32)
		rand.Read(bytes)
		token := hex.EncodeToString(bytes)

		isProduction := c.GetHeader("X-Forwarded-Proto") == "https"
		c.SetCookie("csrf_token", token, 86400, "/", "", isProduction, false) // Not HttpOnly — JS must read it
	}
}
