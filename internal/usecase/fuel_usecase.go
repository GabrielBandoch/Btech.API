package usecase

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/google/uuid"
)

// anomalyThreshold is the percentage deviation from recent baseline that triggers an anomaly flag.
// Future: per-organization configurable via fuel_anomaly_threshold setting.
const anomalyThreshold = 0.30

// anomalyMinRecords is the minimum number of prior records needed before anomaly detection fires.
const anomalyMinRecords = 3

// editGracePeriod is the window within which operators may edit a fuel record.
const editGracePeriod = 24 * time.Hour

// validFuelTypes is the set of accepted fuel type values.
var validFuelTypes = map[string]struct{}{
	domain.FuelTypeGasoline: {},
	domain.FuelTypeEthanol:  {},
	domain.FuelTypeDiesel:   {},
	domain.FuelTypeGNV:      {},
	domain.FuelTypeElectric: {},
}

// FuelUseCase defines the business operations for the fuel module.
type FuelUseCase interface {
	CreateRecord(ctx context.Context, orgID string, r domain.FuelRecord) (domain.FuelRecord, error)
	UpdateRecord(ctx context.Context, orgID string, id string, r domain.FuelRecord, userRole string) (domain.FuelRecord, error)
	DeleteRecord(ctx context.Context, orgID string, id string) error
	GetRecords(ctx context.Context, orgID string, filter domain.FuelFilter) ([]domain.FuelRecord, error)
	GetByID(ctx context.Context, orgID string, id string) (domain.FuelRecord, error)
	GetDashboard(ctx context.Context, orgID string) (domain.FuelDashboard, error)
	GetEfficiencyReport(ctx context.Context, orgID string, filter domain.FuelFilter) ([]domain.FuelEfficiencyReport, error)
}

type fuelUseCase struct {
	fuelRepo     domain.FuelRepository
	vehicleRepo  domain.VehicleRepository
	driverRepo   domain.DriverRepository
	auditUseCase AuditUseCase
	logger       *slog.Logger
}

func NewFuelUseCase(
	fuelRepo domain.FuelRepository,
	vehicleRepo domain.VehicleRepository,
	driverRepo domain.DriverRepository,
	auditUseCase AuditUseCase,
	logger *slog.Logger,
) FuelUseCase {
	return &fuelUseCase{
		fuelRepo:     fuelRepo,
		vehicleRepo:  vehicleRepo,
		driverRepo:   driverRepo,
		auditUseCase: auditUseCase,
		logger:       logger,
	}
}

// --------------------------------------------------------------------------
// GetRecords — list with optional filters
// --------------------------------------------------------------------------
func (uc *fuelUseCase) GetRecords(ctx context.Context, orgID string, filter domain.FuelFilter) ([]domain.FuelRecord, error) {
	return uc.fuelRepo.GetAll(ctx, orgID, filter)
}

// --------------------------------------------------------------------------
// GetByID — single record
// --------------------------------------------------------------------------
func (uc *fuelUseCase) GetByID(ctx context.Context, orgID string, id string) (domain.FuelRecord, error) {
	return uc.fuelRepo.GetByID(ctx, orgID, id)
}

