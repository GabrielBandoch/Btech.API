package usecase

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

// --------------------------------------------------------------------------
// Mocks for Fuel UseCase Testing
// --------------------------------------------------------------------------

type mockFuelRepository struct {
	records   map[string]domain.FuelRecord
	byVehicle map[string][]domain.FuelRecord
}

func newMockFuelRepository() *mockFuelRepository {
	return &mockFuelRepository{
		records:   make(map[string]domain.FuelRecord),
		byVehicle: make(map[string][]domain.FuelRecord),
	}
}

func (m *mockFuelRepository) GetAll(ctx context.Context, orgID string, filter domain.FuelFilter) ([]domain.FuelRecord, error) {
	var list []domain.FuelRecord
	for _, r := range m.records {
		if r.OrganizationID == orgID && r.DeletedAt == nil {
			list = append(list, r)
		}
	}
	return list, nil
}

func (m *mockFuelRepository) GetByID(ctx context.Context, orgID string, id string) (domain.FuelRecord, error) {
	r, ok := m.records[id]
	if !ok || r.OrganizationID != orgID || r.DeletedAt != nil {
		return domain.FuelRecord{}, domain.ErrFuelRecordNotFound
	}
	return r, nil
}

func (m *mockFuelRepository) Create(ctx context.Context, orgID string, r domain.FuelRecord) (domain.FuelRecord, error) {
	r.OrganizationID = orgID
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	m.records[r.ID] = r
	m.byVehicle[r.VehicleID] = append([]domain.FuelRecord{r}, m.byVehicle[r.VehicleID]...) // DESC order by date (prepend)
	return r, nil
}

func (m *mockFuelRepository) Update(ctx context.Context, orgID string, id string, r domain.FuelRecord) (domain.FuelRecord, error) {
	existing, ok := m.records[id]
	if !ok || existing.OrganizationID != orgID || existing.DeletedAt != nil {
		return domain.FuelRecord{}, domain.ErrFuelRecordNotFound
	}
	r.ID = id
	r.OrganizationID = orgID
	r.CreatedAt = existing.CreatedAt
	r.UpdatedAt = time.Now()
	m.records[id] = r

	// Update byVehicle list
	list := m.byVehicle[r.VehicleID]
	for idx, val := range list {
		if val.ID == id {
			list[idx] = r
			break
		}
	}
	m.byVehicle[r.VehicleID] = list
	return r, nil
}

func (m *mockFuelRepository) Delete(ctx context.Context, orgID string, id string) error {
	r, ok := m.records[id]
	if !ok || r.OrganizationID != orgID || r.DeletedAt != nil {
		return domain.ErrFuelRecordNotFound
	}
	now := time.Now()
	r.DeletedAt = &now
	m.records[id] = r
	return nil
}

func (m *mockFuelRepository) GetLastNByVehicle(ctx context.Context, orgID string, vehicleID string, n int) ([]domain.FuelRecord, error) {
	var list []domain.FuelRecord
	for _, r := range m.byVehicle[vehicleID] {
		if r.OrganizationID == orgID && r.DeletedAt == nil {
			list = append(list, r)
			if len(list) == n {
				break
			}
		}
	}
	return list, nil
}

func (m *mockFuelRepository) GetEfficiencyReport(ctx context.Context, orgID string, filter domain.FuelFilter) ([]domain.FuelEfficiencyReport, error) {
	return []domain.FuelEfficiencyReport{}, nil
}

func (m *mockFuelRepository) GetDashboardStats(ctx context.Context, orgID string) (domain.FuelDashboard, error) {
	return domain.FuelDashboard{}, nil
}

type mockVehicleRepository struct {
	vehicles map[string]domain.Vehicle
}

func newMockVehicleRepository() *mockVehicleRepository {
	return &mockVehicleRepository{
		vehicles: make(map[string]domain.Vehicle),
	}
}

func (m *mockVehicleRepository) GetAll(ctx context.Context, orgID string) ([]domain.Vehicle, error) {
	return nil, nil
}

