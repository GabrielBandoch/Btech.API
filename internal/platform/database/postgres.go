package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
}

// NewPostgresDB creates a pgxpool connection pool and returns it.
func NewPostgresDB(ctx context.Context, connStr string, logger *slog.Logger) (*PostgresDB, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = 15 * time.Minute
	config.MaxConnLifetime = 1 * time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	logger.Info("Successfully connected to PostgreSQL connection pool")

	return &PostgresDB{
		Pool: pool,
	}, nil
}

// Close gracefully closes the pgxpool connection.
func (db *PostgresDB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// RunMigrations runs goose migrations from the specified folder.
func RunMigrations(connStr string, migrationsDir string, logger *slog.Logger) error {
	logger.Info("Running database migrations", slog.String("dir", migrationsDir))

	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open sql connection for migrations: %w", err)
	}
	defer sqlDB.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		return fmt.Errorf("goose migration failed: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}