// --------------------------------------------------------------------------
// CreateRecord — the most complex operation:
//   1. Validate required fields and fuel type
//   2. Validate date is not too far in the future
//   3. Validate vehicleID belongs to org
//   4. Optionally validate driverID belongs to org (cross-tenant attack mitigation)
//   5. Validate odometer is not a regression vs. the vehicle's last record
//   6. Compute TotalCost = Liters * PricePerLiter (client value is ignored)
//   7. Persist the record
//   8. Run anomaly detection (non-blocking; logged on failure)
//   9. Emit fuel.create audit event
// --------------------------------------------------------------------------
func (uc *fuelUseCase) CreateRecord(ctx context.Context, orgID string, r domain.FuelRecord) (domain.FuelRecord, error) {
	// 1. Input validation
	if err := validateFuelInput(r); err != nil {
		return domain.FuelRecord{}, err
	}

	// 2. Validate vehicle belongs to this org
	if _, err := uc.vehicleRepo.GetByID(ctx, orgID, r.VehicleID); err != nil {
		return domain.FuelRecord{}, domain.ErrFuelVehicleNotInOrg
	}

	// 3. Optionally validate driver belongs to this org
	if r.DriverID != nil && *r.DriverID != "" {
		if _, err := uc.driverRepo.GetByID(ctx, orgID, *r.DriverID); err != nil {
			return domain.FuelRecord{}, domain.ErrFuelDriverNotInOrg
		}
	}

	// 4. Odometer regression check
	if err := uc.validateOdometerNotRegressed(ctx, orgID, r.VehicleID, r.OdometerReading); err != nil {
		return domain.FuelRecord{}, err
	}

	// 5. Compute total cost (always overwritten — client value never trusted)
	r.TotalCost = math.Round(r.Liters*r.PricePerLiter*100) / 100

	// 6. Assign ID and org
	r.ID = uuid.New().String()
	r.OrganizationID = orgID

	// 7. Persist
	created, err := uc.fuelRepo.Create(ctx, orgID, r)
	if err != nil {
		return domain.FuelRecord{}, err
	}

	// 8. Anomaly detection (non-blocking — never fails the request)
	uc.runAnomalyDetection(ctx, orgID, created)

	// 9. Audit
	uc.auditUseCase.Log(ctx, domain.EventFuelCreate, "fuel_record", &created.ID, map[string]interface{}{
		"organization_id":  orgID,
		"vehicle_id":       created.VehicleID,
		"liters":           created.Liters,
		"total_cost":       created.TotalCost,
		"odometer_reading": created.OdometerReading,
		"fuel_type":        created.FuelType,
	})

	return created, nil
}

// --------------------------------------------------------------------------
// UpdateRecord — with 24-hour role-based edit restriction:
//   - Operators: can only edit within 24h of creation.
//   - Owners / Admins: can edit at any time.
//   - All updates recompute TotalCost and run odometer regression check.
//   - Audit includes previous and new values plus list of changed fields.
// --------------------------------------------------------------------------
func (uc *fuelUseCase) UpdateRecord(ctx context.Context, orgID string, id string, r domain.FuelRecord, userRole string) (domain.FuelRecord, error) {
	// Fetch existing record (also validates tenant ownership)
	existing, err := uc.fuelRepo.GetByID(ctx, orgID, id)
	if err != nil {
		return domain.FuelRecord{}, err
	}

	// 24h edit restriction for operators
	isPrivileged := strings.EqualFold(userRole, "owner") || strings.EqualFold(userRole, "admin")
	if !isPrivileged && time.Since(existing.CreatedAt) > editGracePeriod {
		return domain.FuelRecord{}, domain.ErrFuelEditForbiddenAfter24h
	}

	// Validate input fields
	if err := validateFuelInput(r); err != nil {
		return domain.FuelRecord{}, err
	}

	// Validate vehicle if changed
	if r.VehicleID != existing.VehicleID {
		if _, err := uc.vehicleRepo.GetByID(ctx, orgID, r.VehicleID); err != nil {
			return domain.FuelRecord{}, domain.ErrFuelVehicleNotInOrg
		}
	}

	// Validate driver if provided and changed
	if r.DriverID != nil && *r.DriverID != "" {
		if _, err := uc.driverRepo.GetByID(ctx, orgID, *r.DriverID); err != nil {
			return domain.FuelRecord{}, domain.ErrFuelDriverNotInOrg
		}
	}

	// Odometer regression check (compare against the prior record for this vehicle,
	// excluding the record being updated itself)
	if r.OdometerReading != existing.OdometerReading || r.VehicleID != existing.VehicleID {
		if err := uc.validateOdometerNotRegressed(ctx, orgID, r.VehicleID, r.OdometerReading); err != nil {
			// Allow if the regression is only against itself (i.e., the latest record is the one being updated)
			// We re-check: get last record and if it's this record, skip the error
			last, lookupErr := uc.fuelRepo.GetLastNByVehicle(ctx, orgID, r.VehicleID, 1)
			if lookupErr != nil || len(last) == 0 || last[0].ID != id {
				return domain.FuelRecord{}, err
			}
		}
	}

	// Recompute total cost
	r.TotalCost = math.Round(r.Liters*r.PricePerLiter*100) / 100
	r.OrganizationID = orgID

	// Build diff for audit metadata
	changedFields := buildFuelDiff(existing, r)

	// Persist
	updated, err := uc.fuelRepo.Update(ctx, orgID, id, r)
	if err != nil {
		return domain.FuelRecord{}, err
	}

	// Audit with before/after diff
	uc.auditUseCase.Log(ctx, domain.EventFuelUpdate, "fuel_record", &updated.ID, map[string]interface{}{
		"organization_id": orgID,
		"vehicle_id":      updated.VehicleID,
		"changed_fields":  changedFields,
	})

	return updated, nil
}

