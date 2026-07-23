package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend/dto"
	"backend/repositories"
	"backend/services"
	"backend/utils"
	"backend/validators"

	"github.com/gin-gonic/gin"
)

const (
	accessTokenCookieMaxAge  = 15 * 60           // 15 minutes
	refreshTokenCookieMaxAge = 30 * 24 * 60 * 60 // 30 days
)

// AuthController handles all auth-related HTTP endpoints.
type AuthController struct {
	authSvc services.AuthService
	env     string
}

func NewAuthController(authSvc services.AuthService, env string) *AuthController {
	return &AuthController{
		authSvc: authSvc,
		env:     env,
	}
}

func (ac *AuthController) isProduction() bool {
	return ac.env == "production"
}

// ── Register ──

// Register godoc
// @Summary      Register a new user
// @Description  Creates a new user account and sends an email verification link. The user must verify their email before logging in.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RegisterRequest  true  "Registration data"
// @Success      201      {object}  dto.SuccessResponse  "Account created successfully"
// @Failure      400      {object}  dto.ErrorResponse    "Validation error or weak password"
// @Failure      409      {object}  dto.ErrorResponse    "Email already taken"
// @Failure      500      {object}  dto.ErrorResponse    "Internal server error"
// @Router       /auth/register [post]
func (ac *AuthController) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Email, password, first_name, and last_name are required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	user, _, err := ac.authSvc.Register(c.Request.Context(), req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		if errors.Is(err, repositories.ErrEmailTaken) {
			c.JSON(http.StatusConflict, dto.ErrorResponse{
				Success: false,
				Message: "An account with this email already exists.",
				Error:   dto.ErrorDetail{Code: "EMAIL_TAKEN"},
			})
			return
		}
		if isPasswordValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: "WEAK_PASSWORD"},
			})
			return
		}
		slog.Error("Registration failed", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "An unexpected error occurred. Please try again.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse{
		Success: true,
		Message: "Account created. Please check your email to verify your account.",
		Data: gin.H{
			"user": gin.H{
				"id":         user.ID,
				"email":      user.Email,
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"role":       user.Role,
			},
		},
	})
}

// ── Login ──

// Login godoc
// @Summary      Log in a user
// @Description  Authenticates a user with email and password. Sets HttpOnly cookies for access and refresh tokens.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest  true  "Login credentials"
// @Success      200      {object}  dto.SuccessResponse  "Login successful"
// @Failure      400      {object}  dto.ErrorResponse    "Validation error"
// @Failure      401      {object}  dto.ErrorResponse    "Invalid credentials"
// @Failure      403      {object}  dto.ErrorResponse    "Email not verified"
// @Failure      429      {object}  dto.ErrorResponse    "Account locked"
// @Failure      500      {object}  dto.ErrorResponse    "Internal server error"
// @Router       /auth/login [post]
func (ac *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Email and password are required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	user, accessToken, refreshToken, _, err := ac.authSvc.Login(c.Request.Context(), req.Email, req.Password, ipAddress, userAgent)
	if err != nil {
		code := "INTERNAL_ERROR"
		status := http.StatusInternalServerError
		message := "An unexpected error occurred."

		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			code = "INVALID_CREDENTIALS"
			status = http.StatusUnauthorized
			message = "Invalid email or password."
		case errors.Is(err, services.ErrAccountLocked):
			code = "ACCOUNT_LOCKED"
			status = http.StatusTooManyRequests
			message = "Account locked due to too many failed attempts. Try again in 15 minutes."
		case errors.Is(err, services.ErrEmailNotVerified):
			code = "EMAIL_NOT_VERIFIED"
			status = http.StatusForbidden
			message = "Please verify your email address before logging in."
		}

		c.JSON(status, dto.ErrorResponse{
			Success: false,
			Message: message,
			Error:   dto.ErrorDetail{Code: code},
		})
		return
	}

	// Set HttpOnly Secure cookies
	ac.setAuthCookies(c, accessToken, refreshToken)

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Login successful.",
		Data: gin.H{
			"user": gin.H{
				"id":          user.ID,
				"email":       user.Email,
				"first_name":  user.FirstName,
				"last_name":   user.LastName,
				"role":        user.Role,
				"is_verified": user.IsVerified,
			},
		},
	})
}

