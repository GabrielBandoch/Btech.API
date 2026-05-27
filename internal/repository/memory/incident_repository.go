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
		incidents: []domain.Incident{
			{
				ID:           "INC-001",
				TripID:       "TR-8820",
				VehiclePlaca: "MLX-9018",
				DriverName:   "João Santos",
				Type:         "atraso",
				Severity:     "media",
				Description:  "Congestionamento severo de tráfego por acidente na pista.",
				Timestamp:    "2026-05-25T10:15:00Z",
				Location:     "Teófilo Otoni - MG",
				Status:       "aberta",
			},
			{
				ID:           "INC-002",
				TripID:       "VT-422",
				VehiclePlaca: "KGB-8840",
				DriverName:   "Marcos Souza",
				Type:         "falha_mecanica",
				Severity:     "critica",
				Description:  "Variação térmica detectada na câmara fria.",
				Timestamp:    "2026-05-25T11:02:00Z",
				Location:     "Joinville - SC",
				Status:       "aberta",
			},
		},
	}
}

func (r *memoryIncidentRepository) GetAll(ctx context.Context) ([]domain.Incident, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.incidents, nil
	}
}

func (r *memoryIncidentRepository) Create(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	select {
	case <-ctx.Done():
		return domain.Incident{}, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()
		newInc := incident
		newInc.ID = fmt.Sprintf("INC-%03d", rand.Intn(100)+5)
		newInc.Timestamp = time.Now().Format(time.RFC3339)
		r.incidents = append([]domain.Incident{newInc}, r.incidents...) // prepend
		return newInc, nil
	}
}

func (r *memoryIncidentRepository) Update(ctx context.Context, id string, incident domain.Incident) (domain.Incident, error) {
	select {
	case <-ctx.Done():
		return domain.Incident{}, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, inc := range r.incidents {
			if inc.ID == id {
				if incident.Status != "" {
					r.incidents[i].Status = incident.Status
				}
				return r.incidents[i], nil
			}
		}
		return domain.Incident{}, fmt.Errorf("incident not found")
	}
}
