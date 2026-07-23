package routes

import (
	"backend/controllers"
	"backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupSuperAdminRoutes(
	r *gin.Engine,
	superAdminCtrl *controllers.SuperAdminController,
	authMiddleware gin.HandlerFunc,
) {
	sa := r.Group("/api/v1/super-admin")
	sa.Use(authMiddleware)
	sa.Use(middleware.SuperAdminOnly())
	{
		// Dashboard
		sa.GET("/dashboard", superAdminCtrl.GetDashboard)

		// Admin Management
		sa.GET("/admins", superAdminCtrl.ListAdmins)
		sa.POST("/admins", superAdminCtrl.CreateAdmin)
		sa.GET("/admins/:id", superAdminCtrl.GetAdmin)
		sa.PUT("/admins/:id", superAdminCtrl.UpdateAdmin)
		sa.PATCH("/admins/:id/status", superAdminCtrl.UpdateAdminStatus)
		sa.DELETE("/admins/:id", superAdminCtrl.DeleteAdmin)
	}
}
