package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fuelRepository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewFuelRepository(db *pgxpool.Pool, logger *slog.Logger) domain.FuelRepository {
	return &fuelRepository{db: db, logger: logger}
}

// --------------------------------------------------------------------------
// GetAll returns fuel records for the given org, optionally filtered.
// --------------------------------------------------------------------------
func (r *fuelRepository) GetAll(ctx context.Context, orgID string, filter domain.FuelFilter) ([]domain.FuelRecord, error) {
	query := `
		SELECT id, organization_id, vehicle_id, driver_id, date, liters, price_per_liter,
		       total_cost, odometer_reading, fuel_type, station_name, notes, is_anomaly,
		       created_at, updated_at, deleted_at
		FROM fuel_records
		WHERE organization_id = $1 AND deleted_at IS NULL`

	args := []interface{}{orgID}
	argIdx := 2

	if filter.VehicleID != "" {
		query += ` AND vehicle_id = $` + itoa(argIdx)
		args = append(args, filter.VehicleID)
		argIdx++
	}
	if filter.DriverID != "" {
		query += ` AND driver_id = $` + itoa(argIdx)
		args = append(args, filter.DriverID)
		argIdx++
	}
	if filter.FuelType != "" {
		query += ` AND fuel_type = $` + itoa(argIdx)
		args = append(args, filter.FuelType)
		argIdx++
	}
	if filter.StartDate != nil {
		query += ` AND date >= $` + itoa(argIdx)
		args = append(args, filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		query += ` AND date <= $` + itoa(argIdx)
		args = append(args, filter.EndDate)
		argIdx++
	}

	query += ` ORDER BY date DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFuelRows(rows)
}

// --------------------------------------------------------------------------
// GetByID returns a single fuel record scoped to org.
// --------------------------------------------------------------------------
func (r *fuelRepository) GetByID(ctx context.Context, orgID string, id string) (domain.FuelRecord, error) {
	query := `
		SELECT id, organization_id, vehicle_id, driver_id, date, liters, price_per_liter,
		       total_cost, odometer_reading, fuel_type, station_name, notes, is_anomaly,
		       created_at, updated_at, deleted_at
		FROM fuel_records
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL`

	row := r.db.QueryRow(ctx, query, id, orgID)
	rec, err := scanFuelRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.FuelRecord{}, domain.ErrFuelRecordNotFound
		}
		return domain.FuelRecord{}, err
	}
	return rec, nil
}

// --------------------------------------------------------------------------
// Create inserts a new fuel record.
// --------------------------------------------------------------------------
func (r *fuelRepository) Create(ctx context.Context, orgID string, rec domain.FuelRecord) (domain.FuelRecord, error) {
	query := `
		INSERT INTO fuel_records (
			id, organization_id, vehicle_id, driver_id, date, liters, price_per_liter,
			total_cost, odometer_reading, fuel_type, station_name, notes, is_anomaly,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, organization_id, vehicle_id, driver_id, date, liters, price_per_liter,
		          total_cost, odometer_reading, fuel_type, station_name, notes, is_anomaly,
		          created_at, updated_at, deleted_at`

	now := time.Now().UTC()
	row := r.db.QueryRow(ctx, query,
		rec.ID,
		orgID,
		rec.VehicleID,
		rec.DriverID,
		rec.Date,
		rec.Liters,
		rec.PricePerLiter,
		rec.TotalCost,
		rec.OdometerReading,
		rec.FuelType,
		rec.StationName,
		rec.Notes,
		rec.IsAnomaly,
		now,
		now,
	)

	created, err := scanFuelRow(row)
	if err != nil {
		return domain.FuelRecord{}, err
	}
	return created, nil
}

// --------------------------------------------------------------------------
// Update replaces fuel record fields. Always recomputes updated_at.
// --------------------------------------------------------------------------
func (r *fuelRepository) Update(ctx context.Context, orgID string, id string, rec domain.FuelRecord) (domain.FuelRecord, error) {
	query := `
		UPDATE fuel_records SET
			vehicle_id       = $1,
			driver_id        = $2,
			date             = $3,
			liters           = $4,
			price_per_liter  = $5,
			total_cost       = $6,
			odometer_reading = $7,
			fuel_type        = $8,
			station_name     = $9,
			notes            = $10,
			is_anomaly       = $11,
			updated_at       = $12
		WHERE id = $13 AND organization_id = $14 AND deleted_at IS NULL
		RETURNING id, organization_id, vehicle_id, driver_id, date, liters, price_per_liter,
		          total_cost, odometer_reading, fuel_type, station_name, notes, is_anomaly,
		          created_at, updated_at, deleted_at`

	row := r.db.QueryRow(ctx, query,
		rec.VehicleID,
		rec.DriverID,
		rec.Date,
		rec.Liters,
		rec.PricePerLiter,
		rec.TotalCost,
		rec.OdometerReading,
		rec.FuelType,
		rec.StationName,
		rec.Notes,
		rec.IsAnomaly,
		time.Now().UTC(),
		id,
		orgID,
	)

	updated, err := scanFuelRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.FuelRecord{}, domain.ErrFuelRecordNotFound
		}
		return domain.FuelRecord{}, err
	}
	return updated, nil
}

// --------------------------------------------------------------------------
// Delete performs a soft delete (sets deleted_at = NOW()).
// --------------------------------------------------------------------------
func (r *fuelRepository) Delete(ctx context.Context, orgID string, id string) error {
	query := `
		UPDATE fuel_records
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL`

	tag, err := r.db.Exec(ctx, query, id, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrFuelRecordNotFound
	}
	return nil
}

// --------------------------------------------------------------------------
// GetLastNByVehicle returns the N most recent records for anomaly detection.
// Results are ordered by date DESC (most recent first).
// --------------------------------------------------------------------------
func (r *fuelRepository) GetLastNByVehicle(ctx context.Context, orgID string, vehicleID string, n int) ([]domain.FuelRecord, error) {
	query := `
		SELECT id, organization_id, vehicle_id, driver_id, date, liters, price_per_liter,
		       total_cost, odometer_reading, fuel_type, station_name, notes, is_anomaly,
		       created_at, updated_at, deleted_at
		FROM fuel_records
		WHERE organization_id = $1 AND vehicle_id = $2 AND deleted_at IS NULL
		ORDER BY date DESC
		LIMIT $3`

	rows, err := r.db.Query(ctx, query, orgID, vehicleID, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFuelRows(rows)
}

// --------------------------------------------------------------------------
// GetEfficiencyReport computes per-vehicle aggregations for the report page.
// km/L and cost/km are 0 when a vehicle has fewer than 2 records (no delta km).
// --------------------------------------------------------------------------
func (r *fuelRepository) GetEfficiencyReport(ctx context.Context, orgID string, filter domain.FuelFilter) ([]domain.FuelEfficiencyReport, error) {
	query := `
		SELECT
			vehicle_id,
			SUM(liters)                                                 AS total_liters,
			SUM(total_cost)                                             AS total_cost,
			COALESCE(MAX(odometer_reading) - MIN(odometer_reading), 0) AS total_km,
			COUNT(*)                                                    AS record_count
		FROM fuel_records
		WHERE organization_id = $1 AND deleted_at IS NULL`

	args := []interface{}{orgID}
	argIdx := 2

	if filter.VehicleID != "" {
		query += ` AND vehicle_id = $` + itoa(argIdx)
		args = append(args, filter.VehicleID)
		argIdx++
	}
	if filter.StartDate != nil {
		query += ` AND date >= $` + itoa(argIdx)
		args = append(args, filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		query += ` AND date <= $` + itoa(argIdx)
		args = append(args, filter.EndDate)
		argIdx++
	}

	query += ` GROUP BY vehicle_id ORDER BY total_cost DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []domain.FuelEfficiencyReport
	for rows.Next() {
		var rep domain.FuelEfficiencyReport
		var totalKm int
		if err := rows.Scan(
			&rep.VehicleID,
			&rep.TotalLiters,
			&rep.TotalCost,
			&totalKm,
			&rep.RecordCount,
		); err != nil {
			return nil, err
		}
		rep.TotalKm = totalKm

		// Only compute efficiency ratios when we have km delta (≥ 2 records)
		if totalKm > 0 && rep.TotalLiters > 0 {
			rep.AvgEfficiencyKmL = float64(totalKm) / rep.TotalLiters
		}
		if totalKm > 0 && rep.TotalCost > 0 {
			rep.AvgCostPerKm = rep.TotalCost / float64(totalKm)
		}

		reports = append(reports, rep)
	}

	if reports == nil {
		reports = []domain.FuelEfficiencyReport{}
	}
	return reports, nil
}

// --------------------------------------------------------------------------
// GetDashboardStats computes all KPIs for the dashboard endpoint.
// --------------------------------------------------------------------------
func (r *fuelRepository) GetDashboardStats(ctx context.Context, orgID string) (domain.FuelDashboard, error) {
	// Current month boundaries
	now := time.Now().UTC()
	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
	lastMonthEnd := thisMonthStart.Add(-time.Nanosecond)

	// --- This month aggregates ---
	thisMonthQuery := `
		SELECT
			COUNT(*)                                                 AS record_count,
			COALESCE(SUM(total_cost), 0)                             AS total_cost,
			COALESCE(SUM(liters), 0)                                 AS total_liters,
			COALESCE(MAX(odometer_reading) - MIN(odometer_reading), 0) AS total_km,
			COALESCE(SUM(CASE WHEN is_anomaly THEN 1 ELSE 0 END), 0) AS anomaly_count
		FROM fuel_records
		WHERE organization_id = $1 AND deleted_at IS NULL
		  AND date >= $2 AND date <= $3`

	var dashboard domain.FuelDashboard
	var totalKmThisMonth int

	err := r.db.QueryRow(ctx, thisMonthQuery, orgID, thisMonthStart, now).Scan(
		&dashboard.TotalRecordsThisMonth,
		&dashboard.TotalCostThisMonth,
		&dashboard.TotalLitersThisMonth,
		&totalKmThisMonth,
		&dashboard.AnomalyCount,
	)
	if err != nil {
		return domain.FuelDashboard{}, err
	}

	// Compute fleet avg efficiency (km/L) for this month
	if totalKmThisMonth > 0 && dashboard.TotalLitersThisMonth > 0 {
		dashboard.AvgEfficiencyKmL = float64(totalKmThisMonth) / dashboard.TotalLitersThisMonth
	}

	// Compute average cost per liter for this month
	if dashboard.TotalLitersThisMonth > 0 {
		dashboard.AvgCostPerLiter = dashboard.TotalCostThisMonth / dashboard.TotalLitersThisMonth
	}

	// --- Last month cost (for trend delta) ---
	lastMonthQuery := `
		SELECT COALESCE(SUM(total_cost), 0)
		FROM fuel_records
		WHERE organization_id = $1 AND deleted_at IS NULL
		  AND date >= $2 AND date <= $3`

	var lastMonthCost float64
	err = r.db.QueryRow(ctx, lastMonthQuery, orgID, lastMonthStart, lastMonthEnd).Scan(&lastMonthCost)
	if err != nil {
		return domain.FuelDashboard{}, err
	}

	if lastMonthCost > 0 {
		dashboard.CostVsLastMonth = ((dashboard.TotalCostThisMonth - lastMonthCost) / lastMonthCost) * 100
	}

	// --- Top consuming vehicle this month ---
	topVehicleQuery := `
		SELECT vehicle_id
		FROM fuel_records
		WHERE organization_id = $1 AND deleted_at IS NULL
		  AND date >= $2 AND date <= $3
		GROUP BY vehicle_id
		ORDER BY SUM(total_cost) DESC
		LIMIT 1`

	var topVehicleID string
	err = r.db.QueryRow(ctx, topVehicleQuery, orgID, thisMonthStart, now).Scan(&topVehicleID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return domain.FuelDashboard{}, err
	}
	if topVehicleID != "" {
		dashboard.TopConsumingVehicleID = &topVehicleID
	}

	return dashboard, nil
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// scanFuelRow scans a single pgx row into a FuelRecord.
func scanFuelRow(row pgx.Row) (domain.FuelRecord, error) {
	var rec domain.FuelRecord
	err := row.Scan(
		&rec.ID,
		&rec.OrganizationID,
		&rec.VehicleID,
		&rec.DriverID,
		&rec.Date,
		&rec.Liters,
		&rec.PricePerLiter,
		&rec.TotalCost,
		&rec.OdometerReading,
		&rec.FuelType,
		&rec.StationName,
		&rec.Notes,
		&rec.IsAnomaly,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&rec.DeletedAt,
	)
	return rec, err
}

// scanFuelRows scans multiple pgx rows into a slice of FuelRecord.
func scanFuelRows(rows pgx.Rows) ([]domain.FuelRecord, error) {
	var records []domain.FuelRecord
	for rows.Next() {
		var rec domain.FuelRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.OrganizationID,
			&rec.VehicleID,
			&rec.DriverID,
			&rec.Date,
			&rec.Liters,
			&rec.PricePerLiter,
			&rec.TotalCost,
			&rec.OdometerReading,
			&rec.FuelType,
			&rec.StationName,
			&rec.Notes,
			&rec.IsAnomaly,
			&rec.CreatedAt,
			&rec.UpdatedAt,
			&rec.DeletedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if records == nil {
		records = []domain.FuelRecord{}
	}
	return records, nil
}

// itoa converts an int to a string (used for building dynamic SQL parameter placeholders).
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
