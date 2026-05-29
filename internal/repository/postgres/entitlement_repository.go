package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresEntitlementRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEntitlementRepository(pool *pgxpool.Pool) domain.EntitlementRepository {
	return &PostgresEntitlementRepository{
		pool: pool,
	}
}

func (r *PostgresEntitlementRepository) GetActiveOverride(ctx context.Context, orgID string, key string) (*domain.OrganizationEntitlementOverride, error) {
	query := `SELECT id, organization_id, key, value, expires_at, created_at 
	          FROM organization_entitlement_overrides 
	          WHERE organization_id = $1 AND key = $2 AND (expires_at IS NULL OR expires_at > NOW())`
	
	var override domain.OrganizationEntitlementOverride
	err := r.pool.QueryRow(ctx, query, orgID, key).Scan(
		&override.ID,
		&override.OrganizationID,
		&override.Key,
		&override.Value,
		&override.ExpiresAt,
		&override.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Return nil, nil when no override is found, which is a expected normal condition
		}
		return nil, fmt.Errorf("failed to get active override: %w", err)
	}
	return &override, nil
}

func (r *PostgresEntitlementRepository) ListActiveOverrides(ctx context.Context, orgID string) ([]*domain.OrganizationEntitlementOverride, error) {
	query := `SELECT id, organization_id, key, value, expires_at, created_at 
	          FROM organization_entitlement_overrides 
	          WHERE organization_id = $1 AND (expires_at IS NULL OR expires_at > NOW())`
	
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active overrides: %w", err)
	}
	defer rows.Close()

	var overrides []*domain.OrganizationEntitlementOverride
	for rows.Next() {
		var override domain.OrganizationEntitlementOverride
		err := rows.Scan(
			&override.ID,
			&override.OrganizationID,
			&override.Key,
			&override.Value,
			&override.ExpiresAt,
			&override.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan override row: %w", err)
		}
		overrides = append(overrides, &override)
	}
	return overrides, nil
}

func (r *PostgresEntitlementRepository) SetOverride(ctx context.Context, override *domain.OrganizationEntitlementOverride) error {
	// Wait, we also want to upsert on (organization_id, key) if a unique constraint exists, but since we didn't add a unique constraint on (org_id, key) in the migration for overrides, we can delete the old one first, or we can look up if it exists by key and org_id and delete/update.
	// To be extremely robust, let's delete any existing override for the same org and key first, then insert!
	// This avoids duplicate overrides for the same key.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	deleteQuery := `DELETE FROM organization_entitlement_overrides WHERE organization_id = $1 AND key = $2`
	_, err = tx.Exec(ctx, deleteQuery, override.OrganizationID, override.Key)
	if err != nil {
		return fmt.Errorf("failed to delete existing override: %w", err)
	}

	insertQuery := `INSERT INTO organization_entitlement_overrides (id, organization_id, key, value, expires_at, created_at)
	                VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.Exec(ctx, insertQuery,
		override.ID,
		override.OrganizationID,
		override.Key,
		override.Value,
		override.ExpiresAt,
		override.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert override: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit override transaction: %w", err)
	}

	return nil
}

func (r *PostgresEntitlementRepository) DeleteOverride(ctx context.Context, orgID string, key string) error {
	query := `DELETE FROM organization_entitlement_overrides WHERE organization_id = $1 AND key = $2`
	_, err := r.pool.Exec(ctx, query, orgID, key)
	if err != nil {
		return fmt.Errorf("failed to delete override: %w", err)
	}
	return nil
}
