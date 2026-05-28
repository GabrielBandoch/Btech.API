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
	repo         domain.IncidentRepository
	auditUseCase AuditUseCase
}

// NewIncidentUseCase creates a new instance of IncidentUseCase.
func NewIncidentUseCase(repo domain.IncidentRepository, auditUseCase AuditUseCase) IncidentUseCase {
	return &incidentUseCase{
		repo:         repo,
		auditUseCase: auditUseCase,
	}
}

// GetIncidents retrieves all incidents for the organization.
func (uc *incidentUseCase) GetIncidents(ctx context.Context, orgID string) ([]domain.Incident, error) {
	return uc.repo.GetAll(ctx, orgID)
}

// CreateIncident registers a new incident for the organization.
func (uc *incidentUseCase) CreateIncident(ctx context.Context, orgID string, incident domain.Incident) (domain.Incident, error) {
	inc, err := uc.repo.Create(ctx, orgID, incident)
	if err != nil {
		return inc, err
	}

	uc.auditUseCase.Log(ctx, domain.EventIncidentCreate, "incident", &inc.ID, map[string]interface{}{
		"type":        inc.Type,
		"severity":    inc.Severity,
		"location":    inc.Location,
		"driver_name": inc.DriverName,
	})

	return inc, nil
}

// UpdateIncident updates incident attributes within the organization.
func (uc *incidentUseCase) UpdateIncident(ctx context.Context, orgID string, id string, incident domain.Incident) (domain.Incident, error) {
	return uc.repo.Update(ctx, orgID, id, incident)
}