// ── Verify Email ──

// VerifyEmail godoc
// @Summary      Verify email address
// @Description  Verifies a user's email address using the token sent via email.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      dto.VerifyEmailRequest  true  "Verification token"
// @Success      200      {object}  dto.SuccessResponse     "Email verified successfully"
// @Failure      400      {object}  dto.ErrorResponse       "Invalid or expired token"
// @Router       /auth/verify-email [post]
func (ac *AuthController) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Verification token is required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	if err := ac.authSvc.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Invalid or expired verification token.",
			Error:   dto.ErrorDetail{Code: "INVALID_TOKEN"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Email verified successfully. You can now log in.",
	})
}

// ── Resend Verification ──

// ResendVerification godoc
// @Summary      Resend verification email
// @Description  Resends the email verification link. Always returns success to prevent email enumeration.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ResendVerificationRequest  true  "Email address"
// @Success      200      {object}  dto.SuccessResponse            "Verification email sent (if account exists)"
// @Failure      400      {object}  dto.ErrorResponse              "Validation error"
// @Router       /auth/resend-verification [post]
func (ac *AuthController) ResendVerification(c *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Email is required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	// Always return success to prevent email enumeration
	ac.authSvc.ResendVerification(c.Request.Context(), req.Email)

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "If an account exists with that email, a verification link has been sent.",
	})
}

// ── Refresh ──

// Refresh godoc
// @Summary      Refresh access token
// @Description  Rotates the refresh token (read from HttpOnly cookie) and issues a new access token. No Authorization header needed.
// @Tags         Authentication
// @Produce      json
// @Success      200  {object}  dto.SuccessResponse  "Tokens refreshed"
// @Failure      401  {object}  dto.ErrorResponse    "Invalid or reused refresh token"
// @Router       /auth/refresh [post]
func (ac *AuthController) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Message: "Refresh token not found.",
			Error:   dto.ErrorDetail{Code: "NO_REFRESH_TOKEN"},
		})
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	newAccessToken, newRefreshToken, err := ac.authSvc.RefreshTokens(c.Request.Context(), refreshToken, ipAddress, userAgent)
	if err != nil {
		// Clear cookies on failure
		ac.clearAuthCookies(c)

		code := "INVALID_TOKEN"
		message := "Session expired. Please log in again."

		if errors.Is(err, services.ErrTokenReuse) {
			code = "TOKEN_REUSE_DETECTED"
			message = "Security alert: session compromise detected. All sessions have been revoked."
		}

		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Message: message,
			Error:   dto.ErrorDetail{Code: code},
		})
		return
	}

	ac.setAuthCookies(c, newAccessToken, newRefreshToken)

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Tokens refreshed.",
	})
}

// ── Forgot Password ──

// ForgotPassword godoc
// @Summary      Request password reset
// @Description  Sends a password reset link to the provided email. Always returns success to prevent email enumeration.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ForgotPasswordRequest  true  "Email address"
// @Success      200      {object}  dto.SuccessResponse        "Reset link sent (if account exists)"
// @Failure      400      {object}  dto.ErrorResponse          "Validation error"
// @Router       /auth/forgot-password [post]
func (ac *AuthController) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Email is required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	// Always return success to prevent email enumeration
	ac.authSvc.ForgotPassword(c.Request.Context(), req.Email)

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "If an account exists with that email, a password reset link has been sent.",
	})
}

// ── Reset Password ──

