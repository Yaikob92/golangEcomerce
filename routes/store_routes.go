package routes

import (
	"backend/controllers"

	"github.com/gin-gonic/gin"
)

func SetupStoreRoutes(
	r *gin.Engine,
	categoryCtrl *controllers.CategoryController,
	brandCtrl *controllers.BrandController,
	productCtrl *controllers.ProductController,
) {
	store := r.Group("/api/v1/store")
	{
		// Categories (public)
		store.GET("/categories", categoryCtrl.ListActiveCategories)

		// Brands (public)
		store.GET("/brands", brandCtrl.ListActiveBrands)

		// Products (public)
		store.GET("/products", productCtrl.ListPublicProducts)
		store.GET("/products/:slug", productCtrl.GetProductBySlug)
	}
}
