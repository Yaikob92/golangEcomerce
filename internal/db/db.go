package db

import (
	"context"
	"fmt"
	"time"

	"backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// InitPool creates and returns a pgx connection pool based on Config.
func InitPool(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := cfg.DB.URL

	// Configure the pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string DSN: %w", err)
	}

	// Pool settings tuned for Neon serverless (free tier allows ~10 connections)
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	// Establish connection pool with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify the database connection is alive
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
