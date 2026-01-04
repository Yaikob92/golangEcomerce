package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/yaikob92/ecommerce/controllers"
)

func SetupRoutes(incomingRouter *gin.Engine) {
	incomingRouter.POST("/users/singnup", controllers.Singnup())
	incomingRouter.POST("/user/login", controllers.Login())
	incomingRouter.POST("/admin/addproduct", controllers.AddProduct())
	incomingRouter.GET("/user/productView", controllers.SearchProduct())
	incomingRouter.GET("/user/search", controllers.SearchProductByQuery())
}
