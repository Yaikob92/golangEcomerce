package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"backend/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already taken")
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash, firstName, lastName, role string) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	GetByID(ctx context.Context, id string) (models.User, error)
	UpdateVerificationStatus(ctx context.Context, id string, verified bool) error
	IncrementFailedLogin(ctx context.Context, id string) (int, error)
	ResetFailedLogin(ctx context.Context, id string) error
	LockAccount(ctx context.Context, id string, duration time.Duration) error
	UpdatePassword(ctx context.Context, id string, newPasswordHash string) error
	UpdateProfile(ctx context.Context, user models.User) error
	UpdateProfilePicture(ctx context.Context, userID, pictureURL string) error
	UpdateNotificationPreferences(ctx context.Context, user models.User) error
}

type postgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{pool: pool}
}

func (r *postgresUserRepository) Create(ctx context.Context, email, passwordHash, firstName, lastName, role string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, first_name, last_name, role, is_verified)
		 VALUES ($1, $2, $3, $4, $5, FALSE)
		 RETURNING id, email, password_hash, first_name, last_name, role, is_verified, failed_login_attempts, lockout_until, 
		           COALESCE(phone, ''), COALESCE(company_name, ''), COALESCE(address, ''), COALESCE(profile_picture_url, ''), 
		           COALESCE(preferred_language, 'en'), COALESCE(timezone, 'UTC'), 
		           COALESCE(email_notifications, true), COALESCE(sms_notifications, false), COALESCE(marketing_emails, false), 
		           created_at, updated_at`,
		email, passwordHash, firstName, lastName, role,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil, 
	       &u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL, &u.PreferredLanguage, &u.Timezone, 
	       &u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		// Unique key violation
		if isDuplicateError(err) {
			return models.User{}, ErrEmailTaken
		}
		return models.User{}, fmt.Errorf("failed to create user: %w", err)
	}

	return u, nil
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, first_name, last_name, role, is_verified, failed_login_attempts, lockout_until, 
		        COALESCE(phone, ''), COALESCE(company_name, ''), COALESCE(address, ''), COALESCE(profile_picture_url, ''), 
		        COALESCE(preferred_language, 'en'), COALESCE(timezone, 'UTC'), 
		        COALESCE(email_notifications, true), COALESCE(sms_notifications, false), COALESCE(marketing_emails, false), 
		        created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil, 
	       &u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL, &u.PreferredLanguage, &u.Timezone, 
	       &u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, err
	}

	return u, nil
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, first_name, last_name, role, is_verified, failed_login_attempts, lockout_until, 
		        COALESCE(phone, ''), COALESCE(company_name, ''), COALESCE(address, ''), COALESCE(profile_picture_url, ''), 
		        COALESCE(preferred_language, 'en'), COALESCE(timezone, 'UTC'), 
		        COALESCE(email_notifications, true), COALESCE(sms_notifications, false), COALESCE(marketing_emails, false), 
		        created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Role, &u.IsVerified, &u.FailedLoginAttempts, &u.LockoutUntil, 
	       &u.Phone, &u.CompanyName, &u.Address, &u.ProfilePictureURL, &u.PreferredLanguage, &u.Timezone, 
	       &u.EmailNotifications, &u.SMSNotifications, &u.MarketingEmails, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, err
	}

	return u, nil
}

func (r *postgresUserRepository) UpdateVerificationStatus(ctx context.Context, id string, verified bool) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET is_verified = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		verified, id,
	)
	return err
}

func (r *postgresUserRepository) IncrementFailedLogin(ctx context.Context, id string) (int, error) {
	var attempts int
	err := r.pool.QueryRow(ctx,
		"UPDATE users SET failed_login_attempts = failed_login_attempts + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1 RETURNING failed_login_attempts",
		id,
	).Scan(&attempts)
	return attempts, err
}

func (r *postgresUserRepository) ResetFailedLogin(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET failed_login_attempts = 0, lockout_until = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $1",
		id,
	)
	return err
}

func (r *postgresUserRepository) LockAccount(ctx context.Context, id string, duration time.Duration) error {
	lockoutTime := time.Now().Add(duration)
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET lockout_until = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		lockoutTime, id,
	)
	return err
}

func (r *postgresUserRepository) UpdatePassword(ctx context.Context, id string, newPasswordHash string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET password_hash = $1, failed_login_attempts = 0, lockout_until = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		newPasswordHash, id,
	)
	return err
}

func (r *postgresUserRepository) UpdateProfile(ctx context.Context, user models.User) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET 
			first_name = $1, last_name = $2, phone = $3, company_name = $4, 
			address = $5, preferred_language = $6, timezone = $7, updated_at = CURRENT_TIMESTAMP 
			WHERE id = $8`,
		user.FirstName, user.LastName, user.Phone, user.CompanyName,
		user.Address, user.PreferredLanguage, user.Timezone, user.ID,
	)
	return err
}

func (r *postgresUserRepository) UpdateProfilePicture(ctx context.Context, userID, pictureURL string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE users SET profile_picture_url = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
		pictureURL, userID,
	)
	return err
}

func (r *postgresUserRepository) UpdateNotificationPreferences(ctx context.Context, user models.User) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET 
			email_notifications = $1, sms_notifications = $2, marketing_emails = $3, updated_at = CURRENT_TIMESTAMP 
			WHERE id = $4`,
		user.EmailNotifications, user.SMSNotifications, user.MarketingEmails, user.ID,
	)
	return err
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// pgx error code for unique key violation is 23505
	return containsSubstring(errStr, "23505") || containsSubstring(errStr, "unique constraint")
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && stringsIndex(s, sub) >= 0
}

func stringsIndex(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
