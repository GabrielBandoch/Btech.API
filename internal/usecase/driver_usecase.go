package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type DriverUseCase interface {
	GetDrivers(ctx context.Context) ([]domain.Driver, error)
	GetDriverByID(ctx context.Context, id string) (domain.Driver, error)
	CreateDriver(ctx context.Context, driver domain.Driver) (domain.Driver, error)
}

type driverUseCase struct {
	repo domain.DriverRepository
}

// NewDriverUseCase creates a new instance of DriverUseCase with injected repository.
func NewDriverUseCase(repo domain.DriverRepository) DriverUseCase {
	return &driverUseCase{
		repo: repo,
	}
}

// GetDrivers retrieves all drivers from the repository.
func (uc *driverUseCase) GetDrivers(ctx context.Context) ([]domain.Driver, error) {
	return uc.repo.GetAll(ctx)
}

// GetDriverByID retrieves a single driver by their ID.
func (uc *driverUseCase) GetDriverByID(ctx context.Context, id string) (domain.Driver, error) {
	return uc.repo.GetByID(ctx, id)
}

// CreateDriver registers a new driver.
func (uc *driverUseCase) CreateDriver(ctx context.Context, driver domain.Driver) (domain.Driver, error) {
	return uc.repo.Create(ctx, driver)
}