// --------------------------------------------------------------------------
// DeleteRecord — soft delete with audit
// --------------------------------------------------------------------------
func (uc *fuelUseCase) DeleteRecord(ctx context.Context, orgID string, id string) error {
	// Fetch before deleting (for audit metadata)
	existing, err := uc.fuelRepo.GetByID(ctx, orgID, id)
	if err != nil {
		return err
	}

	if err := uc.fuelRepo.Delete(ctx, orgID, id); err != nil {
		return err
	}

	uc.auditUseCase.Log(ctx, domain.EventFuelDelete, "fuel_record", &id, map[string]interface{}{
		"organization_id": orgID,
		"vehicle_id":      existing.VehicleID,
		"record_date":     existing.Date.Format(time.RFC3339),
		"total_cost":      existing.TotalCost,
	})

	return nil
}

// --------------------------------------------------------------------------
// GetDashboard — aggregated KPIs
// --------------------------------------------------------------------------
func (uc *fuelUseCase) GetDashboard(ctx context.Context, orgID string) (domain.FuelDashboard, error) {
	return uc.fuelRepo.GetDashboardStats(ctx, orgID)
}

// --------------------------------------------------------------------------
// GetEfficiencyReport — per-vehicle aggregations
// --------------------------------------------------------------------------
func (uc *fuelUseCase) GetEfficiencyReport(ctx context.Context, orgID string, filter domain.FuelFilter) ([]domain.FuelEfficiencyReport, error) {
	return uc.fuelRepo.GetEfficiencyReport(ctx, orgID, filter)
}

// --------------------------------------------------------------------------
// Private helpers
// --------------------------------------------------------------------------

// validateFuelInput checks all required fields before any persistence operation.
func validateFuelInput(r domain.FuelRecord) error {
	if _, ok := validFuelTypes[r.FuelType]; !ok {
		return domain.ErrFuelInvalidFuelType
	}
	if r.Liters <= 0 {
		return domain.ErrFuelInvalidLiters
	}
	if r.PricePerLiter <= 0 {
		return domain.ErrFuelInvalidPricePerLiter
	}
	if r.OdometerReading < 0 {
		return domain.ErrFuelInvalidOdometerReading
	}
	// Reject dates more than 24h in the future (timezone buffer)
	if r.Date.After(time.Now().UTC().Add(24 * time.Hour)) {
		return domain.ErrFuelFutureDate
	}
	return nil
}

// validateOdometerNotRegressed checks that the new odometer reading is not below
// the most recent record for the vehicle. Returns ErrFuelOdometerRegression on violation.
func (uc *fuelUseCase) validateOdometerNotRegressed(ctx context.Context, orgID string, vehicleID string, newReading int) error {
	recent, err := uc.fuelRepo.GetLastNByVehicle(ctx, orgID, vehicleID, 1)
	if err != nil {
		return err
	}
	if len(recent) == 0 {
		return nil // No prior records — any reading is valid
	}
	if newReading < recent[0].OdometerReading {
		return domain.ErrFuelOdometerRegression
	}
	return nil
}

