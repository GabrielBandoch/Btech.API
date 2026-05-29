package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type VehicleUseCase interface {
	GetVehicles(ctx context.Context, orgID string) ([]domain.Vehicle, error)
	GetVehicleByID(ctx context.Context, orgID string, id string) (domain.Vehicle, error)
	CreateVehicle(ctx context.Context, orgID string, vehicle domain.Vehicle) (domain.Vehicle, error)
	UpdateVehicle(ctx context.Context, orgID string, id string, vehicle domain.Vehicle) (domain.Vehicle, error)
}

type vehicleUseCase struct {
	repo         domain.VehicleRepository
	auditUseCase AuditUseCase
}

// NewVehicleUseCase instantiates a new VehicleUseCase with dependencies injected.
func NewVehicleUseCase(repo domain.VehicleRepository, auditUseCase AuditUseCase) VehicleUseCase {
	return &vehicleUseCase{
		repo:         repo,
		auditUseCase: auditUseCase,
	}
}

func (uc *vehicleUseCase) GetVehicles(ctx context.Context, orgID string) ([]domain.Vehicle, error) {
	return uc.repo.GetAll(ctx, orgID)
}

func (uc *vehicleUseCase) GetVehicleByID(ctx context.Context, orgID string, id string) (domain.Vehicle, error) {
	return uc.repo.GetByID(ctx, orgID, id)
}

func (uc *vehicleUseCase) CreateVehicle(ctx context.Context, orgID string, vehicle domain.Vehicle) (domain.Vehicle, error) {
	v, err := uc.repo.Create(ctx, orgID, vehicle)
	if err != nil {
		return domain.Vehicle{}, err
	}

	uc.auditUseCase.Log(ctx, domain.EventVehicleCreate, "vehicle", &v.ID, map[string]interface{}{
		"brand": v.Brand,
		"model": v.Model,
		"placa": v.Placa,
	})

	return v, nil
}

func (uc *vehicleUseCase) UpdateVehicle(ctx context.Context, orgID string, id string, vehicle domain.Vehicle) (domain.Vehicle, error) {
	v, err := uc.repo.Update(ctx, orgID, id, vehicle)
	if err != nil {
		return domain.Vehicle{}, err
	}

	uc.auditUseCase.Log(ctx, domain.EventVehicleUpdate, "vehicle", &v.ID, map[string]interface{}{
		"status":  v.Status,
		"mileage": v.Mileage,
	})

	return v, nil
}
