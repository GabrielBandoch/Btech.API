package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type IncidentUseCase interface {
	GetIncidents(ctx context.Context, orgID string) ([]domain.Incident, error)
	CreateIncident(ctx context.Context, orgID string, incident domain.Incident) (domain.Incident, error)
	UpdateIncident(ctx context.Context, orgID string, id string, incident domain.Incident) (domain.Incident, error)
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

// GetIncidents retrieves all incidents for the organization.
func (uc *incidentUseCase) GetIncidents(ctx context.Context, orgID string) ([]domain.Incident, error) {
	return uc.repo.GetAll(ctx, orgID)
}

// CreateIncident registers a new incident for the organization.
func (uc *incidentUseCase) CreateIncident(ctx context.Context, orgID string, incident domain.Incident) (domain.Incident, error) {
	return uc.repo.Create(ctx, orgID, incident)
}

// UpdateIncident updates incident attributes within the organization.
func (uc *incidentUseCase) UpdateIncident(ctx context.Context, orgID string, id string, incident domain.Incident) (domain.Incident, error) {
	return uc.repo.Update(ctx, orgID, id, incident)
}
