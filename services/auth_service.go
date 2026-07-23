package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"backend/config"
	"backend/dto"
	"backend/models"
	"backend/repositories"
	"backend/utils"
	"backend/validators"
)

const (
	MaxFailedLoginAttempts = 10
	AccountLockoutDuration = 15 * time.Minute
	RefreshTokenExpiry     = 30 * 24 * time.Hour // 30 days
	VerificationTokenExpiry = 60                  // minutes
	PasswordResetExpiry     = 30                  // minutes
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountLocked      = errors.New("account is locked due to too many failed login attempts")
	ErrEmailNotVerified   = errors.New("email address has not been verified")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrTokenReuse         = errors.New("token reuse detected, all sessions revoked")
)

// AuthService holds all authentication business logic.
type AuthService interface {
	Register(ctx context.Context, email, password, firstName, lastName string) (models.User, string, error)
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (user models.User, accessToken string, refreshToken string, session models.Session, err error)
	VerifyEmail(ctx context.Context, token string) error
	ResendVerification(ctx context.Context, email string) error
	RefreshTokens(ctx context.Context, oldRefreshToken, ipAddress, userAgent string) (accessToken string, newRefreshToken string, err error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	Logout(ctx context.Context, sessionID string) error
	LogoutAll(ctx context.Context, userID string) error
	GetActiveSessions(ctx context.Context, userID string) ([]models.Session, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	GetUserByID(ctx context.Context, id string) (models.User, error)
	UpdateUserProfile(ctx context.Context, userID string, req dto.UpdateProfileRequest) (models.User, error)
	ChangeUserPassword(ctx context.Context, userID, currentPassword, newPassword string) error
	UpdateProfilePicture(ctx context.Context, userID, pictureURL string) error
	UpdateNotificationPreferences(ctx context.Context, userID string, req dto.UpdateNotificationPreferencesRequest) error
}

type authService struct {
	userRepo    repositories.UserRepository
	sessionRepo repositories.SessionRepository
	tokenRepo   repositories.TokenRepository
	auditRepo   repositories.AuditRepository
	emailSvc    EmailService
	cfg         *config.Config
}

func NewAuthService(
	userRepo repositories.UserRepository,
	sessionRepo repositories.SessionRepository,
	tokenRepo repositories.TokenRepository,
	auditRepo repositories.AuditRepository,
	emailSvc EmailService,
	cfg *config.Config,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenRepo:   tokenRepo,
		auditRepo:   auditRepo,
		emailSvc:    emailSvc,
		cfg:         cfg,
	}
}

// ── Register ──

func (s *authService) Register(ctx context.Context, email, password, firstName, lastName string) (models.User, string, error) {
	// Validate password strength
	if err := validators.ValidatePassword(password); err != nil {
		return models.User{}, "", err
	}

	// Hash password with Argon2id
	hash, err := utils.GenerateArgon2Hash(password)
	if err != nil {
		return models.User{}, "", err
	}

	// Create user
	user, err := s.userRepo.Create(ctx, email, hash, firstName, lastName, "customer")
	if err != nil {
		return models.User{}, "", err
	}

	// Generate email verification token
	token, err := utils.GenerateSecureToken()
	if err != nil {
		slog.Error("Failed to generate verification token", slog.Any("error", err))
		return user, "", nil // User created, but verification failed — non-fatal
	}

	tokenHash := utils.HashSHA256(token)
	if err := s.tokenRepo.CreateEmailVerification(ctx, user.ID, tokenHash, VerificationTokenExpiry); err != nil {
		slog.Error("Failed to save verification token", slog.Any("error", err))
		return user, "", nil
	}

	// Send verification email in background (non-blocking)
	go func() {
		bgCtx := context.Background()
		if err := s.emailSvc.SendVerificationEmail(bgCtx, email, token); err != nil {
			slog.Error("Failed to send verification email", slog.Any("error", err))
		}
	}()

	return user, token, nil
}

// ── Login ──

func (s *authService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (models.User, string, string, models.Session, error) {
	deviceInfo := utils.ParseUserAgent(userAgent)

	// Find user
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repositories.ErrUserNotFound) {
			return models.User{}, "", "", models.Session{}, ErrInvalidCredentials
		}
		return models.User{}, "", "", models.Session{}, err
	}

	// Check account lockout
	if user.LockoutUntil != nil && user.LockoutUntil.After(time.Now()) {
		s.auditRepo.Log(ctx, &user.ID, "login_failed_locked", ipAddress, deviceInfo.DeviceName, deviceInfo.Browser, deviceInfo.OperatingSystem)
		return models.User{}, "", "", models.Session{}, ErrAccountLocked
	}

	// Verify password
	match, err := utils.CompareArgon2Hash(password, user.PasswordHash)
	if err != nil || !match {
		// Increment failed login attempts
		attempts, _ := s.userRepo.IncrementFailedLogin(ctx, user.ID)
		s.auditRepo.Log(ctx, &user.ID, "login_failed", ipAddress, deviceInfo.DeviceName, deviceInfo.Browser, deviceInfo.OperatingSystem)

		if attempts >= MaxFailedLoginAttempts {
			s.userRepo.LockAccount(ctx, user.ID, AccountLockoutDuration)
			s.auditRepo.Log(ctx, &user.ID, "account_locked", ipAddress, deviceInfo.DeviceName, deviceInfo.Browser, deviceInfo.OperatingSystem)
			return models.User{}, "", "", models.Session{}, ErrAccountLocked
		}

		return models.User{}, "", "", models.Session{}, ErrInvalidCredentials
	}

	// Check email verification
	if !user.IsVerified {
		return models.User{}, "", "", models.Session{}, ErrEmailNotVerified
	}

	// Reset failed login attempts on success
	s.userRepo.ResetFailedLogin(ctx, user.ID)

	// Generate access token
	accessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Role, s.cfg.JWT.Secret, s.cfg.JWT.Issuer, s.cfg.JWT.Audience)
	if err != nil {
		return models.User{}, "", "", models.Session{}, err
	}

	// Generate refresh token (cryptographically random, NOT JWT)
	refreshToken, err := utils.GenerateSecureToken()
	if err != nil {
		return models.User{}, "", "", models.Session{}, err
	}

	// Create session with hashed refresh token
	session := models.Session{
		UserID:           user.ID,
		RefreshTokenHash: utils.HashSHA256(refreshToken),
		DeviceName:       deviceInfo.DeviceName,
		Browser:          deviceInfo.Browser,
		OperatingSystem:  deviceInfo.OperatingSystem,
		IPAddress:        ipAddress,
		UserAgent:        userAgent,
		ExpiresAt:        time.Now().Add(RefreshTokenExpiry),
	}

	if err := s.sessionRepo.Create(ctx, &session); err != nil {
		return models.User{}, "", "", models.Session{}, err
	}

	s.auditRepo.Log(ctx, &user.ID, "login", ipAddress, deviceInfo.DeviceName, deviceInfo.Browser, deviceInfo.OperatingSystem)

	return user, accessToken, refreshToken, session, nil
}