func (m *mockVehicleRepository) GetByID(ctx context.Context, orgID string, id string) (domain.Vehicle, error) {
	v, ok := m.vehicles[id]
	if !ok || v.OrganizationID != orgID {
		return domain.Vehicle{}, errors.New("vehicle not found")
	}
	return v, nil
}

func (m *mockVehicleRepository) Create(ctx context.Context, orgID string, vehicle domain.Vehicle) (domain.Vehicle, error) {
	return domain.Vehicle{}, nil
}

func (m *mockVehicleRepository) Update(ctx context.Context, orgID string, id string, vehicle domain.Vehicle) (domain.Vehicle, error) {
	return domain.Vehicle{}, nil
}

type mockDriverRepository struct {
	drivers map[string]domain.Driver
}

func newMockDriverRepository() *mockDriverRepository {
	return &mockDriverRepository{
		drivers: make(map[string]domain.Driver),
	}
}

func (m *mockDriverRepository) GetAll(ctx context.Context, orgID string) ([]domain.Driver, error) {
	return nil, nil
}

func (m *mockDriverRepository) GetByID(ctx context.Context, orgID string, id string) (domain.Driver, error) {
	d, ok := m.drivers[id]
	if !ok || d.OrganizationID != orgID {
		return domain.Driver{}, errors.New("driver not found")
	}
	return d, nil
}

func (m *mockDriverRepository) Create(ctx context.Context, orgID string, driver domain.Driver) (domain.Driver, error) {
	return domain.Driver{}, nil
}

func (m *mockDriverRepository) Count(ctx context.Context, orgID string) (int, error) {
	return 0, nil
}

// --------------------------------------------------------------------------
// UseCase Tests
// --------------------------------------------------------------------------

