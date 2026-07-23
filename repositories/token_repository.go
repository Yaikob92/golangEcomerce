package repositories

import (
	"context"
	"errors"

	"backend/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token has expired")
)

type TokenRepository interface {
	CreateEmailVerification(ctx context.Context, userID, tokenHash string, expiresInMinutes int) error
	GetEmailVerification(ctx context.Context, tokenHash string) (models.EmailVerification, error)
	DeleteEmailVerification(ctx context.Context, id string) error
	DeleteAllEmailVerificationsForUser(ctx context.Context, userID string) error

	CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresInMinutes int) error
	GetPasswordReset(ctx context.Context, tokenHash string) (models.PasswordReset, error)
	DeletePasswordReset(ctx context.Context, id string) error
	DeleteAllPasswordResetsForUser(ctx context.Context, userID string) error
}

type postgresTokenRepository struct {
	pool *pgxpool.Pool
}

func NewTokenRepository(pool *pgxpool.Pool) TokenRepository {
	return &postgresTokenRepository{pool: pool}
}

// ── Email Verification ──

func (r *postgresTokenRepository) CreateEmailVerification(ctx context.Context, userID, tokenHash string, expiresInMinutes int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verifications (user_id, token_hash, expires_at)
		 VALUES ($1, $2, CURRENT_TIMESTAMP + INTERVAL '1 minute' * $3)`,
		userID, tokenHash, expiresInMinutes,
	)
	return err
}

func (r *postgresTokenRepository) GetEmailVerification(ctx context.Context, tokenHash string) (models.EmailVerification, error) {
	var v models.EmailVerification
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at
		 FROM email_verifications WHERE token_hash = $1`,
		tokenHash,
	).Scan(&v.ID, &v.UserID, &v.TokenHash, &v.ExpiresAt, &v.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.EmailVerification{}, ErrTokenNotFound
		}
		return models.EmailVerification{}, err
	}
	return v, nil
}

func (r *postgresTokenRepository) DeleteEmailVerification(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM email_verifications WHERE id = $1", id)
	return err
}

func (r *postgresTokenRepository) DeleteAllEmailVerificationsForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM email_verifications WHERE user_id = $1", userID)
	return err
}

// ── Password Reset ──

func (r *postgresTokenRepository) CreatePasswordReset(ctx context.Context, userID, tokenHash string, expiresInMinutes int) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_resets (user_id, token_hash, expires_at)
		 VALUES ($1, $2, CURRENT_TIMESTAMP + INTERVAL '1 minute' * $3)`,
		userID, tokenHash, expiresInMinutes,
	)
	return err
}

func (r *postgresTokenRepository) GetPasswordReset(ctx context.Context, tokenHash string) (models.PasswordReset, error) {
	var p models.PasswordReset
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at
		 FROM password_resets WHERE token_hash = $1`,
		tokenHash,
	).Scan(&p.ID, &p.UserID, &p.TokenHash, &p.ExpiresAt, &p.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.PasswordReset{}, ErrTokenNotFound
		}
		return models.PasswordReset{}, err
	}
	return p, nil
}

func (r *postgresTokenRepository) DeletePasswordReset(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM password_resets WHERE id = $1", id)
	return err
}

func (r *postgresTokenRepository) DeleteAllPasswordResetsForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM password_resets WHERE user_id = $1", userID)
	return err
}