// ── Verify Email ──

func (s *authService) VerifyEmail(ctx context.Context, token string) error {
	tokenHash := utils.HashSHA256(token)

	verification, err := s.tokenRepo.GetEmailVerification(ctx, tokenHash)
	if err != nil {
		return ErrInvalidToken
	}

	// Check expiration
	if time.Now().After(verification.ExpiresAt) {
		s.tokenRepo.DeleteEmailVerification(ctx, verification.ID)
		return ErrInvalidToken
	}

	// Mark user as verified
	if err := s.userRepo.UpdateVerificationStatus(ctx, verification.UserID, true); err != nil {
		return err
	}

	// Delete all verification tokens for this user (one-time use)
	s.tokenRepo.DeleteAllEmailVerificationsForUser(ctx, verification.UserID)

	s.auditRepo.Log(ctx, &verification.UserID, "email_verified", "", "", "", "")

	return nil
}

// ── Resend Verification ──

func (s *authService) ResendVerification(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	if user.IsVerified {
		return nil
	}

	// Delete old tokens
	s.tokenRepo.DeleteAllEmailVerificationsForUser(ctx, user.ID)

	// Generate new token
	token, err := utils.GenerateSecureToken()
	if err != nil {
		return err
	}

	tokenHash := utils.HashSHA256(token)
	if err := s.tokenRepo.CreateEmailVerification(ctx, user.ID, tokenHash, VerificationTokenExpiry); err != nil {
		return err
	}

	return s.emailSvc.SendVerificationEmail(ctx, email, token)
}

// ── Refresh Token Rotation ──

func (s *authService) RefreshTokens(ctx context.Context, oldRefreshToken, ipAddress, userAgent string) (string, string, error) {
	oldHash := utils.HashSHA256(oldRefreshToken)

	// Find session by hashed token
	session, err := s.sessionRepo.GetByTokenHash(ctx, oldHash)
	if err != nil {
		if errors.Is(err, repositories.ErrSessionNotFound) {
			// Possible replay attack — the token was already rotated.
			// We cannot determine the user from the token alone, so just reject.
			return "", "", ErrInvalidToken
		}
		return "", "", err
	}

	// Check if session is revoked (replay attack detection)
	if session.RevokedAt != nil {
		// Token reuse detected! Revoke ALL sessions for this user.
		slog.Warn("Refresh token reuse detected — revoking all sessions",
			slog.String("user_id", session.UserID),
			slog.String("session_id", session.ID),
			slog.String("ip", ipAddress),
		)
		s.sessionRepo.RevokeAllForUser(ctx, session.UserID)
		s.auditRepo.Log(ctx, &session.UserID, "token_reuse_detected", ipAddress, "", "", "")
		return "", "", ErrTokenReuse
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		s.sessionRepo.Revoke(ctx, session.ID)
		return "", "", ErrInvalidToken
	}

	// Get user for new access token
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return "", "", err
	}

	// Generate new access token
	newAccessToken, err := utils.GenerateAccessToken(user.ID, user.Email, user.Role, s.cfg.JWT.Secret, s.cfg.JWT.Issuer, s.cfg.JWT.Audience)
	if err != nil {
		return "", "", err
	}

	// Generate new refresh token
	newRefreshToken, err := utils.GenerateSecureToken()
	if err != nil {
		return "", "", err
	}

	newHash := utils.HashSHA256(newRefreshToken)

	// Update session with new hash (old hash is now invalid)
	if err := s.sessionRepo.UpdateTokenHash(ctx, session.ID, newHash, time.Now().Add(RefreshTokenExpiry)); err != nil {
		return "", "", err
	}

	s.auditRepo.Log(ctx, &user.ID, "token_refresh", ipAddress, "", "", "")

	return newAccessToken, newRefreshToken, nil
}

