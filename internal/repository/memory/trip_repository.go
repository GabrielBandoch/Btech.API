package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type memoryTripRepository struct {
	mu    sync.RWMutex
	trips []domain.Trip
}

// NewMemoryTripRepository returns a new instance of TripRepository.
func NewMemoryTripRepository() domain.TripRepository {
	return &memoryTripRepository{
		trips: []domain.Trip{
			{
				ID:              "TR-990",
				Origin:          "CD São Paulo - SP",
				Destination:     "CD Rio de Janeiro - RJ",
				Status:          "em_transito",
				DriverName:      "Carlos Alberto",
				DriverAvatar:    "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
				VehiclePlaca:    "BRA-2E19",
				VehicleModel:    "Mercedes-Benz Actros",
				CargoType:       "Eletrônicos Premium",
				CargoValue:      450000.00,
				CargoWeight:     8500,
				EstimatedTime:   "12:30",
				Speed:           82,
				FuelLevel:       75,
				LastSignalTime:  "Faz 1 min",
				CurrentLocation: "Resende - RJ",
				Checkpoints: []domain.Checkpoint{
					{Name: "CD São Paulo - SP", Timestamp: "2026-05-25T08:00:00Z", Type: "origin"},
					{Name: "Pedágio Jacareí", Timestamp: "2026-05-25T09:30:00Z", Type: "checkpoint"},
					{Name: "Checkpoint Resende", PlannedTime: "12:15", Type: "current"},
					{Name: "CD Rio de Janeiro - RJ", PlannedTime: "15:00", Type: "destination"},
				},
			},
			{
				ID:                  "VT-422",
				Origin:              "CD Curitiba - PR",
				Destination:         "CD Porto Alegre - RS",
				Status:              "em_transito",
				DriverName:          "Marcos Souza",
				DriverAvatar:        "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
				VehiclePlaca:        "KGB-8840",
				VehicleModel:        "Volvo FH 540",
				CargoType:           "Vacinas Climatizadas",
				CargoValue:          1200000.00,
				CargoWeight:         4200,
				TemperatureRequired: "-18°C a -22°C",
				EstimatedTime:       "14:45",
				Speed:               78,
				FuelLevel:           64,
				LastSignalTime:      "Faz 3 min",
				CurrentLocation:     "Joinville - SC",
				Checkpoints: []domain.Checkpoint{
					{Name: "CD Curitiba - PR", Timestamp: "2026-05-25T07:15:00Z", Type: "origin"},
					{Name: "Pedágio Garuva", Timestamp: "2026-05-25T08:45:00Z", Type: "checkpoint"},
					{Name: "CD Porto Alegre - RS", PlannedTime: "16:30", Type: "destination"},
				},
			},
			{
				ID:              "TR-8820",
				Origin:          "CD Belo Horizonte - MG",
				Destination:     "CD Salvador - BA",
				Status:          "atrasada",
				DriverName:      "João Santos",
				DriverAvatar:    "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
				VehiclePlaca:    "MLX-9018",
				VehicleModel:    "Scania R 450",
				CargoType:       "Carga Seca Geral",
				CargoValue:      150000.00,
				CargoWeight:     12000,
				EstimatedTime:   "Atrasado (+45m)",
				Speed:           0,
				FuelLevel:       22,
				LastSignalTime:  "Faz 12 min",
				CurrentLocation: "Teófilo Otoni - MG",
				Checkpoints: []domain.Checkpoint{
					{Name: "CD Belo Horizonte - MG", Timestamp: "2026-05-24T22:00:00Z", Type: "origin"},
					{Name: "Teófilo Otoni - MG", Timestamp: "2026-05-25T04:30:00Z", Type: "checkpoint"},
					{Name: "CD Salvador - BA", PlannedTime: "18:00", Type: "destination"},
				},
			},
		},
	}
}

func (r *memoryTripRepository) GetAll(ctx context.Context) ([]domain.Trip, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.trips, nil
	}
}

func (r *memoryTripRepository) GetByID(ctx context.Context, id string) (domain.Trip, error) {
	select {
	case <-ctx.Done():
		return domain.Trip{}, ctx.Err()
	default:
		r.mu.RLock()
		defer r.mu.RUnlock()
		for _, t := range r.trips {
			if t.ID == id {
				return t, nil
			}
		}
		return domain.Trip{}, fmt.Errorf("trip not found")
	}
}

func (r *memoryTripRepository) Update(ctx context.Context, id string, trip domain.Trip) (domain.Trip, error) {
	select {
	case <-ctx.Done():
		return domain.Trip{}, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, t := range r.trips {
			if t.ID == id {
				if trip.Status != "" {
					r.trips[i].Status = trip.Status
				}
				if trip.EstimatedTime != "" {
					r.trips[i].EstimatedTime = trip.EstimatedTime
				}
				return r.trips[i], nil
			}
		}
		return domain.Trip{}, fmt.Errorf("trip not found")
	}
}
