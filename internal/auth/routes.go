package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, handler *Handler) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", handler.Register)
		authGroup.POST("/login", handler.Login)
		authGroup.POST("/refresh", handler.Refresh)
		authGroup.POST("/logout", handler.Logout)
	}
}
