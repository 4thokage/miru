package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/exp/slog"
)

var Pool *pgxpool.Pool
var logger = slog.Default()

func InitDB(ctx context.Context) error {
	connStr := getEnv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/manga_reader")

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	Pool = pool
	logger.Info("postgres connection pool initialized")
	return nil
}

func Close() {
	if Pool != nil {
		Pool.Close()
		logger.Info("postgres connection pool closed")
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
