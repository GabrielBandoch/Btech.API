package postgres

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresMaintenancePlanRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMaintenancePlanRepository(pool *pgxpool.Pool) domain.MaintenancePlanRepository {
	return &PostgresMaintenancePlanRepository{
		pool: pool,
	}
}

func (r *PostgresMaintenancePlanRepository) GetAll(ctx context.Context, orgID string, vehicleID string) ([]domain.MaintenancePlan, error) {
	query := `SELECT id, organization_id, vehicle_id, name, interval_km, interval_months, 
	                 last_maintenance_km, last_maintenance_date, next_due_km, next_due_date, 
	                 created_at, updated_at, deleted_at 
	          FROM maintenance_plans 
	          WHERE organization_id = $1 AND deleted_at IS NULL`

	var rows pgx.Rows
	var err error

	if vehicleID != "" {
		query += ` AND vehicle_id = $2 ORDER BY created_at DESC`
		rows, err = r.pool.Query(ctx, query, orgID, vehicleID)
	} else {
		query += ` ORDER BY created_at DESC`
		rows, err = r.pool.Query(ctx, query, orgID)
	}

	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var plans []domain.MaintenancePlan
	for rows.Next() {
		var p domain.MaintenancePlan
		err := rows.Scan(
			&p.ID,
			&p.OrganizationID,
			&p.VehicleID,
			&p.Name,
			&p.IntervalKM,
			&p.IntervalMonths,
			&p.LastMaintenanceKM,
			&p.LastMaintenanceDate,
			&p.NextDueKM,
			&p.NextDueDate,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan maintenance plan: %w", err)
		}
		plans = append(plans, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return plans, nil
}

func (r *PostgresMaintenancePlanRepository) GetByID(ctx context.Context, orgID string, id string) (domain.MaintenancePlan, error) {
	query := `SELECT id, organization_id, vehicle_id, name, interval_km, interval_months, 
	                 last_maintenance_km, last_maintenance_date, next_due_km, next_due_date, 
	                 created_at, updated_at, deleted_at 
	          FROM maintenance_plans 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var p domain.MaintenancePlan
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&p.ID,
		&p.OrganizationID,
		&p.VehicleID,
		&p.Name,
		&p.IntervalKM,
		&p.IntervalMonths,
		&p.LastMaintenanceKM,
		&p.LastMaintenanceDate,
		&p.NextDueKM,
		&p.NextDueDate,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.MaintenancePlan{}, fmt.Errorf("maintenance plan not found")
		}
		return domain.MaintenancePlan{}, fmt.Errorf("database query error: %w", err)
	}

	return p, nil
}

func (r *PostgresMaintenancePlanRepository) Create(ctx context.Context, orgID string, plan domain.MaintenancePlan) (domain.MaintenancePlan, error) {
	if plan.ID == "" {
		plan.ID = fmt.Sprintf("PLN-%03d", rand.Intn(1000)+10)
	}
	plan.OrganizationID = orgID
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = time.Now()
	}
	plan.UpdatedAt = plan.CreatedAt

	query := `INSERT INTO maintenance_plans (
	            id, organization_id, vehicle_id, name, interval_km, interval_months, 
	            last_maintenance_km, last_maintenance_date, next_due_km, next_due_date, 
	            created_at, updated_at, deleted_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.pool.Exec(ctx, query,
		plan.ID,
		plan.OrganizationID,
		plan.VehicleID,
		plan.Name,
		plan.IntervalKM,
		plan.IntervalMonths,
		plan.LastMaintenanceKM,
		plan.LastMaintenanceDate,
		plan.NextDueKM,
		plan.NextDueDate,
		plan.CreatedAt,
		plan.UpdatedAt,
		plan.DeletedAt,
	)

	if err != nil {
		return domain.MaintenancePlan{}, fmt.Errorf("failed to create maintenance plan in db: %w", err)
	}

	return plan, nil
}

func (r *PostgresMaintenancePlanRepository) Update(ctx context.Context, orgID string, id string, plan domain.MaintenancePlan) (domain.MaintenancePlan, error) {
	var existing domain.MaintenancePlan
	queryCheck := `SELECT id, organization_id, vehicle_id, name, interval_km, interval_months, 
	                      last_maintenance_km, last_maintenance_date, next_due_km, next_due_date, 
	                      created_at, updated_at, deleted_at 
	               FROM maintenance_plans 
	               WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	err := r.pool.QueryRow(ctx, queryCheck, orgID, id).Scan(
		&existing.ID,
		&existing.OrganizationID,
		&existing.VehicleID,
		&existing.Name,
		&existing.IntervalKM,
		&existing.IntervalMonths,
		&existing.LastMaintenanceKM,
		&existing.LastMaintenanceDate,
		&existing.NextDueKM,
		&existing.NextDueDate,
		&existing.CreatedAt,
		&existing.UpdatedAt,
		&existing.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.MaintenancePlan{}, fmt.Errorf("maintenance plan not found")
		}
		return domain.MaintenancePlan{}, fmt.Errorf("database query error: %w", err)
	}

	// Apply updates
	if plan.Name != "" {
		existing.Name = plan.Name
	}
	if plan.IntervalKM != nil {
		existing.IntervalKM = plan.IntervalKM
	}
	if plan.IntervalMonths != nil {
		existing.IntervalMonths = plan.IntervalMonths
	}
	if plan.LastMaintenanceKM != nil {
		existing.LastMaintenanceKM = plan.LastMaintenanceKM
	}
	if plan.LastMaintenanceDate != nil {
		existing.LastMaintenanceDate = plan.LastMaintenanceDate
	}
	if plan.NextDueKM != nil {
		existing.NextDueKM = plan.NextDueKM
	}
	if plan.NextDueDate != nil {
		existing.NextDueDate = plan.NextDueDate
	}
	existing.UpdatedAt = time.Now()

	updateQuery := `UPDATE maintenance_plans 
	                SET name = $1, interval_km = $2, interval_months = $3, 
	                    last_maintenance_km = $4, last_maintenance_date = $5, 
	                    next_due_km = $6, next_due_date = $7, updated_at = $8 
	                WHERE organization_id = $9 AND id = $10`

	_, err = r.pool.Exec(ctx, updateQuery,
		existing.Name,
		existing.IntervalKM,
		existing.IntervalMonths,
		existing.LastMaintenanceKM,
		existing.LastMaintenanceDate,
		existing.NextDueKM,
		existing.NextDueDate,
		existing.UpdatedAt,
		orgID,
		id,
	)

	if err != nil {
		return domain.MaintenancePlan{}, fmt.Errorf("failed to update maintenance plan in db: %w", err)
	}

	return existing, nil
}

func (r *PostgresMaintenancePlanRepository) Delete(ctx context.Context, orgID string, id string) error {
	query := `UPDATE maintenance_plans SET deleted_at = $1 WHERE organization_id = $2 AND id = $3 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, time.Now(), orgID, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete maintenance plan: %w", err)
	}
	return nil
}
