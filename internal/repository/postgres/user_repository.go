package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository instantiates a PostgreSQL user repository.
func NewPostgresUserRepository(pool *pgxpool.Pool) domain.UserRepository {
	return &PostgresUserRepository{
		pool: pool,
	}
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, name, email, password_hash, role, created_at, updated_at FROM users WHERE id = $1`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	query := `SELECT id, name, email, password_hash, role, created_at, updated_at FROM users WHERE LOWER(email) = $1`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, normalizedEmail).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return &user, nil
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))

	query := `INSERT INTO users (id, name, email, password_hash, role, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return domain.ErrEmailAlreadyExists
			}
		}
		return fmt.Errorf("database error: %w", err)
	}

	return nil
}