// runAnomalyDetection computes km/L for the newly created record and flags it
// if it deviates more than anomalyThreshold from the vehicle's recent baseline.
// This function is intentionally non-blocking: any error is logged but not returned.
func (uc *fuelUseCase) runAnomalyDetection(ctx context.Context, orgID string, created domain.FuelRecord) {
	// Fetch the last N records EXCLUDING the just-created one (it's already in DB, so fetch up to 8 to get 7 prior records)
	recent, err := uc.fuelRepo.GetLastNByVehicle(ctx, orgID, created.VehicleID, 8)
	if err != nil {
		uc.logger.Error("fuel anomaly detection: failed to fetch recent records",
			slog.String("vehicle_id", created.VehicleID), slog.String("err", err.Error()))
		return
	}

	// Filter out the record just created
	var priorRecords []domain.FuelRecord
	for _, rec := range recent {
		if rec.ID != created.ID {
			priorRecords = append(priorRecords, rec)
		}
	}

	// Need at least 3 prior records to activate anomaly detection
	if len(priorRecords) < 3 {
		return
	}

	// Compute the km/L for the current fill-up using the delta from the previous record
	prevRecord := priorRecords[0] // Most recent prior record (ordered DESC)
	deltaKm := created.OdometerReading - prevRecord.OdometerReading
	if deltaKm <= 0 || created.Liters <= 0 {
		return // Cannot compute efficiency — skip detection
	}
	currentEfficiency := float64(deltaKm) / created.Liters

	// Compute the baseline average efficiency from prior records (max 5 data points, which requires 6 prior records)
	var efficiencies []float64
	for i := 1; i < len(priorRecords) && i <= 5; i++ {
		km := priorRecords[i-1].OdometerReading - priorRecords[i].OdometerReading
		lit := priorRecords[i-1].Liters
		if km > 0 && lit > 0 {
			efficiencies = append(efficiencies, float64(km)/lit)
		}
	}
	if len(efficiencies) == 0 {
		return
	}
	var sumEff float64
	for _, e := range efficiencies {
		sumEff += e
	}
	avgEfficiency := sumEff / float64(len(efficiencies))

	// Compute deviation
	deviation := math.Abs(currentEfficiency-avgEfficiency) / avgEfficiency
	if deviation <= anomalyThreshold {
		return // Within acceptable range
	}

	// Flag the record as anomaly (update in DB)
	flagged := created
	flagged.IsAnomaly = true
	if _, err := uc.fuelRepo.Update(ctx, orgID, created.ID, flagged); err != nil {
		uc.logger.Error("fuel anomaly detection: failed to flag record",
			slog.String("record_id", created.ID), slog.String("err", err.Error()))
		return
	}

	// Emit anomaly audit event
	deviationPct := deviation * 100
	uc.auditUseCase.Log(ctx, domain.EventFuelAnomaly, "fuel_record", &created.ID, map[string]interface{}{
		"organization_id":    orgID,
		"vehicle_id":         created.VehicleID,
		"observed_km_l":      math.Round(currentEfficiency*100) / 100,
		"avg_km_l_last_n":    math.Round(avgEfficiency*100) / 100,
		"deviation_pct":      math.Round(deviationPct*10) / 10,
		"threshold_pct":      anomalyThreshold * 100,
	})
}

// buildFuelDiff builds a metadata map describing which fields changed between
// the existing record and the updated values, including before/after values.
func buildFuelDiff(before, after domain.FuelRecord) map[string]interface{} {
	diff := map[string]interface{}{}

	if before.VehicleID != after.VehicleID {
		diff["vehicleId"] = map[string]interface{}{"before": before.VehicleID, "after": after.VehicleID}
	}
	if ptrStr(before.DriverID) != ptrStr(after.DriverID) {
		diff["driverId"] = map[string]interface{}{"before": before.DriverID, "after": after.DriverID}
	}
	if !before.Date.Equal(after.Date) {
		diff["date"] = map[string]interface{}{"before": before.Date.Format(time.RFC3339), "after": after.Date.Format(time.RFC3339)}
	}
	if before.Liters != after.Liters {
		diff["liters"] = map[string]interface{}{"before": before.Liters, "after": after.Liters}
	}
	if before.PricePerLiter != after.PricePerLiter {
		diff["pricePerLiter"] = map[string]interface{}{"before": before.PricePerLiter, "after": after.PricePerLiter}
	}
	if before.TotalCost != after.TotalCost {
		diff["totalCost"] = map[string]interface{}{"before": before.TotalCost, "after": after.TotalCost}
	}
	if before.OdometerReading != after.OdometerReading {
		diff["odometerReading"] = map[string]interface{}{"before": before.OdometerReading, "after": after.OdometerReading}
	}
	if before.FuelType != after.FuelType {
		diff["fuelType"] = map[string]interface{}{"before": before.FuelType, "after": after.FuelType}
	}
	if ptrStr(before.StationName) != ptrStr(after.StationName) {
		diff["stationName"] = map[string]interface{}{"before": before.StationName, "after": after.StationName}
	}
	if ptrStr(before.Notes) != ptrStr(after.Notes) {
		diff["notes"] = map[string]interface{}{"before": before.Notes, "after": after.Notes}
	}

	return diff
}

// ptrStr safely dereferences a *string, returning "" for nil.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Ensure fuelUseCase satisfies the interface at compile time.
var _ FuelUseCase = (*fuelUseCase)(nil)