// ResetPassword godoc
// @Summary      Reset password
// @Description  Resets the user's password using a valid reset token. All active sessions are revoked after a successful reset.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      dto.ResetPasswordRequest  true  "Reset token and new password"
// @Success      200      {object}  dto.SuccessResponse       "Password reset successful"
// @Failure      400      {object}  dto.ErrorResponse         "Invalid token or weak password"
// @Router       /auth/reset-password [post]
func (ac *AuthController) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Token and new password are required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	if err := ac.authSvc.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		if isPasswordValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: "WEAK_PASSWORD"},
			})
			return
		}
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Invalid or expired reset token.",
			Error:   dto.ErrorDetail{Code: "INVALID_TOKEN"},
		})
		return
	}

	ac.clearAuthCookies(c)

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Password reset successful. Please log in with your new password.",
	})
}

// ── Logout ──

// Logout godoc
// @Summary      Log out current session
// @Description  Revokes the current session and clears authentication cookies.
// @Tags         Session Management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.SuccessResponse  "Logged out successfully"
// @Router       /auth/logout [post]
func (ac *AuthController) Logout(c *gin.Context) {
	// Try to find session from refresh token cookie
	refreshToken, _ := c.Cookie("refresh_token")
	if refreshToken != "" {
		tokenHash := utils.HashSHA256(refreshToken)
		// Best effort — revoke the session matching this token
		// The service layer can handle this if we expose a method, but for simplicity
		// we revoke based on the user in context
		userID, exists := c.Get("userID")
		if exists {
			ac.authSvc.Logout(c.Request.Context(), userID.(string))
		}
		_ = tokenHash // used for logging if needed
	}

	ac.clearAuthCookies(c)

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Logged out successfully.",
	})
}

// LogoutAll godoc
// @Summary      Log out all sessions
// @Description  Revokes all active sessions for the current user across all devices.
// @Tags         Session Management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.SuccessResponse  "All sessions revoked"
// @Failure      401  {object}  dto.ErrorResponse    "Authentication required"
// @Failure      500  {object}  dto.ErrorResponse    "Internal server error"
// @Router       /auth/logout-all [post]
func (ac *AuthController) LogoutAll(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Success: false,
			Message: "Authentication required.",
			Error:   dto.ErrorDetail{Code: "UNAUTHORIZED"},
		})
		return
	}

	if err := ac.authSvc.LogoutAll(c.Request.Context(), userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to logout all sessions.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	ac.clearAuthCookies(c)

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "All sessions have been revoked.",
	})
}

// ── Sessions ──

// GetSessions godoc
// @Summary      List active sessions
// @Description  Returns all active sessions for the authenticated user, marking the current session.
// @Tags         Session Management
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.SuccessResponse  "List of active sessions"
// @Failure      500  {object}  dto.ErrorResponse    "Internal server error"
// @Router       /auth/sessions [get]
func (ac *AuthController) GetSessions(c *gin.Context) {
	userID := c.GetString("userID")

	sessions, err := ac.authSvc.GetActiveSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to retrieve sessions.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	// Determine current session from refresh token
	currentRefreshToken, _ := c.Cookie("refresh_token")
	currentHash := ""
	if currentRefreshToken != "" {
		currentHash = utils.HashSHA256(currentRefreshToken)
	}

	var response []dto.SessionResponse
	for _, s := range sessions {
		response = append(response, dto.SessionResponse{
			ID:              s.ID,
			DeviceName:      s.DeviceName,
			Browser:         s.Browser,
			OperatingSystem: s.OperatingSystem,
			IPAddress:       s.IPAddress,
			LastActive:      s.LastUsedAt.Format(time.RFC3339),
			IsCurrent:       s.RefreshTokenHash == currentHash,
		})
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data:    gin.H{"sessions": response},
	})
}

// RevokeSession godoc
// @Summary      Revoke a specific session
// @Description  Revokes a specific session by ID. The session must belong to the authenticated user.
// @Tags         Session Management
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string               true  "Session ID"
// @Success      200  {object}  dto.SuccessResponse   "Session revoked"
// @Failure      404  {object}  dto.ErrorResponse     "Session not found"
// @Router       /auth/sessions/{id} [delete]
func (ac *AuthController) RevokeSession(c *gin.Context) {
	userID := c.GetString("userID")
	sessionID := c.Param("id")

	if err := ac.authSvc.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Message: "Session not found.",
			Error:   dto.ErrorDetail{Code: "SESSION_NOT_FOUND"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Session revoked.",
	})
}

