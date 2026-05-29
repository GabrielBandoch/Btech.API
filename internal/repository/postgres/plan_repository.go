package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPlanRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPlanRepository(pool *pgxpool.Pool) domain.PlanRepository {
	return &PostgresPlanRepository{
		pool: pool,
	}
}

func (r *PostgresPlanRepository) GetByID(ctx context.Context, id string) (*domain.Plan, error) {
	query := `SELECT id, code, name, description, monthly_price, yearly_price, is_active, created_at 
	          FROM plans WHERE id = $1`
	
	var plan domain.Plan
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.Description,
		&plan.MonthlyPrice,
		&plan.YearlyPrice,
		&plan.IsActive,
		&plan.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPlanNotFound
		}
		return nil, fmt.Errorf("failed to get plan by ID: %w", err)
	}
	return &plan, nil
}

func (r *PostgresPlanRepository) GetByCode(ctx context.Context, code string) (*domain.Plan, error) {
	query := `SELECT id, code, name, description, monthly_price, yearly_price, is_active, created_at 
	          FROM plans WHERE code = $1`
	
	var plan domain.Plan
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&plan.ID,
		&plan.Code,
		&plan.Name,
		&plan.Description,
		&plan.MonthlyPrice,
		&plan.YearlyPrice,
		&plan.IsActive,
		&plan.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPlanNotFound
		}
		return nil, fmt.Errorf("failed to get plan by code: %w", err)
	}
	return &plan, nil
}

func (r *PostgresPlanRepository) ListActive(ctx context.Context) ([]*domain.Plan, error) {
	query := `SELECT id, code, name, description, monthly_price, yearly_price, is_active, created_at 
	          FROM plans WHERE is_active = true ORDER BY monthly_price ASC`
	
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active plans: %w", err)
	}
	defer rows.Close()

	var plans []*domain.Plan
	for rows.Next() {
		var plan domain.Plan
		err := rows.Scan(
			&plan.ID,
			&plan.Code,
			&plan.Name,
			&plan.Description,
			&plan.MonthlyPrice,
			&plan.YearlyPrice,
			&plan.IsActive,
			&plan.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan row: %w", err)
		}
		plans = append(plans, &plan)
	}
	return plans, nil
}

func (r *PostgresPlanRepository) GetEntitlements(ctx context.Context, planID string) ([]*domain.PlanEntitlement, error) {
	query := `SELECT plan_id, key, value FROM plan_entitlements WHERE plan_id = $1`
	
	rows, err := r.pool.Query(ctx, query, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan entitlements: %w", err)
	}
	defer rows.Close()

	var ents []*domain.PlanEntitlement
	for rows.Next() {
		var ent domain.PlanEntitlement
		err := rows.Scan(
			&ent.PlanID,
			&ent.Key,
			&ent.Value,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan entitlement row: %w", err)
		}
		ents = append(ents, &ent)
	}
	return ents, nil
}

func (r *PostgresPlanRepository) Create(ctx context.Context, plan *domain.Plan) error {
	query := `INSERT INTO plans (id, code, name, description, monthly_price, yearly_price, is_active, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.pool.Exec(ctx, query,
		plan.ID,
		plan.Code,
		plan.Name,
		plan.Description,
		plan.MonthlyPrice,
		plan.YearlyPrice,
		plan.IsActive,
		plan.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create plan: %w", err)
	}
	return nil
}

func (r *PostgresPlanRepository) CreateEntitlement(ctx context.Context, ent *domain.PlanEntitlement) error {
	query := `INSERT INTO plan_entitlements (plan_id, key, value) VALUES ($1, $2, $3)
	          ON CONFLICT (plan_id, key) DO UPDATE SET value = EXCLUDED.value`
	_, err := r.pool.Exec(ctx, query, ent.PlanID, ent.Key, ent.Value)
	if err != nil {
		return fmt.Errorf("failed to create plan entitlement: %w", err)
	}
	return nil
}