func TestFuelUseCase_CreateRecord(t *testing.T) {
	fuelRepo := newMockFuelRepository()
	vehicleRepo := newMockVehicleRepository()
	driverRepo := newMockDriverRepository()
	audit := &mockAuditUseCase{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := NewFuelUseCase(fuelRepo, vehicleRepo, driverRepo, audit, logger)

	ctx := context.Background()
	orgID := "org-1"

	// Seed tenant assets
	vehicleRepo.vehicles["V1"] = domain.Vehicle{ID: "V1", OrganizationID: orgID}
	driverRepo.drivers["D1"] = domain.Driver{ID: "D1", OrganizationID: orgID}

	t.Run("Success_Create_Minimal", func(t *testing.T) {
		r := domain.FuelRecord{
			VehicleID:       "V1",
			Liters:          50.0,
			PricePerLiter:   6.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}

		created, err := uc.CreateRecord(ctx, orgID, r)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if created.ID == "" {
			t.Error("expected generated ID to be populated")
		}
		if created.TotalCost != 300.0 {
			t.Errorf("expected computed TotalCost to be 300.0, got %f", created.TotalCost)
		}
		if created.IsAnomaly {
			t.Error("expected IsAnomaly to be false with first record")
		}
	})

	t.Run("Validation_InvalidFuelType", func(t *testing.T) {
		r := domain.FuelRecord{
			VehicleID:       "V1",
			Liters:          50.0,
			PricePerLiter:   6.0,
			OdometerReading: 1000,
			FuelType:        "water",
			Date:            time.Now(),
		}
		_, err := uc.CreateRecord(ctx, orgID, r)
		if !errors.Is(err, domain.ErrFuelInvalidFuelType) {
			t.Errorf("expected ErrFuelInvalidFuelType, got %v", err)
		}
	})

	t.Run("Validation_InvalidLiters", func(t *testing.T) {
		r := domain.FuelRecord{
			VehicleID:       "V1",
			Liters:          0,
			PricePerLiter:   6.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}
		_, err := uc.CreateRecord(ctx, orgID, r)
		if !errors.Is(err, domain.ErrFuelInvalidLiters) {
			t.Errorf("expected ErrFuelInvalidLiters, got %v", err)
		}
	})

	t.Run("Validation_FutureDate", func(t *testing.T) {
		r := domain.FuelRecord{
			VehicleID:       "V1",
			Liters:          50.0,
			PricePerLiter:   6.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now().Add(48 * time.Hour),
		}
		_, err := uc.CreateRecord(ctx, orgID, r)
		if !errors.Is(err, domain.ErrFuelFutureDate) {
			t.Errorf("expected ErrFuelFutureDate, got %v", err)
		}
	})

	t.Run("TenantCheck_VehicleNotInOrg", func(t *testing.T) {
		// V2 belongs to org-2
		vehicleRepo.vehicles["V2"] = domain.Vehicle{ID: "V2", OrganizationID: "org-2"}
		r := domain.FuelRecord{
			VehicleID:       "V2",
			Liters:          50.0,
			PricePerLiter:   6.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}
		_, err := uc.CreateRecord(ctx, orgID, r)
		if !errors.Is(err, domain.ErrFuelVehicleNotInOrg) {
			t.Errorf("expected ErrFuelVehicleNotInOrg, got %v", err)
		}
	})

	t.Run("TenantCheck_DriverNotInOrg", func(t *testing.T) {
		driverRepo.drivers["D2"] = domain.Driver{ID: "D2", OrganizationID: "org-2"}
		driverID := "D2"
		r := domain.FuelRecord{
			VehicleID:       "V1",
			DriverID:        &driverID,
			Liters:          50.0,
			PricePerLiter:   6.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}
		_, err := uc.CreateRecord(ctx, orgID, r)
		if !errors.Is(err, domain.ErrFuelDriverNotInOrg) {
			t.Errorf("expected ErrFuelDriverNotInOrg, got %v", err)
		}
	})

	t.Run("OdometerRegression_Fail", func(t *testing.T) {
		// Seed a prior record with odometer = 1000
		_, _ = fuelRepo.Create(ctx, orgID, domain.FuelRecord{
			ID:              "prev-rec",
			VehicleID:       "V1",
			Liters:          40.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now().Add(-1 * time.Hour),
		})

		r := domain.FuelRecord{
			VehicleID:       "V1",
			Liters:          50.0,
			PricePerLiter:   6.0,
			OdometerReading: 990, // Regression!
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}
		_, err := uc.CreateRecord(ctx, orgID, r)
		if !errors.Is(err, domain.ErrFuelOdometerRegression) {
			t.Errorf("expected ErrFuelOdometerRegression, got %v", err)
		}
	})

	t.Run("AnomalyDetection_Triggered", func(t *testing.T) {
		vehicleID := "V-ANOMALY-TEST"
		vehicleRepo.vehicles[vehicleID] = domain.Vehicle{ID: vehicleID, OrganizationID: orgID}

		// Baseline: 10 km/L average
		// Record 1: Odo 1000, Liters 50
		// Record 2: Odo 1500, Liters 50 (Odo delta 500, Eff = 10 km/L)
		// Record 3: Odo 2000, Liters 50 (Odo delta 500, Eff = 10 km/L)
		// Record 4: Odo 2500, Liters 50 (Odo delta 500, Eff = 10 km/L)
		// Record 5 (New): Odo 2800, Liters 50 (Odo delta 300, Eff = 6 km/L)
		// Deviation: (10 - 6) / 10 = 0.40 (40%) -> Above 30% threshold -> Anomaly!

		recs := []domain.FuelRecord{
			{ID: "a1", VehicleID: vehicleID, Liters: 50, OdometerReading: 1000, FuelType: domain.FuelTypeGasoline, Date: time.Now().Add(-5 * time.Hour)},
			{ID: "a2", VehicleID: vehicleID, Liters: 50, OdometerReading: 1500, FuelType: domain.FuelTypeGasoline, Date: time.Now().Add(-4 * time.Hour)},
			{ID: "a3", VehicleID: vehicleID, Liters: 50, OdometerReading: 2000, FuelType: domain.FuelTypeGasoline, Date: time.Now().Add(-3 * time.Hour)},
			{ID: "a4", VehicleID: vehicleID, Liters: 50, OdometerReading: 2500, FuelType: domain.FuelTypeGasoline, Date: time.Now().Add(-2 * time.Hour)},
		}

		for _, rec := range recs {
			_, _ = fuelRepo.Create(ctx, orgID, rec)
		}

		r := domain.FuelRecord{
			VehicleID:       vehicleID,
			Liters:          50.0,
			PricePerLiter:   6.0,
			OdometerReading: 2800, // Efficiency = 6 km/L.
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}

		created, err := uc.CreateRecord(ctx, orgID, r)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}

		// Wait briefly or verify directly. (Since mock repo works in memory, it runs synchronously)
		// The mock repo is updated synchronously inside the use case execution.
		updated, _ := fuelRepo.GetByID(ctx, orgID, created.ID)
		if !updated.IsAnomaly {
			t.Error("expected newly created record efficiency deviation to trigger IsAnomaly = true")
		}
	})
}