// ── Me ──

// Me godoc
// @Summary      Get current user profile
// @Description  Returns the profile information of the authenticated user.
// @Tags         User
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.SuccessResponse  "User profile"
// @Failure      404  {object}  dto.ErrorResponse    "User not found"
// @Router       /auth/me [get]
func (ac *AuthController) Me(c *gin.Context) {
	userID := c.GetString("userID")

	user, err := ac.authSvc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Success: false,
			Message: "User not found.",
			Error:   dto.ErrorDetail{Code: "USER_NOT_FOUND"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Data: gin.H{
			"user": gin.H{
				"id":                  user.ID,
				"email":               user.Email,
				"first_name":          user.FirstName,
				"last_name":           user.LastName,
				"role":                user.Role,
				"is_verified":         user.IsVerified,
				"phone":               user.Phone,
				"company_name":        user.CompanyName,
				"address":             user.Address,
				"profile_picture_url": user.ProfilePictureURL,
				"preferred_language":  user.PreferredLanguage,
				"timezone":            user.Timezone,
				"email_notifications": user.EmailNotifications,
				"sms_notifications":   user.SMSNotifications,
				"marketing_emails":    user.MarketingEmails,
				"created_at":          user.CreatedAt.Format(time.RFC3339),
				"updated_at":          user.UpdatedAt.Format(time.RFC3339),
			},
		},
	})
}

// ── Update Profile ──

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Updates the profile information of the authenticated user.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.UpdateProfileRequest  true  "Profile data"
// @Success      200      {object}  dto.SuccessResponse       "Profile updated"
// @Failure      400      {object}  dto.ErrorResponse         "Validation error"
// @Failure      401      {object}  dto.ErrorResponse         "Authentication required"
// @Router       /auth/me [put]
func (ac *AuthController) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "First name and last name are required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	user, err := ac.authSvc.UpdateUserProfile(c.Request.Context(), userID, req)
	if err != nil {
		slog.Error("Failed to update profile", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to update profile.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Profile updated successfully.",
		Data: gin.H{
			"user": gin.H{
				"id":                  user.ID,
				"email":               user.Email,
				"first_name":          user.FirstName,
				"last_name":           user.LastName,
				"role":                user.Role,
				"is_verified":         user.IsVerified,
				"phone":               user.Phone,
				"company_name":        user.CompanyName,
				"address":             user.Address,
				"profile_picture_url": user.ProfilePictureURL,
				"preferred_language":  user.PreferredLanguage,
				"timezone":            user.Timezone,
				"email_notifications": user.EmailNotifications,
				"sms_notifications":   user.SMSNotifications,
				"marketing_emails":    user.MarketingEmails,
				"created_at":          user.CreatedAt.Format(time.RFC3339),
				"updated_at":          user.UpdatedAt.Format(time.RFC3339),
			},
		},
	})
}

// ── Change Password ──

// ChangePassword godoc
// @Summary      Change password
// @Description  Changes the password of the authenticated user. Requires current password.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.ChangePasswordRequest  true  "Password data"
// @Success      200      {object}  dto.SuccessResponse        "Password changed"
// @Failure      400      {object}  dto.ErrorResponse          "Validation error or weak password"
// @Failure      401      {object}  dto.ErrorResponse          "Invalid current password"
// @Router       /auth/me/password [put]
func (ac *AuthController) ChangePassword(c *gin.Context) {
	userID := c.GetString("userID")

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Current password and new password are required.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	err := ac.authSvc.ChangeUserPassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Success: false,
				Message: "Current password is incorrect.",
				Error:   dto.ErrorDetail{Code: "INVALID_PASSWORD"},
			})
			return
		}
		if isPasswordValidationError(err) {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Success: false,
				Message: err.Error(),
				Error:   dto.ErrorDetail{Code: "WEAK_PASSWORD"},
			})
			return
		}
		slog.Error("Failed to change password", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to change password.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Password changed successfully.",
	})
}

