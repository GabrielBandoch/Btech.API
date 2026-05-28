package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type TripUseCase interface {
	GetTrips(ctx context.Context, orgID string) ([]domain.Trip, error)
	GetTripByID(ctx context.Context, orgID string, id string) (domain.Trip, error)
	UpdateTrip(ctx context.Context, orgID string, id string, trip domain.Trip) (domain.Trip, error)
}

type tripUseCase struct {
	repo         domain.TripRepository
	auditUseCase AuditUseCase
}

// NewTripUseCase creates a new instance of TripUseCase with injected repository and audit usecase.
func NewTripUseCase(repo domain.TripRepository, auditUseCase AuditUseCase) TripUseCase {
	return &tripUseCase{
		repo:         repo,
		auditUseCase: auditUseCase,
	}
}

// GetTrips retrieves all trips for the organization.
func (uc *tripUseCase) GetTrips(ctx context.Context, orgID string) ([]domain.Trip, error) {
	return uc.repo.GetAll(ctx, orgID)
}

// GetTripByID retrieves a single trip by ID within the organization.
func (uc *tripUseCase) GetTripByID(ctx context.Context, orgID string, id string) (domain.Trip, error) {
	return uc.repo.GetByID(ctx, orgID, id)
}

// UpdateTrip updates trip attributes within the organization.
func (uc *tripUseCase) UpdateTrip(ctx context.Context, orgID string, id string, trip domain.Trip) (domain.Trip, error) {
	t, err := uc.repo.Update(ctx, orgID, id, trip)
	if err != nil {
		return t, err
	}

	uc.auditUseCase.Log(ctx, domain.EventTripUpdate, "trip", &t.ID, map[string]interface{}{
		"status":          t.Status,
		"driver_name":     t.DriverName,
		"vehicle_placa":   t.VehiclePlaca,
		"estimated_time":  t.EstimatedTime,
	})

	return t, nil
}
