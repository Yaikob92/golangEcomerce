package repositories

import (
	"context"
	"errors"
	"time"

	"backend/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionRevoked  = errors.New("session has been revoked")
	ErrSessionExpired  = errors.New("session has expired")
)

type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, id string) (models.Session, error)
	GetByTokenHash(ctx context.Context, hash string) (models.Session, error)
	UpdateTokenHash(ctx context.Context, sessionID, newHash string, expiresAt time.Time) error
	Revoke(ctx context.Context, sessionID string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	GetActiveForUser(ctx context.Context, userID string) ([]models.Session, error)
}

type postgresSessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) SessionRepository {
	return &postgresSessionRepository{pool: pool}
}

func (r *postgresSessionRepository) Create(ctx context.Context, s *models.Session) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO sessions (user_id, refresh_token_hash, device_name, browser, operating_system, ip_address, user_agent, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at, last_used_at`,
		s.UserID, s.RefreshTokenHash, s.DeviceName, s.Browser, s.OperatingSystem, s.IPAddress, s.UserAgent, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt, &s.LastUsedAt)
}

func (r *postgresSessionRepository) GetByID(ctx context.Context, id string) (models.Session, error) {
	var s models.Session
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, refresh_token_hash, device_name, browser, operating_system, ip_address, user_agent, created_at, last_used_at, expires_at, revoked_at
		 FROM sessions WHERE id = $1`,
		id,
	).Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.DeviceName, &s.Browser, &s.OperatingSystem, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.LastUsedAt, &s.ExpiresAt, &s.RevokedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Session{}, ErrSessionNotFound
		}
		return models.Session{}, err
	}

	return s, nil
}

func (r *postgresSessionRepository) GetByTokenHash(ctx context.Context, hash string) (models.Session, error) {
	var s models.Session
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, refresh_token_hash, device_name, browser, operating_system, ip_address, user_agent, created_at, last_used_at, expires_at, revoked_at
		 FROM sessions WHERE refresh_token_hash = $1`,
		hash,
	).Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.DeviceName, &s.Browser, &s.OperatingSystem, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.LastUsedAt, &s.ExpiresAt, &s.RevokedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Session{}, ErrSessionNotFound
		}
		return models.Session{}, err
	}

	return s, nil
}

func (r *postgresSessionRepository) UpdateTokenHash(ctx context.Context, sessionID, newHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions 
		 SET refresh_token_hash = $1, last_used_at = CURRENT_TIMESTAMP, expires_at = $2
		 WHERE id = $3 AND revoked_at IS NULL`,
		newHash, expiresAt, sessionID,
	)
	return err
}

func (r *postgresSessionRepository) Revoke(ctx context.Context, sessionID string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE id = $1 AND revoked_at IS NULL",
		sessionID,
	)
	return err
}

func (r *postgresSessionRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND revoked_at IS NULL",
		userID,
	)
	return err
}

func (r *postgresSessionRepository) GetActiveForUser(ctx context.Context, userID string) ([]models.Session, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, refresh_token_hash, device_name, browser, operating_system, ip_address, user_agent, created_at, last_used_at, expires_at, revoked_at
		 FROM sessions 
		 WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP
		 ORDER BY last_used_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		err := rows.Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.DeviceName, &s.Browser, &s.OperatingSystem, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.LastUsedAt, &s.ExpiresAt, &s.RevokedAt)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}