// ── Upload Avatar ──

// UploadAvatar godoc
// @Summary      Upload avatar
// @Description  Uploads a profile picture for the authenticated user.
// @Tags         User
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        avatar  formData  file  true  "Avatar image"
// @Success      200     {object}  dto.SuccessResponse  "Avatar uploaded"
// @Failure      400     {object}  dto.ErrorResponse    "No file or invalid file"
// @Failure      401     {object}  dto.ErrorResponse    "Authentication required"
// @Router       /auth/me/avatar [post]
func (ac *AuthController) UploadAvatar(c *gin.Context) {
	userID := c.GetString("userID")

	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "No file uploaded.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	if file.Size > 5<<20 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "File too large. Maximum size is 5MB.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	ext := ""
	if i := strings.LastIndexByte(file.Filename, '.'); i >= 0 {
		ext = file.Filename[i+1:]
	}
	allowed := map[string]bool{"jpg": true, "jpeg": true, "png": true, "webp": true, "gif": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Invalid file type. Allowed: jpg, jpeg, png, webp, gif.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	filename := userID + "." + ext
	savePath := "uploads/avatars/" + filename

	if err := c.SaveUploadedFile(file, savePath); err != nil {
		slog.Error("Failed to save avatar", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to save file.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	pictureURL := "/uploads/avatars/" + filename
	if err := ac.authSvc.UpdateProfilePicture(c.Request.Context(), userID, pictureURL); err != nil {
		slog.Error("Failed to update profile picture", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to update profile picture.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Avatar uploaded successfully.",
		Data: gin.H{
			"profile_picture_url": pictureURL,
		},
	})
}

// ── Update Notification Preferences ──

// UpdateNotificationPreferences godoc
// @Summary      Update notification preferences
// @Description  Updates the notification settings of the authenticated user.
// @Tags         User
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      dto.UpdateNotificationPreferencesRequest  true  "Notification settings"
// @Success      200      {object}  dto.SuccessResponse                       "Preferences updated"
// @Failure      400      {object}  dto.ErrorResponse                         "Validation error"
// @Failure      401      {object}  dto.ErrorResponse                         "Authentication required"
// @Router       /auth/me/notifications [put]
func (ac *AuthController) UpdateNotificationPreferences(c *gin.Context) {
	userID := c.GetString("userID")

	var req dto.UpdateNotificationPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Success: false,
			Message: "Invalid request body.",
			Error:   dto.ErrorDetail{Code: "VALIDATION_ERROR"},
		})
		return
	}

	err := ac.authSvc.UpdateNotificationPreferences(c.Request.Context(), userID, req)
	if err != nil {
		slog.Error("Failed to update notification preferences", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Success: false,
			Message: "Failed to update notification preferences.",
			Error:   dto.ErrorDetail{Code: "INTERNAL_ERROR"},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Notification preferences updated successfully.",
	})
}

// ── Cookie Helpers ──

func (ac *AuthController) setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	secure := ac.isProduction()

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", accessToken, accessTokenCookieMaxAge, "/", "", secure, true)            // HttpOnly
	c.SetCookie("refresh_token", refreshToken, refreshTokenCookieMaxAge, "/api/auth", "", secure, true) // HttpOnly, scoped path
}

func (ac *AuthController) clearAuthCookies(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", ac.isProduction(), true)
	c.SetCookie("refresh_token", "", -1, "/api/auth", "", ac.isProduction(), true)
	c.SetCookie("csrf_token", "", -1, "/", "", ac.isProduction(), false)
}

// ── Helpers ──

func isPasswordValidationError(err error) bool {
	return errors.Is(err, validators.ErrPasswordTooShort) ||
		errors.Is(err, validators.ErrPasswordNoUpper) ||
		errors.Is(err, validators.ErrPasswordNoLower) ||
		errors.Is(err, validators.ErrPasswordNoDigit) ||
		errors.Is(err, validators.ErrPasswordNoSpecial)
}
