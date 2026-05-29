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

type PostgresMaintenanceRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMaintenanceRepository(pool *pgxpool.Pool) domain.MaintenanceRepository {
	return &PostgresMaintenanceRepository{
		pool: pool,
	}
}

func (r *PostgresMaintenanceRepository) GetAll(ctx context.Context, orgID string, filter domain.MaintenanceFilter) ([]domain.Maintenance, error) {
	query := `SELECT id, organization_id, vehicle_id, maintenance_plan_id, supplier_id, 
	                 type, priority, status, date, odometer_at_service, downtime_hours, cost, description, attachments, 
	                 created_at, updated_at, deleted_at 
	          FROM maintenances 
	          WHERE organization_id = $1 AND deleted_at IS NULL`

	args := []interface{}{orgID}
	placeholderIdx := 2

	if filter.VehicleID != "" {
		query += fmt.Sprintf(" AND vehicle_id = $%d", placeholderIdx)
		args = append(args, filter.VehicleID)
		placeholderIdx++
	}
	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", placeholderIdx)
		args = append(args, filter.Type)
		placeholderIdx++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", placeholderIdx)
		args = append(args, filter.Status)
		placeholderIdx++
	}
	if filter.SupplierID != "" {
		query += fmt.Sprintf(" AND supplier_id = $%d", placeholderIdx)
		args = append(args, filter.SupplierID)
		placeholderIdx++
	}
	if filter.Priority != "" {
		query += fmt.Sprintf(" AND priority = $%d", placeholderIdx)
		args = append(args, filter.Priority)
		placeholderIdx++
	}
	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND date >= $%d", placeholderIdx)
		args = append(args, *filter.StartDate)
		placeholderIdx++
	}
	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND date <= $%d", placeholderIdx)
		args = append(args, *filter.EndDate)
		placeholderIdx++
	}

	query += " ORDER BY date DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var records []domain.Maintenance
	for rows.Next() {
		var m domain.Maintenance
		var planID *string
		var supplierID *string
		err := rows.Scan(
			&m.ID,
			&m.OrganizationID,
			&m.VehicleID,
			&planID,
			&supplierID,
			&m.Type,
			&m.Priority,
			&m.Status,
			&m.Date,
			&m.OdometerAtService,
			&m.DowntimeHours,
			&m.Cost,
			&m.Description,
			&m.Attachments,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan maintenance: %w", err)
		}
		if planID != nil {
			m.MaintenancePlanID = planID
		}
		if supplierID != nil {
			m.SupplierID = supplierID
		}
		if m.Attachments == nil {
			m.Attachments = []string{}
		}
		records = append(records, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return records, nil
}

func (r *PostgresMaintenanceRepository) GetByID(ctx context.Context, orgID string, id string) (domain.Maintenance, error) {
	query := `SELECT id, organization_id, vehicle_id, maintenance_plan_id, supplier_id, 
	                 type, priority, status, date, odometer_at_service, downtime_hours, cost, description, attachments, 
	                 created_at, updated_at, deleted_at 
	          FROM maintenances 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var m domain.Maintenance
	var planID *string
	var supplierID *string
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&m.ID,
		&m.OrganizationID,
		&m.VehicleID,
		&planID,
		&supplierID,
		&m.Type,
		&m.Priority,
		&m.Status,
		&m.Date,
		&m.OdometerAtService,
		&m.DowntimeHours,
		&m.Cost,
		&m.Description,
		&m.Attachments,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Maintenance{}, fmt.Errorf("maintenance record not found")
		}
		return domain.Maintenance{}, fmt.Errorf("database query error: %w", err)
	}

	if planID != nil {
		m.MaintenancePlanID = planID
	}
	if supplierID != nil {
		m.SupplierID = supplierID
	}
	if m.Attachments == nil {
		m.Attachments = []string{}
	}

	return m, nil
}

func (r *PostgresMaintenanceRepository) Create(ctx context.Context, orgID string, m domain.Maintenance) (domain.Maintenance, error) {
	if m.ID == "" {
		m.ID = fmt.Sprintf("MNT-%03d", rand.Intn(1000)+10)
	}
	m.OrganizationID = orgID
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}
	m.UpdatedAt = m.CreatedAt

	if m.Attachments == nil {
		m.Attachments = []string{}
	}

	query := `INSERT INTO maintenances (
	            id, organization_id, vehicle_id, maintenance_plan_id, supplier_id, 
	            type, priority, status, date, odometer_at_service, downtime_hours, cost, description, attachments, 
	            created_at, updated_at, deleted_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	var planIDVal, supplierIDVal interface{}
	if m.MaintenancePlanID != nil && *m.MaintenancePlanID != "" {
		planIDVal = *m.MaintenancePlanID
	}
	if m.SupplierID != nil && *m.SupplierID != "" {
		supplierIDVal = *m.SupplierID
	}

	_, err := r.pool.Exec(ctx, query,
		m.ID,
		m.OrganizationID,
		m.VehicleID,
		planIDVal,
		supplierIDVal,
		m.Type,
		m.Priority,
		m.Status,
		m.Date,
		m.OdometerAtService,
		m.DowntimeHours,
		m.Cost,
		m.Description,
		m.Attachments,
		m.CreatedAt,
		m.UpdatedAt,
		m.DeletedAt,
	)

	if err != nil {
		return domain.Maintenance{}, fmt.Errorf("failed to create maintenance record in db: %w", err)
	}

	return m, nil
}

func (r *PostgresMaintenanceRepository) Update(ctx context.Context, orgID string, id string, m domain.Maintenance) (domain.Maintenance, error) {
	var existing domain.Maintenance
	var planID *string
	var supplierID *string

	queryCheck := `SELECT id, organization_id, vehicle_id, maintenance_plan_id, supplier_id, 
	                      type, priority, status, date, odometer_at_service, downtime_hours, cost, description, attachments, 
	                      created_at, updated_at, deleted_at 
	               FROM  maintenances 
	               WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	err := r.pool.QueryRow(ctx, queryCheck, orgID, id).Scan(
		&existing.ID,
		&existing.OrganizationID,
		&existing.VehicleID,
		&planID,
		&supplierID,
		&existing.Type,
		&existing.Priority,
		&existing.Status,
		&existing.Date,
		&existing.OdometerAtService,
		&existing.DowntimeHours,
		&existing.Cost,
		&existing.Description,
		&existing.Attachments,
		&existing.CreatedAt,
		&existing.UpdatedAt,
		&existing.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Maintenance{}, fmt.Errorf("maintenance record not found")
		}
		return domain.Maintenance{}, fmt.Errorf("database query error: %w", err)
	}

	if planID != nil {
		existing.MaintenancePlanID = planID
	}
	if supplierID != nil {
		existing.SupplierID = supplierID
	}

	// Apply updates
	if m.VehicleID != "" {
		existing.VehicleID = m.VehicleID
	}
	if m.MaintenancePlanID != nil {
		if *m.MaintenancePlanID == "" {
			existing.MaintenancePlanID = nil
		} else {
			existing.MaintenancePlanID = m.MaintenancePlanID
		}
	}
	if m.SupplierID != nil {
		if *m.SupplierID == "" {
			existing.SupplierID = nil
		} else {
			existing.SupplierID = m.SupplierID
		}
	}
	if m.Type != "" {
		existing.Type = m.Type
	}
	if m.Priority != "" {
		existing.Priority = m.Priority
	}
	if m.Status != "" {
		existing.Status = m.Status
	}
	if !m.Date.IsZero() {
		existing.Date = m.Date
	}
	if m.OdometerAtService != 0 {
		existing.OdometerAtService = m.OdometerAtService
	}
	if m.DowntimeHours != 0 {
		existing.DowntimeHours = m.DowntimeHours
	}
	if m.Cost != 0 {
		existing.Cost = m.Cost
	}
	if m.Description != "" {
		existing.Description = m.Description
	}
	if m.Attachments != nil {
		existing.Attachments = m.Attachments
	}
	existing.UpdatedAt = time.Now()

	updateQuery := `UPDATE maintenances 
	                SET vehicle_id = $1, maintenance_plan_id = $2, supplier_id = $3, 
	                    type = $4, priority = $5, status = $6, date = $7, odometer_at_service = $8, 
	                    downtime_hours = $9, cost = $10, description = $11, attachments = $12, updated_at = $13 
	                WHERE organization_id = $14 AND id = $15`

	var updatePlanID, updateSupplierID interface{}
	if existing.MaintenancePlanID != nil {
		updatePlanID = *existing.MaintenancePlanID
	}
	if existing.SupplierID != nil {
		updateSupplierID = *existing.SupplierID
	}

	_, err = r.pool.Exec(ctx, updateQuery,
		existing.VehicleID,
		updatePlanID,
		updateSupplierID,
		existing.Type,
		existing.Priority,
		existing.Status,
		existing.Date,
		existing.OdometerAtService,
		existing.DowntimeHours,
		existing.Cost,
		existing.Description,
		existing.Attachments,
		existing.UpdatedAt,
		orgID,
		id,
	)

	if err != nil {
		return domain.Maintenance{}, fmt.Errorf("failed to update maintenance in db: %w", err)
	}

	return existing, nil
}

func (r *PostgresMaintenanceRepository) Delete(ctx context.Context, orgID string, id string) error {
	query := `UPDATE maintenances SET deleted_at = $1 WHERE organization_id = $2 AND id = $3 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, time.Now(), orgID, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete maintenance: %w", err)
	}
	return nil
}

func (r *PostgresMaintenanceRepository) GetCostReport(ctx context.Context, orgID string, filter domain.MaintenanceFilter) (map[string]interface{}, error) {
	// Base query parts
	baseWhere := "WHERE organization_id = $1 AND status = 'completed' AND deleted_at IS NULL"
	args := []interface{}{orgID}
	placeholderIdx := 2

	if filter.VehicleID != "" {
		baseWhere += fmt.Sprintf(" AND vehicle_id = $%d", placeholderIdx)
		args = append(args, filter.VehicleID)
		placeholderIdx++
	}
	if filter.SupplierID != "" {
		baseWhere += fmt.Sprintf(" AND supplier_id = $%d", placeholderIdx)
		args = append(args, filter.SupplierID)
		placeholderIdx++
	}
	if filter.StartDate != nil {
		baseWhere += fmt.Sprintf(" AND date >= $%d", placeholderIdx)
		args = append(args, *filter.StartDate)
		placeholderIdx++
	}
	if filter.EndDate != nil {
		baseWhere += fmt.Sprintf(" AND date <= $%d", placeholderIdx)
		args = append(args, *filter.EndDate)
		placeholderIdx++
	}

	// 1. Total Cost
	totalCostQuery := fmt.Sprintf("SELECT COALESCE(SUM(cost), 0) FROM maintenances %s", baseWhere)
	var totalCost float64
	err := r.pool.QueryRow(ctx, totalCostQuery, args...).Scan(&totalCost)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate total cost: %w", err)
	}

	// 2. Cost by Vehicle
	vehicleQuery := fmt.Sprintf(`SELECT vehicle_id, COALESCE(SUM(cost), 0) as total 
	                              FROM maintenances 
	                              %s 
	                              GROUP BY vehicle_id 
	                              ORDER BY total DESC`, baseWhere)
	vehicleRows, err := r.pool.Query(ctx, vehicleQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cost by vehicle: %w", err)
	}
	defer vehicleRows.Close()

	byVehicle := []map[string]interface{}{}
	for vehicleRows.Next() {
		var vID string
		var total float64
		if err := vehicleRows.Scan(&vID, &total); err == nil {
			byVehicle = append(byVehicle, map[string]interface{}{
				"vehicleId": vID,
				"total":     total,
			})
		}
	}

	// 3. Cost by Supplier
	supplierQuery := fmt.Sprintf(`SELECT COALESCE(supplier_id, 'Sem Fornecedor'), COALESCE(SUM(cost), 0) as total 
	                               FROM maintenances 
	                               %s 
	                               GROUP BY supplier_id 
	                               ORDER BY total DESC`, baseWhere)
	supplierRows, err := r.pool.Query(ctx, supplierQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cost by supplier: %w", err)
	}
	defer supplierRows.Close()

	bySupplier := []map[string]interface{}{}
	for supplierRows.Next() {
		var sID string
		var total float64
		if err := supplierRows.Scan(&sID, &total); err == nil {
			bySupplier = append(bySupplier, map[string]interface{}{
				"supplierId": sID,
				"total":      total,
			})
		}
	}

	// 4. Cost by Type
	typeQuery := fmt.Sprintf(`SELECT type, COALESCE(SUM(cost), 0) as total 
	                           FROM maintenances 
	                           %s 
	                           GROUP BY type 
	                           ORDER BY total DESC`, baseWhere)
	typeRows, err := r.pool.Query(ctx, typeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cost by type: %w", err)
	}
	defer typeRows.Close()

	byType := []map[string]interface{}{}
	for typeRows.Next() {
		var mType string
		var total float64
		if err := typeRows.Scan(&mType, &total); err == nil {
			byType = append(byType, map[string]interface{}{
				"type":  mType,
				"total": total,
			})
		}
	}

	// 5. Cost by Period (Monthly)
	periodQuery := fmt.Sprintf(`SELECT TO_CHAR(date, 'YYYY-MM') as month, COALESCE(SUM(cost), 0) as total 
	                             FROM maintenances 
	                             %s 
	                             GROUP BY TO_CHAR(date, 'YYYY-MM') 
	                             ORDER BY month ASC`, baseWhere)
	periodRows, err := r.pool.Query(ctx, periodQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cost by period: %w", err)
	}
	defer periodRows.Close()

	byPeriod := []map[string]interface{}{}
	for periodRows.Next() {
		var month string
		var total float64
		if err := periodRows.Scan(&month, &total); err == nil {
			byPeriod = append(byPeriod, map[string]interface{}{
				"period": month,
				"total":  total,
			})
		}
	}

	return map[string]interface{}{
		"totalCost":  totalCost,
		"byVehicle":  byVehicle,
		"bySupplier": bySupplier,
		"byType":      byType,
		"byPeriod":    byPeriod,
	}, nil
}
