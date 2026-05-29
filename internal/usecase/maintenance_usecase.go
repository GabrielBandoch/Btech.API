package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type MaintenanceUseCase interface {
	// Suppliers
	GetSuppliers(ctx context.Context, orgID string) ([]domain.MaintenanceSupplier, error)
	GetSupplierByID(ctx context.Context, orgID string, id string) (domain.MaintenanceSupplier, error)
	CreateSupplier(ctx context.Context, orgID string, supplier domain.MaintenanceSupplier) (domain.MaintenanceSupplier, error)
	UpdateSupplier(ctx context.Context, orgID string, id string, supplier domain.MaintenanceSupplier) (domain.MaintenanceSupplier, error)
	DeleteSupplier(ctx context.Context, orgID string, id string) error

	// Plans
	GetPlans(ctx context.Context, orgID string, vehicleID string) ([]domain.MaintenancePlan, error)
	GetPlanByID(ctx context.Context, orgID string, id string) (domain.MaintenancePlan, error)
	CreatePlan(ctx context.Context, orgID string, plan domain.MaintenancePlan) (domain.MaintenancePlan, error)
	UpdatePlan(ctx context.Context, orgID string, id string, plan domain.MaintenancePlan) (domain.MaintenancePlan, error)
	DeletePlan(ctx context.Context, orgID string, id string) error

	// Records (Maintenances)
	GetMaintenances(ctx context.Context, orgID string, filter domain.MaintenanceFilter) ([]domain.Maintenance, error)
	GetMaintenanceByID(ctx context.Context, orgID string, id string) (domain.Maintenance, error)
	CreateMaintenance(ctx context.Context, orgID string, m domain.Maintenance) (domain.Maintenance, error)
	UpdateMaintenance(ctx context.Context, orgID string, id string, m domain.Maintenance) (domain.Maintenance, error)
	DeleteMaintenance(ctx context.Context, orgID string, id string) error

	// Alerts
	GetAlerts(ctx context.Context, orgID string, status string) ([]domain.MaintenanceAlert, error)
	ResolveAlert(ctx context.Context, orgID string, id string) error

	// Dashboard & Reports
	GetDashboard(ctx context.Context, orgID string) (map[string]interface{}, error)
	GetCostReport(ctx context.Context, orgID string, filter domain.MaintenanceFilter) (map[string]interface{}, error)

	// Trigger calculation of alerts based on current vehicle status/mileage
	CheckAlertsForVehicle(ctx context.Context, orgID string, vehicleID string) error
}

type maintenanceUseCase struct {
	supplierRepo     domain.MaintenanceSupplierRepository
	planRepo         domain.MaintenancePlanRepository
	maintenanceRepo  domain.MaintenanceRepository
	alertRepo        domain.MaintenanceAlertRepository
	vehicleRepo      domain.VehicleRepository
	auditUseCase     AuditUseCase
	logger           *slog.Logger
}

func NewMaintenanceUseCase(
	supplierRepo domain.MaintenanceSupplierRepository,
	planRepo domain.MaintenancePlanRepository,
	maintenanceRepo domain.MaintenanceRepository,
	alertRepo domain.MaintenanceAlertRepository,
	vehicleRepo domain.VehicleRepository,
	auditUseCase AuditUseCase,
	logger *slog.Logger,
) MaintenanceUseCase {
	return &maintenanceUseCase{
		supplierRepo:     supplierRepo,
		planRepo:         planRepo,
		maintenanceRepo:  maintenanceRepo,
		alertRepo:        alertRepo,
		vehicleRepo:      vehicleRepo,
		auditUseCase:     auditUseCase,
		logger:           logger,
	}
}

// Suppliers CRUD
func (uc *maintenanceUseCase) GetSuppliers(ctx context.Context, orgID string) ([]domain.MaintenanceSupplier, error) {
	return uc.supplierRepo.GetAll(ctx, orgID)
}

func (uc *maintenanceUseCase) GetSupplierByID(ctx context.Context, orgID string, id string) (domain.MaintenanceSupplier, error) {
	return uc.supplierRepo.GetByID(ctx, orgID, id)
}