// ── Forgot Password ──

func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		return nil
	}

	// Delete old reset tokens
	s.tokenRepo.DeleteAllPasswordResetsForUser(ctx, user.ID)

	// Generate new reset token
	token, err := utils.GenerateSecureToken()
	if err != nil {
		return err
	}

	tokenHash := utils.HashSHA256(token)
	if err := s.tokenRepo.CreatePasswordReset(ctx, user.ID, tokenHash, PasswordResetExpiry); err != nil {
		return err
	}

	s.auditRepo.Log(ctx, &user.ID, "password_reset_requested", "", "", "", "")

	return s.emailSvc.SendPasswordResetEmail(ctx, email, token)
}

// ── Reset Password ──

func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := validators.ValidatePassword(newPassword); err != nil {
		return err
	}

	tokenHash := utils.HashSHA256(token)

	reset, err := s.tokenRepo.GetPasswordReset(ctx, tokenHash)
	if err != nil {
		return ErrInvalidToken
	}

	// Check expiration
	if time.Now().After(reset.ExpiresAt) {
		s.tokenRepo.DeletePasswordReset(ctx, reset.ID)
		return ErrInvalidToken
	}

	// Hash new password
	hash, err := utils.GenerateArgon2Hash(newPassword)
	if err != nil {
		return err
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, reset.UserID, hash); err != nil {
		return err
	}

	// Delete all reset tokens (one-time use)
	s.tokenRepo.DeleteAllPasswordResetsForUser(ctx, reset.UserID)

	// Revoke all sessions (security: force re-login)
	s.sessionRepo.RevokeAllForUser(ctx, reset.UserID)

	s.auditRepo.Log(ctx, &reset.UserID, "password_reset", "", "", "", "")

	return nil
}

// ── Logout ──

func (s *authService) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Revoke(ctx, sessionID)
}

func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	s.auditRepo.Log(ctx, &userID, "logout_all", "", "", "", "")
	return s.sessionRepo.RevokeAllForUser(ctx, userID)
}

// ── Sessions ──

func (s *authService) GetActiveSessions(ctx context.Context, userID string) ([]models.Session, error) {
	return s.sessionRepo.GetActiveForUser(ctx, userID)
}

func (s *authService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	// Ensure the user owns this session
	if session.UserID != userID {
		return repositories.ErrSessionNotFound
	}
	return s.sessionRepo.Revoke(ctx, sessionID)
}

func (s *authService) GetUserByID(ctx context.Context, id string) (models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// ── Update Profile ──

func (s *authService) UpdateUserProfile(ctx context.Context, userID string, req dto.UpdateProfileRequest) (models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return models.User{}, err
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Phone = req.Phone
	user.CompanyName = req.CompanyName
	user.Address = req.Address
	user.PreferredLanguage = req.PreferredLanguage
	user.Timezone = req.Timezone

	if err := s.userRepo.UpdateProfile(ctx, user); err != nil {
		return models.User{}, err
	}

	s.auditRepo.Log(ctx, &userID, "profile_updated", "", "", "", "")

	return user, nil
}

// ── Change Password ──

func (s *authService) ChangeUserPassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if err := validators.ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	match, err := utils.CompareArgon2Hash(currentPassword, user.PasswordHash)
	if err != nil || !match {
		return ErrInvalidCredentials
	}

	hash, err := utils.GenerateArgon2Hash(newPassword)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}

	s.auditRepo.Log(ctx, &userID, "password_changed", "", "", "", "")

	return nil
}

// ── Update Profile Picture ──

func (s *authService) UpdateProfilePicture(ctx context.Context, userID, pictureURL string) error {
	if err := s.userRepo.UpdateProfilePicture(ctx, userID, pictureURL); err != nil {
		return err
	}

	s.auditRepo.Log(ctx, &userID, "profile_picture_updated", "", "", "", "")

	return nil
}

// ── Update Notification Preferences ──

func (s *authService) UpdateNotificationPreferences(ctx context.Context, userID string, req dto.UpdateNotificationPreferencesRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	user.EmailNotifications = req.EmailNotifications
	user.SMSNotifications = req.SMSNotifications
	user.MarketingEmails = req.MarketingEmails

	if err := s.userRepo.UpdateNotificationPreferences(ctx, user); err != nil {
		return err
	}

	s.auditRepo.Log(ctx, &userID, "notification_preferences_updated", "", "", "", "")

	return nil
}
