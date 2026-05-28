package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAuditLogRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAuditLogRepository instantiates a new PostgreSQL audit log repository.
func NewPostgresAuditLogRepository(pool *pgxpool.Pool) domain.AuditLogRepository {
	return &PostgresAuditLogRepository{
		pool: pool,
	}
}

func (r *PostgresAuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	metadataBytes, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal audit log metadata: %w", err)
	}

	query := `
		INSERT INTO audit_logs (id, actor_user_id, organization_id, action, entity_type, entity_id, metadata, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err = r.pool.Exec(ctx, query,
		log.ID,
		log.ActorUserID,
		log.OrganizationID,
		log.Action,
		log.EntityType,
		log.EntityID,
		metadataBytes,
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("database error creating audit log: %w", err)
	}

	return nil
}

func (r *PostgresAuditLogRepository) GetByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
		SELECT id, actor_user_id, organization_id, action, entity_type, entity_id, metadata, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("database error fetching audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var metadataBytes []byte
		err := rows.Scan(
			&l.ID,
			&l.ActorUserID,
			&l.OrganizationID,
			&l.Action,
			&l.EntityType,
			&l.EntityID,
			&metadataBytes,
			&l.IPAddress,
			&l.UserAgent,
			&l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log row: %w", err)
		}

		if len(metadataBytes) > 0 {
			if err := json.Unmarshal(metadataBytes, &l.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal audit log metadata: %w", err)
			}
		}

		logs = append(logs, &l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit log rows: %w", err)
	}

	return logs, nil
}
