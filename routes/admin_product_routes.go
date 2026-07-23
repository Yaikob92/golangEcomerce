package routes

import (
	"backend/controllers"
	"backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupAdminProductRoutes(
	r *gin.Engine,
	adminDashCtrl *controllers.AdminDashboardController,
	categoryCtrl *controllers.CategoryController,
	brandCtrl *controllers.BrandController,
	productCtrl *controllers.ProductController,
	orderCtrl *controllers.OrderController,
	customerCtrl *controllers.CustomerController,
	authMiddleware gin.HandlerFunc,
) {
	admin := r.Group("/api/v1/admin")
	admin.Use(authMiddleware)
	admin.Use(middleware.SuperAdminOrAdmin())
	{
		// Dashboard
		admin.GET("/dashboard", adminDashCtrl.GetDashboard)

		// Orders
		admin.GET("/orders", orderCtrl.ListOrders)
		admin.GET("/orders/:id", orderCtrl.GetOrder)

		// Customers
		admin.GET("/customers", customerCtrl.ListCustomers)

		// Categories
		admin.POST("/categories", categoryCtrl.CreateCategory)
		admin.GET("/categories", categoryCtrl.ListCategories)
		admin.GET("/categories/:id", categoryCtrl.GetCategory)
		admin.PUT("/categories/:id", categoryCtrl.UpdateCategory)
		admin.DELETE("/categories/:id", categoryCtrl.DeleteCategory)

		// Brands
		admin.POST("/brands", brandCtrl.CreateBrand)
		admin.GET("/brands", brandCtrl.ListBrands)
		admin.GET("/brands/:id", brandCtrl.GetBrand)
		admin.PUT("/brands/:id", brandCtrl.UpdateBrand)
		admin.DELETE("/brands/:id", brandCtrl.DeleteBrand)

		// Products
		admin.POST("/products", productCtrl.CreateProduct)
		admin.GET("/products", productCtrl.ListProducts)
		admin.GET("/products/:id", productCtrl.GetProduct)
		admin.PUT("/products/:id", productCtrl.UpdateProduct)
		admin.DELETE("/products/:id", productCtrl.DeleteProduct)

		// Product Images
		admin.GET("/products/:id/images", productCtrl.GetProductImages)
		admin.POST("/products/:id/images", productCtrl.AddProductImage)
		admin.DELETE("/products/:id/images/:imageId", productCtrl.RemoveProductImage)
		admin.PATCH("/products/:id/images/primary/:imageId", productCtrl.SetPrimaryImage)
		admin.PUT("/products/:id/images/reorder", productCtrl.ReorderImages)
	}
}
