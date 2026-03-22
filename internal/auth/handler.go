package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaikob92/ecommerce/config"
	"github.com/yaikob92/ecommerce/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	repo   Repository
	config *config.Config
}

func NewHandler(repo Repository, config *config.Config) *Handler {
	return &Handler{
		repo:   repo,
		config: config,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with name, email, password, and optional role
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      RegisterRequest  true  "Registration payload"
// @Success      201      {object}  UserResponse
// @Failure      400      {object}  map[string]string  "Validation error"
// @Failure      409      {object}  map[string]string  "Email already registered"
// @Failure      500      {object}  map[string]string  "Internal server error"
// @Router       /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default role is user if not specified or invalid
	if req.Role != "admin" {
		req.Role = "user"
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// Create user in DB
	user, err := h.repo.CreateUser(c.Request.Context(), req.Name, req.Email, string(hashedPassword), req.Role)
	if err != nil {
		// Note: A real implementation would check for PostgreSQL unique constraint violations (code 23505)
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered or database error"})
		return
	}

	c.JSON(http.StatusCreated, UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate with email and password, returns JWT access and refresh tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      LoginRequest  true  "Login credentials"
// @Success      200      {object}  AuthResponse
// @Failure      400      {object}  map[string]string  "Validation error"
// @Failure      401      {object}  map[string]string  "Invalid credentials"
// @Failure      500      {object}  map[string]string  "Internal server error"
// @Router       /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch user by email
	user, err := h.repo.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Generate Access Token (15 mins)
	accessToken, err := jwt.GenerateAccessToken(user.ID, user.Email, user.Role, h.config.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// Generate Refresh Token (7 days)
	refreshToken, err := jwt.GenerateRefreshToken(user.ID, h.config.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// Store Refresh Token in DB
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	err = h.repo.StoreRefreshToken(c.Request.Context(), user.ID, refreshToken, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange a valid refresh token for new access and refresh tokens (token rotation)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      TokenRefreshRequest  true  "Refresh token payload"
// @Success      200      {object}  AuthResponse
// @Failure      400      {object}  map[string]string  "Validation error"
// @Failure      401      {object}  map[string]string  "Invalid or expired refresh token"
// @Failure      500      {object}  map[string]string  "Internal server error"
// @Router       /api/v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req TokenRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Validate the refresh token signature and expiry
	userID, err := jwt.ValidateRefreshToken(req.RefreshToken, h.config.JWTSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	// 2. Verify the refresh token is registered in DB (not revoked)
	dbUserID, err := h.repo.GetRefreshTokenUserID(c.Request.Context(), req.RefreshToken)
	if err != nil || dbUserID != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "session revoked or invalid"})
		return
	}

	// 3. Get user details to generate a new access token
	user, err := h.repo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	// 4. Generate new Access Token
	newAccessToken, err := jwt.GenerateAccessToken(user.ID, user.Email, user.Role, h.config.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// 5. Generate new Refresh Token (optional: rotate tokens)
	newRefreshToken, err := jwt.GenerateRefreshToken(user.ID, h.config.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
		return
	}

	// 6. Delete old refresh token from DB
	_ = h.repo.DeleteRefreshToken(c.Request.Context(), req.RefreshToken)

	// 7. Store new Refresh Token in DB
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	err = h.repo.StoreRefreshToken(c.Request.Context(), user.ID, newRefreshToken, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save session"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User: UserResponse{
			ID:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		},
	})
}

// Logout godoc
// @Summary      Logout user
// @Description  Revoke the refresh token to end the session
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      LogoutRequest  true  "Logout payload containing refresh token"
// @Success      200      {object}  map[string]string  "Successfully logged out"
// @Failure      400      {object}  map[string]string  "Validation error"
// @Failure      500      {object}  map[string]string  "Internal server error"
// @Router       /api/v1/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Delete refresh token from DB to revoke session
	err := h.repo.DeleteRefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully logged out"})
}