func (uc *maintenanceUseCase) CreateSupplier(ctx context.Context, orgID string, supplier domain.MaintenanceSupplier) (domain.MaintenanceSupplier, error) {
	s, err := uc.supplierRepo.Create(ctx, orgID, supplier)
	if err != nil {
		return domain.MaintenanceSupplier{}, err
	}
	uc.auditUseCase.Log(ctx, domain.EventMaintenanceSupplierCreate, "maintenance_supplier", &s.ID, map[string]interface{}{
		"name": s.Name,
	})
	return s, nil
}

func (uc *maintenanceUseCase) UpdateSupplier(ctx context.Context, orgID string, id string, supplier domain.MaintenanceSupplier) (domain.MaintenanceSupplier, error) {
	s, err := uc.supplierRepo.Update(ctx, orgID, id, supplier)
	if err != nil {
		return domain.MaintenanceSupplier{}, err
	}
	uc.auditUseCase.Log(ctx, domain.EventMaintenanceSupplierUpdate, "maintenance_supplier", &s.ID, map[string]interface{}{
		"name": s.Name,
	})
	return s, nil
}

func (uc *maintenanceUseCase) DeleteSupplier(ctx context.Context, orgID string, id string) error {
	err := uc.supplierRepo.Delete(ctx, orgID, id)
	if err != nil {
		return err
	}
	uc.auditUseCase.Log(ctx, domain.EventMaintenanceSupplierDelete, "maintenance_supplier", &id, nil)
	return nil
}

// Plans CRUD
func (uc *maintenanceUseCase) GetPlans(ctx context.Context, orgID string, vehicleID string) ([]domain.MaintenancePlan, error) {
	return uc.planRepo.GetAll(ctx, orgID, vehicleID)
}

func (uc *maintenanceUseCase) GetPlanByID(ctx context.Context, orgID string, id string) (domain.MaintenancePlan, error) {
	return uc.planRepo.GetByID(ctx, orgID, id)
}

func (uc *maintenanceUseCase) CreatePlan(ctx context.Context, orgID string, plan domain.MaintenancePlan) (domain.MaintenancePlan, error) {
	// Calculate next due values
	uc.calculateNextDue(ctx, orgID, &plan)

	p, err := uc.planRepo.Create(ctx, orgID, plan)
	if err != nil {
		return domain.MaintenancePlan{}, err
	}
	uc.auditUseCase.Log(ctx, domain.EventMaintenancePlanCreate, "maintenance_plan", &p.ID, map[string]interface{}{
		"name":      p.Name,
		"vehicleId": p.VehicleID,
	})

	// Check if this new plan immediately triggers an alert
	_ = uc.CheckAlertsForVehicle(ctx, orgID, p.VehicleID)

	return p, nil
}

func (uc *maintenanceUseCase) UpdatePlan(ctx context.Context, orgID string, id string, plan domain.MaintenancePlan) (domain.MaintenancePlan, error) {
	// To recalculate properly, we fetch the existing plan
	existing, err := uc.planRepo.GetByID(ctx, orgID, id)
	if err != nil {
		return domain.MaintenancePlan{}, err
	}

	// Apply updates to the existing plan struct
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

	// Recalculate next due
	uc.calculateNextDue(ctx, orgID, &existing)

	p, err := uc.planRepo.Update(ctx, orgID, id, existing)
	if err != nil {
		return domain.MaintenancePlan{}, err
	}
	uc.auditUseCase.Log(ctx, domain.EventMaintenancePlanUpdate, "maintenance_plan", &p.ID, map[string]interface{}{
		"name": p.Name,
	})

	// Check alerts for vehicle
	_ = uc.CheckAlertsForVehicle(ctx, orgID, p.VehicleID)

	return p, nil
}

func (uc *maintenanceUseCase) DeletePlan(ctx context.Context, orgID string, id string) error {
	err := uc.planRepo.Delete(ctx, orgID, id)
	if err != nil {
		return err
	}
	uc.auditUseCase.Log(ctx, domain.EventMaintenancePlanDelete, "maintenance_plan", &id, nil)
	return nil
}

// Records CRUD (Maintenances)
func (uc *maintenanceUseCase) GetMaintenances(ctx context.Context, orgID string, filter domain.MaintenanceFilter) ([]domain.Maintenance, error) {
	return uc.maintenanceRepo.GetAll(ctx, orgID, filter)
}

