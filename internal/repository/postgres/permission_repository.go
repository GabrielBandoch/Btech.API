package postgres

import (
	"context"
	"fmt"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPermissionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPermissionRepository instantiates a new Postgres permission repository.
func NewPostgresPermissionRepository(pool *pgxpool.Pool) domain.PermissionRepository {
	return &PostgresPermissionRepository{
		pool: pool,
	}
}

func (r *PostgresPermissionRepository) GetPermissionsByRole(ctx context.Context, role string) ([]string, error) {
	query := `
		SELECT p.name 
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role = $1`

	rows, err := r.pool.Query(ctx, query, role)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan permission name: %w", err)
		}
		permissions = append(permissions, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating permission rows: %w", err)
	}

	return permissions, nil
}
