package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type TripUseCase interface {
	GetTrips(ctx context.Context) ([]domain.Trip, error)
	GetTripByID(ctx context.Context, id string) (domain.Trip, error)
	UpdateTrip(ctx context.Context, id string, trip domain.Trip) (domain.Trip, error)
}

type tripUseCase struct {
	repo domain.TripRepository
}

// NewTripUseCase creates a new instance of TripUseCase with injected repository.
func NewTripUseCase(repo domain.TripRepository) TripUseCase {
	return &tripUseCase{
		repo: repo,
	}
}

// GetTrips retrieves all trips.
func (uc *tripUseCase) GetTrips(ctx context.Context) ([]domain.Trip, error) {
	return uc.repo.GetAll(ctx)
}

// GetTripByID retrieves a single trip by ID.
func (uc *tripUseCase) GetTripByID(ctx context.Context, id string) (domain.Trip, error) {
	return uc.repo.GetByID(ctx, id)
}

// UpdateTrip updates trip attributes.
func (uc *tripUseCase) UpdateTrip(ctx context.Context, id string, trip domain.Trip) (domain.Trip, error) {
	return uc.repo.Update(ctx, id, trip)
}
