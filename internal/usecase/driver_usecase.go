package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type DriverUseCase interface {
	GetDrivers(ctx context.Context) ([]domain.Driver, error)
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
