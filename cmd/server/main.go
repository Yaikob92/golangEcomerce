package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/config"
	"backend/controllers"
	"backend/database"
	_ "backend/docs"
	"backend/middleware"
	"backend/repositories"
	"backend/routes"
	"backend/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Fashion E-Commerce API
// @version         1.0
// @description     A high-performance Go e-commerce backend API with full authentication and role-based access control.
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

	// 2. Setup PostgreSQL Pool Connection
	pool, err := database.InitPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	// 3. Optional Redis Connection for Rate Limiting
	var redisClient *redis.Client
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		rc, err := database.InitRedis(redisURL)
		if err != nil {
			slog.Warn("Redis connection failed, rate limiting fallback enabled", slog.Any("error", err))
		} else {
			redisClient = rc
			defer redisClient.Close()
		}
	}

	// 4. Initialize Repositories
	userRepo := repositories.NewUserRepository(pool)
	sessionRepo := repositories.NewSessionRepository(pool)
	tokenRepo := repositories.NewTokenRepository(pool)
	auditRepo := repositories.NewAuditRepository(pool)
	brandRepo := repositories.NewBrandRepository(pool)
	categoryRepo := repositories.NewCategoryRepository(pool)
	productRepo := repositories.NewProductRepository(pool)
	productImageRepo := repositories.NewProductImageRepository(pool)
	orderRepo := repositories.NewOrderRepository(pool)
	superAdminRepo := repositories.NewSuperAdminRepository(pool)

	// 5. Initialize Services
	emailSvc := services.NewEmailService(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
		cfg.SMTP.From,
		cfg.FrontendURL,
	)
	authSvc := services.NewAuthService(
		userRepo,
		sessionRepo,
		tokenRepo,
		auditRepo,
		emailSvc,
		cfg,
	)
	rateLimitSvc := services.NewRateLimitService(redisClient)
	brandSvc := services.NewBrandService(brandRepo, productRepo)
	categorySvc := services.NewCategoryService(categoryRepo, productRepo)
	productSvc := services.NewProductService(productRepo, productImageRepo, categoryRepo, brandRepo)
	productImageSvc := services.NewProductImageService(productImageRepo, productRepo)
	orderSvc := services.NewOrderService(orderRepo)
	customerSvc := services.NewCustomerService(pool)
	superAdminSvc := services.NewSuperAdminService(superAdminRepo, userRepo, auditRepo)

	// 6. Initialize Controllers
	authCtrl := controllers.NewAuthController(authSvc, cfg.Env)
	adminDashCtrl := controllers.NewAdminDashboardController(superAdminSvc)
	categoryCtrl := controllers.NewCategoryController(categorySvc)
	brandCtrl := controllers.NewBrandController(brandSvc)
	productCtrl := controllers.NewProductController(productSvc, productImageSvc)
	orderCtrl := controllers.NewOrderController(orderSvc)
	customerCtrl := controllers.NewCustomerController(customerSvc)
	superAdminCtrl := controllers.NewSuperAdminController(superAdminSvc, cfg.Env)

	// 7. Setup Router & Middlewares
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS & CSRF
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.CSRFMiddleware())

	// Auth Middleware
	authMiddleware := middleware.JWTAuthMiddleware(cfg.JWT)

	// Static route for uploaded avatars
	router.Static("/uploads", "./uploads")

	// Health check endpoint
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Unix()})
	})

	// 8. Register Routes
	routes.SetupAuthRoutes(router, authCtrl, rateLimitSvc, authMiddleware)
	routes.SetupAdminProductRoutes(router, adminDashCtrl, categoryCtrl, brandCtrl, productCtrl, orderCtrl, customerCtrl, authMiddleware)
	routes.SetupSuperAdminRoutes(router, superAdminCtrl, authMiddleware)
	routes.SetupStoreRoutes(router, categoryCtrl, brandCtrl, productCtrl)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 9. Start HTTP Server with Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("Server running in %s mode on port %s...", cfg.Env, cfg.Port)
		log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen and serve error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}
