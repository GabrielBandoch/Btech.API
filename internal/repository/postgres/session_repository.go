package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserSessionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserSessionRepository instantiates a PostgreSQL session repository.
func NewPostgresUserSessionRepository(pool *pgxpool.Pool) domain.UserSessionRepository {
	return &PostgresUserSessionRepository{
		pool: pool,
	}
}

func (r *PostgresUserSessionRepository) GetByID(ctx context.Context, id string) (*domain.UserSession, error) {
	query := `
		SELECT id, user_id, organization_id, token_hash, token_version, user_agent, ip_address, is_revoked, expires_at, last_seen_at, created_at, updated_at
		FROM user_sessions
		WHERE id = $1`

	var s domain.UserSession
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.UserID,
		&s.OrganizationID,
		&s.TokenHash,
		&s.TokenVersion,
		&s.UserAgent,
		&s.IPAddress,
		&s.IsRevoked,
		&s.ExpiresAt,
		&s.LastSeenAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("database error getting session: %w", err)
	}

	return &s, nil
}

func (r *PostgresUserSessionRepository) Create(ctx context.Context, s *domain.UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, organization_id, token_hash, token_version, user_agent, ip_address, is_revoked, expires_at, last_seen_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		s.ID,
		s.UserID,
		s.OrganizationID,
		s.TokenHash,
		s.TokenVersion,
		s.UserAgent,
		s.IPAddress,
		s.IsRevoked,
		s.ExpiresAt,
		s.LastSeenAt,
		s.CreatedAt,
		s.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("database error creating session: %w", err)
	}

	return nil
}

func (r *PostgresUserSessionRepository) Update(ctx context.Context, s *domain.UserSession) error {
	query := `
		UPDATE user_sessions
		SET token_hash = $1, token_version = $2, user_agent = $3, ip_address = $4, is_revoked = $5, expires_at = $6, last_seen_at = $7, updated_at = $8
		WHERE id = $9`

	_, err := r.pool.Exec(ctx, query,
		s.TokenHash,
		s.TokenVersion,
		s.UserAgent,
		s.IPAddress,
		s.IsRevoked,
		s.ExpiresAt,
		s.LastSeenAt,
		s.UpdatedAt,
		s.ID,
	)

	if err != nil {
		return fmt.Errorf("database error updating session: %w", err)
	}

	return nil
}

func (r *PostgresUserSessionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM user_sessions WHERE id = $1`

	commandTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("database error deleting session: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrSessionNotFound
	}

	return nil
}

func (r *PostgresUserSessionRepository) ListByUserID(ctx context.Context, userID, orgID string) ([]*domain.UserSession, error) {
	query := `
		SELECT id, user_id, organization_id, token_hash, token_version, user_agent, ip_address, is_revoked, expires_at, last_seen_at, created_at, updated_at
		FROM user_sessions
		WHERE user_id = $1 AND organization_id = $2
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID, orgID)
	if err != nil {
		return nil, fmt.Errorf("database error listing sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*domain.UserSession
	for rows.Next() {
		var s domain.UserSession
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.OrganizationID,
			&s.TokenHash,
			&s.TokenVersion,
			&s.UserAgent,
			&s.IPAddress,
			&s.IsRevoked,
			&s.ExpiresAt,
			&s.LastSeenAt,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessions = append(sessions, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}

func (r *PostgresUserSessionRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	query := `
		UPDATE user_sessions
		SET is_revoked = true, updated_at = $1
		WHERE user_id = $2 AND is_revoked = false`

	_, err := r.pool.Exec(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("database error revoking sessions: %w", err)
	}

	return nil
}
