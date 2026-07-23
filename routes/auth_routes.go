package routes

import (
	"time"

	"backend/controllers"
	"backend/middleware"
	"backend/services"

	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(r *gin.Engine, authCtrl *controllers.AuthController, rateLimiter services.RateLimitService, jwtCfg interface{ GetSecret() string }) {
	// We can't use the config.JWTConfig interface directly, so we'll accept it differently.
	// This function is called from main.go where we pass what we need.
}

// SetupAuthRoutes registers all authentication routes with appropriate middleware.
func SetupAuthRoutes(
	r *gin.Engine,
	authCtrl *controllers.AuthController,
	rateLimiter services.RateLimitService,
	authMiddleware gin.HandlerFunc,
) {
	auth := r.Group("/api/auth")
	{
		// Public endpoints with rate limiting
		auth.POST("/register",
			middleware.RateLimitMiddleware(rateLimiter, "register", 5, 15*time.Minute),
			authCtrl.Register,
		)
		auth.POST("/login",
			middleware.RateLimitMiddleware(rateLimiter, "login", 10, 15*time.Minute),
			authCtrl.Login,
		)
		auth.POST("/verify-email",
			middleware.RateLimitMiddleware(rateLimiter, "verify-email", 10, 15*time.Minute),
			authCtrl.VerifyEmail,
		)
		auth.POST("/resend-verification",
			middleware.RateLimitMiddleware(rateLimiter, "resend-verification", 3, 15*time.Minute),
			authCtrl.ResendVerification,
		)
		auth.POST("/forgot-password",
			middleware.RateLimitMiddleware(rateLimiter, "forgot-password", 3, 15*time.Minute),
			authCtrl.ForgotPassword,
		)
		auth.POST("/reset-password",
			middleware.RateLimitMiddleware(rateLimiter, "reset-password", 5, 15*time.Minute),
			authCtrl.ResetPassword,
		)

		// Refresh — reads from cookie, no JWT auth needed
		auth.POST("/refresh",
			middleware.RateLimitMiddleware(rateLimiter, "refresh", 30, 15*time.Minute),
			authCtrl.Refresh,
		)

		// Protected endpoints — require valid JWT
		protected := auth.Group("")
		protected.Use(authMiddleware)
		{
			protected.GET("/me", authCtrl.Me)
			protected.PUT("/me", authCtrl.UpdateProfile)
			protected.PUT("/me/password", authCtrl.ChangePassword)
			protected.POST("/me/avatar", authCtrl.UploadAvatar)
			protected.PUT("/me/notifications", authCtrl.UpdateNotificationPreferences)
			protected.POST("/logout", authCtrl.Logout)
			protected.POST("/logout-all", authCtrl.LogoutAll)
			protected.GET("/sessions", authCtrl.GetSessions)
			protected.DELETE("/sessions/:id", authCtrl.RevokeSession)
		}
	}
}
