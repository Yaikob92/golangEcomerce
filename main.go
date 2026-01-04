package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/yaikob92/ecommerce/controllers"
	"github.com/yaikob92/ecommerce/database"
	"github.com/yaikob92/ecommerce/middleware"
	"github.com/yaikob92/ecommerce/routes"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database connection

	app := controllers.NewApplication(database.ProductData(database.client, "Products"), database.UserData(database.client, "Users"))

	// Initialize Gin router
	router := gin.Default()
	router.Use(gin.Logger())

	routes.UserRoutes(router)
	router.Use(middleware.Authentication())

	router.GET("/addtocart", app.AddToCart)
	router.GET("/removeitem", app.RemoveItem)
	router.GET("/cartcheckout", app.BuyFromCart)
	router.GET("/instantbuy", app.InstantBuy)

	// Start server
	log.Printf("Server starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