func (uc *maintenanceUseCase) GetMaintenanceByID(ctx context.Context, orgID string, id string) (domain.Maintenance, error) {
	return uc.maintenanceRepo.GetByID(ctx, orgID, id)
}

func (uc *maintenanceUseCase) CreateMaintenance(ctx context.Context, orgID string, m domain.Maintenance) (domain.Maintenance, error) {
	// 1. Create the record
	rec, err := uc.maintenanceRepo.Create(ctx, orgID, m)
	if err != nil {
		return domain.Maintenance{}, err
	}

	uc.auditUseCase.Log(ctx, domain.EventMaintenanceCreate, "maintenance", &rec.ID, map[string]interface{}{
		"type":      rec.Type,
		"status":    rec.Status,
		"vehicleId": rec.VehicleID,
	})

	// 2. Handle side effects (vehicle status/mileage updates, plan calculations, alerts resolution)
	uc.handleMaintenanceSideEffects(ctx, orgID, rec)

	return rec, nil
}

func (uc *maintenanceUseCase) UpdateMaintenance(ctx context.Context, orgID string, id string, m domain.Maintenance) (domain.Maintenance, error) {
	// 1. Update the record
	rec, err := uc.maintenanceRepo.Update(ctx, orgID, id, m)
	if err != nil {
		return domain.Maintenance{}, err
	}

	uc.auditUseCase.Log(ctx, domain.EventMaintenanceUpdate, "maintenance", &rec.ID, map[string]interface{}{
		"type":   rec.Type,
		"status": rec.Status,
	})

	// 2. Handle side effects
	uc.handleMaintenanceSideEffects(ctx, orgID, rec)

	return rec, nil
}

func (uc *maintenanceUseCase) DeleteMaintenance(ctx context.Context, orgID string, id string) error {
	err := uc.maintenanceRepo.Delete(ctx, orgID, id)
	if err != nil {
		return err
	}
	uc.auditUseCase.Log(ctx, domain.EventMaintenanceDelete, "maintenance", &id, nil)
	return nil
}

// Alerts
func (uc *maintenanceUseCase) GetAlerts(ctx context.Context, orgID string, status string) ([]domain.MaintenanceAlert, error) {
	return uc.alertRepo.GetAll(ctx, orgID, status)
}

func (uc *maintenanceUseCase) ResolveAlert(ctx context.Context, orgID string, id string) error {
	err := uc.alertRepo.UpdateStatus(ctx, orgID, id, domain.MaintenanceAlertStatusResolved)
	if err != nil {
		return err
	}
	uc.auditUseCase.Log(ctx, domain.EventMaintenanceAlertResolved, "maintenance_alert", &id, nil)
	return nil
}

// Dashboard & Reports
func (uc *maintenanceUseCase) GetDashboard(ctx context.Context, orgID string) (map[string]interface{}, error) {
	// 1. Get all vehicles
	vehicles, err := uc.vehicleRepo.GetAll(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve vehicles for dashboard: %w", err)
	}

	totalVehicles := len(vehicles)
	underMaintenance := 0
	for _, v := range vehicles {
		if v.Status == "manutencao" {
			underMaintenance++
		}
	}

	// Calculate fleet availability
	availability := 100.0
	if totalVehicles > 0 {
		availability = (float64(totalVehicles-underMaintenance) / float64(totalVehicles)) * 100.0
	}

	// 2. Get active alerts
	alerts, err := uc.alertRepo.GetAll(ctx, orgID, domain.MaintenanceAlertStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active alerts for dashboard: %w", err)
	}

	// Vehicles that need attention (have active alerts)
	attentionVehicleIDs := make(map[string]bool)
	for _, a := range alerts {
		attentionVehicleIDs[a.VehicleID] = true
	}
	needingAttentionCount := len(attentionVehicleIDs)

	// Fetch detailed info of vehicles that need attention
	needingAttentionList := []map[string]interface{}{}
	for _, v := range vehicles {
		if attentionVehicleIDs[v.ID] {
			// Find alerts for this vehicle
			vAlerts := []domain.MaintenanceAlert{}
			for _, a := range alerts {
				if a.VehicleID == v.ID {
					vAlerts = append(vAlerts, a)
				}
			}
			needingAttentionList = append(needingAttentionList, map[string]interface{}{
				"vehicle": v,
				"alerts":  vAlerts,
			})
		}
	}

	return map[string]interface{}{
		"totalVehicles":         totalVehicles,
		"underMaintenanceCount": underMaintenance,
		"needingAttentionCount": needingAttentionCount,
		"fleetAvailability":     availability,
		"vehiclesNeedingAttention": needingAttentionList,
	}, nil
}

