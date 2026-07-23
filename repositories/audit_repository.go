package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository interface {
	Log(ctx context.Context, userID *string, event, ipAddress, deviceName, browser, operatingSystem string) error
}

type postgresAuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) AuditRepository {
	return &postgresAuditRepository{pool: pool}
}

func (r *postgresAuditRepository) Log(ctx context.Context, userID *string, event, ipAddress, deviceName, browser, operatingSystem string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_logs (user_id, event, ip_address, device_name, browser, operating_system)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, event, ipAddress, deviceName, browser, operatingSystem,
	)
	return err
}
