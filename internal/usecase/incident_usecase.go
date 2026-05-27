package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type IncidentUseCase interface {
	GetIncidents(ctx context.Context) ([]domain.Incident, error)
	CreateIncident(ctx context.Context, incident domain.Incident) (domain.Incident, error)
	UpdateIncident(ctx context.Context, id string, incident domain.Incident) (domain.Incident, error)
}

type incidentUseCase struct {
	repo domain.IncidentRepository
}

// NewIncidentUseCase creates a new instance of IncidentUseCase.
func NewIncidentUseCase(repo domain.IncidentRepository) IncidentUseCase {
	return &incidentUseCase{
		repo: repo,
	}
}

// GetIncidents retrieves all incidents.
func (uc *incidentUseCase) GetIncidents(ctx context.Context) ([]domain.Incident, error) {
	return uc.repo.GetAll(ctx)
}

// CreateIncident registers a new incident.
func (uc *incidentUseCase) CreateIncident(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	return uc.repo.Create(ctx, incident)
}

// UpdateIncident updates incident attributes.
func (uc *incidentUseCase) UpdateIncident(ctx context.Context, id string, incident domain.Incident) (domain.Incident, error) {
	return uc.repo.Update(ctx, id, incident)
}
