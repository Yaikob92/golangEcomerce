package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/config"
	"backend/utils"

	"github.com/gin-gonic/gin"
)

func TestJWTAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.JWTConfig{
		Secret:   "super-secret-key-that-is-long-enough-for-testing",
		Issuer:   "test-issuer",
		Audience: "test-audience",
	}

	// Helper to set up a test router
	setupRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(JWTAuthMiddleware(cfg))
		r.GET("/test-protected", func(c *gin.Context) {
			userID := c.GetString("userID")
			userEmail := c.GetString("userEmail")
			userRole := c.GetString("userRole")
			c.JSON(http.StatusOK, gin.H{
				"userID":    userID,
				"userEmail": userEmail,
				"userRole":  userRole,
			})
		})
		return r
	}

	t.Run("missing credentials", func(t *testing.T) {
		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-protected", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("valid access token in cookie", func(t *testing.T) {
		token, err := utils.GenerateAccessToken("user-1", "user@test.com", "merchant", cfg.Secret, cfg.Issuer, cfg.Audience)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-protected", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("valid access token in header", func(t *testing.T) {
		token, err := utils.GenerateAccessToken("user-2", "user2@test.com", "customer", cfg.Secret, cfg.Issuer, cfg.Audience)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test-protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-string")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}
