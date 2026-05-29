package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUsageCounterRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUsageCounterRepository(pool *pgxpool.Pool) domain.UsageCounterRepository {
	return &PostgresUsageCounterRepository{
		pool: pool,
	}
}

func (r *PostgresUsageCounterRepository) Get(ctx context.Context, orgID string, metricKey string, billingPeriod string) (*domain.UsageCounter, error) {
	query := `SELECT id, organization_id, metric_key, current_value, billing_period, created_at, updated_at 
	          FROM usage_counters 
	          WHERE organization_id = $1 AND metric_key = $2 AND billing_period = $3`
	
	var uc domain.UsageCounter
	err := r.pool.QueryRow(ctx, query, orgID, metricKey, billingPeriod).Scan(
		&uc.ID,
		&uc.OrganizationID,
		&uc.MetricKey,
		&uc.CurrentValue,
		&uc.BillingPeriod,
		&uc.CreatedAt,
		&uc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return a zero-value counter if it does not exist in the database yet
			return &domain.UsageCounter{
				OrganizationID: orgID,
				MetricKey:      metricKey,
				CurrentValue:   0,
				BillingPeriod:  billingPeriod,
			}, nil
		}
		return nil, fmt.Errorf("failed to get usage counter: %w", err)
	}
	return &uc, nil
}

func (r *PostgresUsageCounterRepository) Increment(ctx context.Context, orgID string, metricKey string, billingPeriod string, delta int) (*domain.UsageCounter, error) {
	newID := newUUID()
	query := `INSERT INTO usage_counters (id, organization_id, metric_key, current_value, billing_period, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	          ON CONFLICT (organization_id, metric_key, billing_period)
	          DO UPDATE SET current_value = usage_counters.current_value + EXCLUDED.current_value, updated_at = NOW()
	          RETURNING id, organization_id, metric_key, current_value, billing_period, created_at, updated_at`
	
	var uc domain.UsageCounter
	err := r.pool.QueryRow(ctx, query, newID, orgID, metricKey, delta, billingPeriod).Scan(
		&uc.ID,
		&uc.OrganizationID,
		&uc.MetricKey,
		&uc.CurrentValue,
		&uc.BillingPeriod,
		&uc.CreatedAt,
		&uc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to increment usage counter: %w", err)
	}
	return &uc, nil
}

func (r *PostgresUsageCounterRepository) Reset(ctx context.Context, orgID string, metricKey string, billingPeriod string) error {
	query := `INSERT INTO usage_counters (id, organization_id, metric_key, current_value, billing_period, created_at, updated_at)
	          VALUES ($1, $2, $3, 0, $4, NOW(), NOW())
	          ON CONFLICT (organization_id, metric_key, billing_period)
	          DO UPDATE SET current_value = 0, updated_at = NOW()`
	
	newID := newUUID()
	_, err := r.pool.Exec(ctx, query, newID, orgID, metricKey, billingPeriod)
	if err != nil {
		return fmt.Errorf("failed to reset usage counter: %w", err)
	}
	return nil
}

// newUUID generates a version 4 UUID.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
