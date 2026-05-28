package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type DriverUseCase interface {
	GetDrivers(ctx context.Context, orgID string) ([]domain.Driver, error)
	GetDriverByID(ctx context.Context, orgID string, id string) (domain.Driver, error)
	CreateDriver(ctx context.Context, orgID string, driver domain.Driver) (domain.Driver, error)
}

type driverUseCase struct {
	repo         domain.DriverRepository
	auditUseCase AuditUseCase
}

// NewDriverUseCase creates a new instance of DriverUseCase with injected repository and audit usecase.
func NewDriverUseCase(repo domain.DriverRepository, auditUseCase AuditUseCase) DriverUseCase {
	return &driverUseCase{
		repo:         repo,
		auditUseCase: auditUseCase,
	}
}

// GetDrivers retrieves all drivers for the organization.
func (uc *driverUseCase) GetDrivers(ctx context.Context, orgID string) ([]domain.Driver, error) {
	return uc.repo.GetAll(ctx, orgID)
}

// GetDriverByID retrieves a single driver by their ID within the organization.
func (uc *driverUseCase) GetDriverByID(ctx context.Context, orgID string, id string) (domain.Driver, error) {
	return uc.repo.GetByID(ctx, orgID, id)
}

// CreateDriver registers a new driver for the organization.
func (uc *driverUseCase) CreateDriver(ctx context.Context, orgID string, driver domain.Driver) (domain.Driver, error) {
	d, err := uc.repo.Create(ctx, orgID, driver)
	if err != nil {
		return d, err
	}

	uc.auditUseCase.Log(ctx, domain.EventDriverCreate, "driver", &d.ID, map[string]interface{}{
		"driver_name": d.Name,
		"license":     d.LicenseExpiry,
	})

	return d, nil
}
