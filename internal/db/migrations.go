package db

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"backend/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations runs all pending migrations located in the migrations directory.
func RunMigrations(cfg *config.Config) error {
	// Note: the golang-migrate pgx v5 driver registers under the "pgx5" URL scheme
	dsn := cfg.DB.URL
	if strings.HasPrefix(dsn, "postgres://") {
		dsn = strings.Replace(dsn, "postgres://", "pgx5://", 1)
	} else if strings.HasPrefix(dsn, "postgresql://") {
		dsn = strings.Replace(dsn, "postgresql://", "pgx5://", 1)
	}

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}
	defer m.Close()

	slog.Info("Running database migrations...")
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No new database migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	slog.Info("Database migrations completed successfully")
	return nil
}
