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
	"github.com/yaikob92/ecommerce/config"
	"github.com/yaikob92/ecommerce/pkg/database"
	"github.com/yaikob92/ecommerce/pkg/middleware"
)

func main() {
	// 1. Load Configuration
	cfg := config.LoadConfig()

	// 2. Setup Database Connection
	dbPool, err := database.ConnectPostgres(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// 3. Configure Web Framework (Gin)
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 4. Setup Middlewares
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorRecovery())

	// Health check route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "connected"})
	})

	// 5. Start Server with Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("Server is running on port %s...", cfg.Port)
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
