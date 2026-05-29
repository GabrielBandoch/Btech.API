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

type PostgresMaintenanceAlertRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMaintenanceAlertRepository(pool *pgxpool.Pool) domain.MaintenanceAlertRepository {
	return &PostgresMaintenanceAlertRepository{
		pool: pool,
	}
}

func (r *PostgresMaintenanceAlertRepository) GetAll(ctx context.Context, orgID string, status string) ([]domain.MaintenanceAlert, error) {
	query := `SELECT id, organization_id, vehicle_id, maintenance_plan_id, type, title, message, status, created_at, updated_at, deleted_at 
	          FROM maintenance_alerts 
	          WHERE organization_id = $1 AND deleted_at IS NULL`

	var rows pgx.Rows
	var err error

	if status != "" {
		query += ` AND status = $2 ORDER BY created_at DESC`
		rows, err = r.pool.Query(ctx, query, orgID, status)
	} else {
		query += ` ORDER BY created_at DESC`
		rows, err = r.pool.Query(ctx, query, orgID)
	}

	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var alerts []domain.MaintenanceAlert
	for rows.Next() {
		var a domain.MaintenanceAlert
		err := rows.Scan(
			&a.ID,
			&a.OrganizationID,
			&a.VehicleID,
			&a.MaintenancePlanID,
			&a.Type,
			&a.Title,
			&a.Message,
			&a.Status,
			&a.CreatedAt,
			&a.UpdatedAt,
			&a.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan maintenance alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return alerts, nil
}

func (r *PostgresMaintenanceAlertRepository) GetByID(ctx context.Context, orgID string, id string) (domain.MaintenanceAlert, error) {
	query := `SELECT id, organization_id, vehicle_id, maintenance_plan_id, type, title, message, status, created_at, updated_at, deleted_at 
	          FROM maintenance_alerts 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var a domain.MaintenanceAlert
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&a.ID,
		&a.OrganizationID,
		&a.VehicleID,
		&a.MaintenancePlanID,
		&a.Type,
		&a.Title,
		&a.Message,
		&a.Status,
		&a.CreatedAt,
		&a.UpdatedAt,
		&a.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.MaintenanceAlert{}, fmt.Errorf("maintenance alert not found")
		}
		return domain.MaintenanceAlert{}, fmt.Errorf("database query error: %w", err)
	}

	return a, nil
}

func (r *PostgresMaintenanceAlertRepository) Create(ctx context.Context, orgID string, alert domain.MaintenanceAlert) (domain.MaintenanceAlert, error) {
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("ALT-%03d", rand.Intn(1000)+10)
	}
	alert.OrganizationID = orgID
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now()
	}
	alert.UpdatedAt = alert.CreatedAt

	if alert.Status == "" {
		alert.Status = domain.MaintenanceAlertStatusActive
	}

	query := `INSERT INTO maintenance_alerts (
	            id, organization_id, vehicle_id, maintenance_plan_id, type, title, message, status, created_at, updated_at, deleted_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.pool.Exec(ctx, query,
		alert.ID,
		alert.OrganizationID,
		alert.VehicleID,
		alert.MaintenancePlanID,
		alert.Type,
		alert.Title,
		alert.Message,
		alert.Status,
		alert.CreatedAt,
		alert.UpdatedAt,
		alert.DeletedAt,
	)

	if err != nil {
		return domain.MaintenanceAlert{}, fmt.Errorf("failed to create alert in db: %w", err)
	}

	return alert, nil
}

func (r *PostgresMaintenanceAlertRepository) UpdateStatus(ctx context.Context, orgID string, id string, status string) error {
	query := `UPDATE maintenance_alerts SET status = $1, updated_at = $2 WHERE organization_id = $3 AND id = $4 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, status, time.Now(), orgID, id)
	if err != nil {
		return fmt.Errorf("failed to update alert status: %w", err)
	}
	return nil
}

func (r *PostgresMaintenanceAlertRepository) ResolveByPlanID(ctx context.Context, orgID string, planID string) error {
	query := `UPDATE maintenance_alerts SET status = $1, updated_at = $2 WHERE organization_id = $3 AND maintenance_plan_id = $4 AND status = $5 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, domain.MaintenanceAlertStatusResolved, time.Now(), orgID, planID, domain.MaintenanceAlertStatusActive)
	if err != nil {
		return fmt.Errorf("failed to resolve active alerts by plan id: %w", err)
	}
	return nil
}

func (r *PostgresMaintenanceAlertRepository) GetActiveByPlanID(ctx context.Context, orgID string, planID string) ([]domain.MaintenanceAlert, error) {
	query := `SELECT id, organization_id, vehicle_id, maintenance_plan_id, type, title, message, status, created_at, updated_at, deleted_at 
	          FROM maintenance_alerts 
	          WHERE organization_id = $1 AND maintenance_plan_id = $2 AND status = $3 AND deleted_at IS NULL`

	rows, err := r.pool.Query(ctx, query, orgID, planID, domain.MaintenanceAlertStatusActive)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var alerts []domain.MaintenanceAlert
	for rows.Next() {
		var a domain.MaintenanceAlert
		err := rows.Scan(
			&a.ID,
			&a.OrganizationID,
			&a.VehicleID,
			&a.MaintenancePlanID,
			&a.Type,
			&a.Title,
			&a.Message,
			&a.Status,
			&a.CreatedAt,
			&a.UpdatedAt,
			&a.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	return alerts, nil
}