func TestFuelUseCase_UpdateRecord(t *testing.T) {
	fuelRepo := newMockFuelRepository()
	vehicleRepo := newMockVehicleRepository()
	driverRepo := newMockDriverRepository()
	audit := &mockAuditUseCase{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := NewFuelUseCase(fuelRepo, vehicleRepo, driverRepo, audit, logger)

	ctx := context.Background()
	orgID := "org-1"

	vehicleRepo.vehicles["V1"] = domain.Vehicle{ID: "V1", OrganizationID: orgID}

	t.Run("OperatorEdit_SuccessWithin24h", func(t *testing.T) {
		r := domain.FuelRecord{
			ID:              "up-1",
			VehicleID:       "V1",
			Liters:          40.0,
			PricePerLiter:   5.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}
		// Create record (sets CreatedAt to now)
		created, _ := fuelRepo.Create(ctx, orgID, r)

		// Operator attempts to edit immediately (within 24h)
		updateData := created
		updateData.Liters = 45.0 // Change liters

		updated, err := uc.UpdateRecord(ctx, orgID, created.ID, updateData, "operator")
		if err != nil {
			t.Fatalf("expected operator update to succeed within 24h, got %v", err)
		}

		if updated.Liters != 45.0 {
			t.Errorf("expected liters to be updated to 45, got %f", updated.Liters)
		}
		if updated.TotalCost != 225.0 {
			t.Errorf("expected total cost to be recomputed to 225.0, got %f", updated.TotalCost)
		}
	})

	t.Run("OperatorEdit_ForbiddenAfter24h", func(t *testing.T) {
		r := domain.FuelRecord{
			ID:              "up-2",
			VehicleID:       "V1",
			Liters:          40.0,
			PricePerLiter:   5.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}
		// Create record in the past
		created, _ := fuelRepo.Create(ctx, orgID, r)
		created.CreatedAt = time.Now().Add(-25 * time.Hour)
		m, _ := fuelRepo.records[created.ID]
		m.CreatedAt = time.Now().Add(-25 * time.Hour)
		fuelRepo.records[created.ID] = m

		// Operator tries to edit
		updateData := created
		updateData.Liters = 45.0

		_, err := uc.UpdateRecord(ctx, orgID, created.ID, updateData, "operator")
		if !errors.Is(err, domain.ErrFuelEditForbiddenAfter24h) {
			t.Errorf("expected ErrFuelEditForbiddenAfter24h for operator after 24h, got %v", err)
		}
	})

	t.Run("AdminEdit_SuccessAfter24h", func(t *testing.T) {
		r := domain.FuelRecord{
			ID:              "up-3",
			VehicleID:       "V1",
			Liters:          40.0,
			PricePerLiter:   5.0,
			OdometerReading: 1000,
			FuelType:        domain.FuelTypeGasoline,
			Date:            time.Now(),
		}
		// Create record in the past
		created, _ := fuelRepo.Create(ctx, orgID, r)
		created.CreatedAt = time.Now().Add(-25 * time.Hour)
		m, _ := fuelRepo.records[created.ID]
		m.CreatedAt = time.Now().Add(-25 * time.Hour)
		fuelRepo.records[created.ID] = m

		// Admin tries to edit
		updateData := created
		updateData.Liters = 50.0

		updated, err := uc.UpdateRecord(ctx, orgID, created.ID, updateData, "admin")
		if err != nil {
			t.Fatalf("expected admin update to succeed after 24h, got %v", err)
		}

		if updated.Liters != 50.0 {
			t.Errorf("expected updated liters to be 50.0, got %f", updated.Liters)
		}
	})
}
