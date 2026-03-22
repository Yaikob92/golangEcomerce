package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/yaikob92/ecommerce/config"
	_ "github.com/yaikob92/ecommerce/docs" // Swagger generated docs
	"github.com/yaikob92/ecommerce/internal/auth"
	"github.com/yaikob92/ecommerce/pkg/database"
	"github.com/yaikob92/ecommerce/pkg/middleware"
)

// @title           E-Commerce API
// @version         1.0
// @description     A high-performance Go e-commerce backend API with JWT authentication
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@ecommerce.io

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your Bearer token in the format: Bearer {token}

func main() {
	// 1. Load Configuration
	cfg := config.LoadConfig()

	// 2. Setup Database Connection
	db, err := database.ConnectPostgres(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	// Auto Migrate models
	err = db.AutoMigrate(&auth.User{}, &auth.RefreshToken{})
	if err != nil {
		log.Fatalf("Failed to run auto-migrations: %v", err)
	}

	// 3. Configure Web Framework (Gin)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 4. Setup Middlewares
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorRecovery())

	// 5. Initialize Repository & Handler
	authRepo := auth.NewRepository(db)
	authHandler := auth.NewHandler(authRepo, cfg)

	// 6. Setup Routes
	v1 := router.Group("/api/v1")
	{
		// Public Auth routes
		auth.RegisterRoutes(v1, authHandler)

		// Health check route
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "connected"})
		})

		// Protected Route Example
		v1.GET("/profile", middleware.AuthMiddleware(cfg.JWTSecret), func(c *gin.Context) {
			userID := c.GetString("user_id")
			email := c.GetString("email")
			role := c.GetString("role")
			c.JSON(http.StatusOK, gin.H{
				"message": "Welcome to your profile!",
				"user_id": userID,
				"email":   email,
				"role":    role,
			})
		})

		// Admin Route Example
		v1.GET("/admin/dashboard", middleware.AuthMiddleware(cfg.JWTSecret), middleware.RoleMiddleware("admin"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Welcome, Administrator!"})
		})
	}

	// Swagger documentation route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 7. Start Server with Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("Server is running on port %s...", cfg.Port)
		log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen and serve error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
