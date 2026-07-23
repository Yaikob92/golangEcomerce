package middleware

import (
	"net/http"

	"backend/dto"

	"github.com/gin-gonic/gin"
)

// RequireRole returns middleware that restricts access to users with one of the specified roles.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Message: "Authentication required",
				Error:   dto.ErrorDetail{Code: "UNAUTHORIZED"},
			})
			c.Abort()
			return
		}

		userRole := role.(string)
		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Success: false,
			Message: "You do not have permission to access this resource",
			Error:   dto.ErrorDetail{Code: "FORBIDDEN"},
		})
		c.Abort()
	}
}

// AdminOnly restricts access to admin users.
func AdminOnly() gin.HandlerFunc {
	return RequireRole("admin")
}

// MerchantOnly restricts access to merchant users.
func MerchantOnly() gin.HandlerFunc {
	return RequireRole("merchant")
}

// CustomerOnly restricts access to customer users.
func CustomerOnly() gin.HandlerFunc {
	return RequireRole("customer")
}

// SuperAdminOnly restricts access to super_admin users.
func SuperAdminOnly() gin.HandlerFunc {
	return RequireRole("super_admin")
}

// SuperAdminOrAdmin restricts access to super_admin and admin users.
func SuperAdminOrAdmin() gin.HandlerFunc {
	return RequireRole("super_admin", "admin")
}
