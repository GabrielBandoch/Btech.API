package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSubscriptionRepository(pool *pgxpool.Pool) domain.SubscriptionRepository {
	return &PostgresSubscriptionRepository{
		pool: pool,
	}
}

func (r *PostgresSubscriptionRepository) GetByOrganizationID(ctx context.Context, orgID string) (*domain.Subscription, error) {
	query := `SELECT id, organization_id, plan_id, status, starts_at, ends_at, trial_ends_at, canceled_at, created_at, updated_at 
	          FROM subscriptions WHERE organization_id = $1`
	
	var sub domain.Subscription
	err := r.pool.QueryRow(ctx, query, orgID).Scan(
		&sub.ID,
		&sub.OrganizationID,
		&sub.PlanID,
		&sub.Status,
		&sub.StartsAt,
		&sub.EndsAt,
		&sub.TrialEndsAt,
		&sub.CanceledAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("failed to get subscription by organization ID: %w", err)
	}
	return &sub, nil
}

func (r *PostgresSubscriptionRepository) GetByID(ctx context.Context, id string) (*domain.Subscription, error) {
	query := `SELECT id, organization_id, plan_id, status, starts_at, ends_at, trial_ends_at, canceled_at, created_at, updated_at 
	          FROM subscriptions WHERE id = $1`
	
	var sub domain.Subscription
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&sub.ID,
		&sub.OrganizationID,
		&sub.PlanID,
		&sub.Status,
		&sub.StartsAt,
		&sub.EndsAt,
		&sub.TrialEndsAt,
		&sub.CanceledAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("failed to get subscription by ID: %w", err)
	}
	return &sub, nil
}

func (r *PostgresSubscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) error {
	query := `INSERT INTO subscriptions (id, organization_id, plan_id, status, starts_at, ends_at, trial_ends_at, canceled_at, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.pool.Exec(ctx, query,
		sub.ID,
		sub.OrganizationID,
		sub.PlanID,
		sub.Status,
		sub.StartsAt,
		sub.EndsAt,
		sub.TrialEndsAt,
		sub.CanceledAt,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

func (r *PostgresSubscriptionRepository) Update(ctx context.Context, sub *domain.Subscription) error {
	query := `UPDATE subscriptions 
	          SET plan_id = $1, status = $2, starts_at = $3, ends_at = $4, trial_ends_at = $5, canceled_at = $6, updated_at = $7
	          WHERE id = $8`
	_, err := r.pool.Exec(ctx, query,
		sub.PlanID,
		sub.Status,
		sub.StartsAt,
		sub.EndsAt,
		sub.TrialEndsAt,
		sub.CanceledAt,
		sub.UpdatedAt,
		sub.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}

func (r *PostgresSubscriptionRepository) CreateEventHistory(ctx context.Context, event *domain.SubscriptionEventHistory) error {
	metadataBytes, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription event metadata: %w", err)
	}

	query := `INSERT INTO subscription_event_history (id, organization_id, subscription_id, event_type, from_status, to_status, metadata, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err = r.pool.Exec(ctx, query,
		event.ID,
		event.OrganizationID,
		event.SubscriptionID,
		event.EventType,
		event.FromStatus,
		event.ToStatus,
		metadataBytes,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create subscription event history: %w", err)
	}
	return nil
}

func (r *PostgresSubscriptionRepository) GetEventHistory(ctx context.Context, orgID string) ([]*domain.SubscriptionEventHistory, error) {
	query := `SELECT id, organization_id, subscription_id, event_type, from_status, to_status, metadata, created_at 
	          FROM subscription_event_history WHERE organization_id = $1 ORDER BY created_at DESC`
	
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription event history: %w", err)
	}
	defer rows.Close()

	var events []*domain.SubscriptionEventHistory
	for rows.Next() {
		var event domain.SubscriptionEventHistory
		var metadataBytes []byte
		err := rows.Scan(
			&event.ID,
			&event.OrganizationID,
			&event.SubscriptionID,
			&event.EventType,
			&event.FromStatus,
			&event.ToStatus,
			&metadataBytes,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription event row: %w", err)
		}

		if len(metadataBytes) > 0 {
			if err := json.Unmarshal(metadataBytes, &event.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal subscription event metadata: %w", err)
			}
		}
		events = append(events, &event)
	}
	return events, nil
}
