package memory

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type memoryIncidentRepository struct {
	mu        sync.RWMutex
	incidents []domain.Incident
}

// NewMemoryIncidentRepository returns a new instance of IncidentRepository.
func NewMemoryIncidentRepository() domain.IncidentRepository {
	return &memoryIncidentRepository{
		incidents: []domain.Incident{},
	}
}

func (r *memoryIncidentRepository) seedIncidentsForOrg(orgID string) []domain.Incident {
	return []domain.Incident{
		{
			ID:             "INC-001",
			OrganizationID: orgID,
			TripID:         "TR-8820",
			VehiclePlaca:   "MLX-9018",
			DriverName:     "João Santos",
			Type:           "atraso",
			Severity:       "media",
			Description:    "Congestionamento severo de tráfego por acidente na pista.",
			Timestamp:      "2026-05-25T10:15:00Z",
			Location:       "Teófilo Otoni - MG",
			Status:         "aberta",
		},
		{
			ID:             "INC-002",
			OrganizationID: orgID,
			TripID:         "VT-422",
			VehiclePlaca:   "KGB-8840",
			DriverName:     "Marcos Souza",
			Type:           "falha_mecanica",
			Severity:       "critica",
			Description:    "Variação térmica detectada na câmara fria.",
			Timestamp:      "2026-05-25T11:02:00Z",
			Location:       "Joinville - SC",
			Status:         "aberta",
		},
	}
}

func (r *memoryIncidentRepository) GetAll(ctx context.Context, orgID string) ([]domain.Incident, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()

		var orgIncidents []domain.Incident
		for _, inc := range r.incidents {
			if inc.OrganizationID == orgID {
				orgIncidents = append(orgIncidents, inc)
			}
		}

		if len(orgIncidents) == 0 {
			seeded := r.seedIncidentsForOrg(orgID)
			r.incidents = append(r.incidents, seeded...)
			orgIncidents = seeded
		}

		return orgIncidents, nil
	}
}

func (r *memoryIncidentRepository) Create(ctx context.Context, orgID string, incident domain.Incident) (domain.Incident, error) {
	select {
	case <-ctx.Done():
		return domain.Incident{}, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()
		newInc := incident
		newInc.OrganizationID = orgID
		newInc.ID = fmt.Sprintf("INC-%03d", rand.Intn(100)+5)
		newInc.Timestamp = time.Now().Format(time.RFC3339)
		r.incidents = append([]domain.Incident{newInc}, r.incidents...) // prepend
		return newInc, nil
	}
}

func (r *memoryIncidentRepository) Update(ctx context.Context, orgID string, id string, incident domain.Incident) (domain.Incident, error) {
	select {
	case <-ctx.Done():
		return domain.Incident{}, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()

		var orgHasIncidents bool
		for _, inc := range r.incidents {
			if inc.OrganizationID == orgID {
				orgHasIncidents = true
				break
			}
		}
		if !orgHasIncidents {
			seeded := r.seedIncidentsForOrg(orgID)
			r.incidents = append(r.incidents, seeded...)
		}

		for i, inc := range r.incidents {
			if inc.OrganizationID == orgID && inc.ID == id {
				if incident.Status != "" {
					r.incidents[i].Status = incident.Status
				}
				return r.incidents[i], nil
			}
		}
		return domain.Incident{}, fmt.Errorf("incident not found")
	}
}