func (uc *maintenanceUseCase) GetCostReport(ctx context.Context, orgID string, filter domain.MaintenanceFilter) (map[string]interface{}, error) {
	return uc.maintenanceRepo.GetCostReport(ctx, orgID, filter)
}

// CheckAlertsForVehicle re-evaluates all plans for a vehicle and creates alerts if close to limits
func (uc *maintenanceUseCase) CheckAlertsForVehicle(ctx context.Context, orgID string, vehicleID string) error {
	vehicle, err := uc.vehicleRepo.GetByID(ctx, orgID, vehicleID)
	if err != nil {
		return err
	}

	plans, err := uc.planRepo.GetAll(ctx, orgID, vehicleID)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, p := range plans {
		// Mileage Alert
		if p.NextDueKM != nil {
			dueKM := *p.NextDueKM
			remainingKM := dueKM - vehicle.Mileage

			// Generate alert if within 1000 km or already overdue
			if remainingKM <= 1000 {
				// Check if there is already an active mileage alert for this plan
				activeAlerts, err := uc.alertRepo.GetActiveByPlanID(ctx, orgID, p.ID)
				hasActiveMileageAlert := false
				if err == nil {
					for _, aa := range activeAlerts {
						if aa.Type == domain.MaintenanceAlertTypeMileage {
							hasActiveMileageAlert = true
							break
						}
					}
				}

				if !hasActiveMileageAlert {
					message := fmt.Sprintf("O veículo %s (%s) está a %d km de atingir o limite para a manutenção preventiva: %s.", 
						vehicle.Model, vehicle.Placa, remainingKM, p.Name)
					if remainingKM < 0 {
						message = fmt.Sprintf("O veículo %s (%s) ultrapassou em %d km o limite para a manutenção preventiva: %s.", 
							vehicle.Model, vehicle.Placa, -remainingKM, p.Name)
					}

					alert := domain.MaintenanceAlert{
						VehicleID:         vehicleID,
						MaintenancePlanID: p.ID,
						Type:              domain.MaintenanceAlertTypeMileage,
						Title:             fmt.Sprintf("Manutenção Preventiva Próxima (%s)", p.Name),
						Message:           message,
						Status:            domain.MaintenanceAlertStatusActive,
					}
					createdAlert, err := uc.alertRepo.Create(ctx, orgID, alert)
					if err == nil {
						uc.auditUseCase.Log(ctx, domain.EventMaintenanceAlertCreated, "maintenance_alert", &createdAlert.ID, map[string]interface{}{
							"planId":    p.ID,
							"vehicleId": vehicleID,
							"type":      domain.MaintenanceAlertTypeMileage,
						})
					}
				}
			}
		}

		// Date Alert
		if p.NextDueDate != nil {
			dueDate := *p.NextDueDate
			remainingDays := int(dueDate.Sub(now).Hours() / 24)

			// Generate alert if within 30 days or overdue
			if remainingDays <= 30 {
				activeAlerts, err := uc.alertRepo.GetActiveByPlanID(ctx, orgID, p.ID)
				hasActiveDateAlert := false
				if err == nil {
					for _, aa := range activeAlerts {
						if aa.Type == domain.MaintenanceAlertTypeDate {
							hasActiveDateAlert = true
							break
						}
					}
				}

				if !hasActiveDateAlert {
					message := fmt.Sprintf("A manutenção preventiva %s para o veículo %s (%s) vence em %d dias (%s).", 
						p.Name, vehicle.Model, vehicle.Placa, remainingDays, dueDate.Format("02/01/2006"))
					if remainingDays < 0 {
						message = fmt.Sprintf("A manutenção preventiva %s para o veículo %s (%s) está atrasada em %d dias (venceu em %s).", 
							p.Name, vehicle.Model, vehicle.Placa, -remainingDays, dueDate.Format("02/01/2006"))
					}

					alert := domain.MaintenanceAlert{
						VehicleID:         vehicleID,
						MaintenancePlanID: p.ID,
						Type:              domain.MaintenanceAlertTypeDate,
						Title:             fmt.Sprintf("Manutenção Preventiva Próxima (%s)", p.Name),
						Message:           message,
						Status:            domain.MaintenanceAlertStatusActive,
					}
					createdAlert, err := uc.alertRepo.Create(ctx, orgID, alert)
					if err == nil {
						uc.auditUseCase.Log(ctx, domain.EventMaintenanceAlertCreated, "maintenance_alert", &createdAlert.ID, map[string]interface{}{
							"planId":    p.ID,
							"vehicleId": vehicleID,
							"type":      domain.MaintenanceAlertTypeDate,
						})
					}
				}
			}
		}
	}

	return nil
}

// Helpers
func (uc *maintenanceUseCase) calculateNextDue(ctx context.Context, orgID string, plan *domain.MaintenancePlan) {
	lastKM := 0
	if plan.LastMaintenanceKM != nil {
		lastKM = *plan.LastMaintenanceKM
	} else {
		// Fallback to current vehicle mileage
		vehicle, err := uc.vehicleRepo.GetByID(ctx, orgID, plan.VehicleID)
		if err == nil {
			lastKM = vehicle.Mileage
		}
	}

	if plan.IntervalKM != nil {
		nextKM := lastKM + *plan.IntervalKM
		plan.NextDueKM = &nextKM
	} else {
		plan.NextDueKM = nil
	}

	lastDate := time.Now()
	if plan.LastMaintenanceDate != nil {
		lastDate = *plan.LastMaintenanceDate
	}

	if plan.IntervalMonths != nil {
		nextDate := lastDate.AddDate(0, *plan.IntervalMonths, 0)
		plan.NextDueDate = &nextDate
	} else {
		plan.NextDueDate = nil
	}
}

func (uc *maintenanceUseCase) handleMaintenanceSideEffects(ctx context.Context, orgID string, m domain.Maintenance) {
	// 1. Vehicle status transitions
	var vehicleUpdate domain.Vehicle
	vehicle, err := uc.vehicleRepo.GetByID(ctx, orgID, m.VehicleID)
	if err == nil {
		if m.Status == domain.MaintenanceStatusInProgress {
			vehicleUpdate.Status = "manutencao"
		} else if m.Status == domain.MaintenanceStatusCompleted || m.Status == domain.MaintenanceStatusCanceled {
			vehicleUpdate.Status = "disponivel"
		}

		// Update vehicle mileage if maintenance is completed and odometer at service is greater than current mileage
		if m.Status == domain.MaintenanceStatusCompleted && m.OdometerAtService > vehicle.Mileage {
			vehicleUpdate.Mileage = m.OdometerAtService
		}

		if vehicleUpdate.Status != "" || vehicleUpdate.Mileage > 0 {
			_, _ = uc.vehicleRepo.Update(ctx, orgID, m.VehicleID, vehicleUpdate)
		}
	}

	// 2. Preventive Plan updates
	if m.Type == domain.MaintenanceTypePreventive && m.Status == domain.MaintenanceStatusCompleted && m.MaintenancePlanID != nil {
		planID := *m.MaintenancePlanID
		plan, err := uc.planRepo.GetByID(ctx, orgID, planID)
		if err == nil {
			plan.LastMaintenanceKM = &m.OdometerAtService
			plan.LastMaintenanceDate = &m.Date

			// Recalculate and update
			uc.calculateNextDue(ctx, orgID, &plan)
			_, _ = uc.planRepo.Update(ctx, orgID, planID, plan)

			// Resolve active alerts for this plan
			_ = uc.alertRepo.ResolveByPlanID(ctx, orgID, planID)
			uc.auditUseCase.Log(ctx, domain.EventMaintenanceAlertResolved, "maintenance_plan", &planID, map[string]interface{}{
				"resolvedByMaintenanceId": m.ID,
			})
			uc.auditUseCase.Log(ctx, domain.EventMaintenanceComplete, "maintenance", &m.ID, map[string]interface{}{
				"planId": planID,
			})
		}
	}

	// 3. Re-evaluate alerts for the vehicle in background/after updating status/mileage
	_ = uc.CheckAlertsForVehicle(ctx, orgID, m.VehicleID)
}
