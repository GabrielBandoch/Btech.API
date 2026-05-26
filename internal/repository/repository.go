package repository

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/btech/fleetcontrol-api/internal/model"
)

type Repository struct {
	mu        sync.RWMutex
	trips     []model.Trip
	drivers   []model.Driver
	incidents []model.Incident
}

func NewRepository() *Repository {
	repo := &Repository{}
	repo.seedData()
	return repo
}

func (r *Repository) seedData() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initial seed drivers
	r.drivers = []model.Driver{
		{
			ID:               "DRV-001",
			Name:             "Carlos Alberto",
			Avatar:           "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			Status:           "em_rota",
			Score:            98,
			TripsCount:       42,
			IncidentsCount:   1,
			NextScale:        "BR-116 RJ-SP",
			Role:             "Motorista Rodoviário Sênior",
			LicenseExpiry:    "2027-12-31",
			ToxicologyExpiry: "2026-11-20",
			TrainingExpiry:   "2026-08-15",
		},
		{
			ID:               "DRV-002",
			Name:             "Fernanda Ramos",
			Avatar:           "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			Status:           "online",
			Score:            96,
			TripsCount:       28,
			IncidentsCount:   0,
			NextScale:        "Livre agora",
			Role:             "Operadora Urbana",
			LicenseExpiry:    "2028-05-14",
			ToxicologyExpiry: "2026-09-10",
			TrainingExpiry:   "2027-02-18",
		},
		{
			ID:               "DRV-003",
			Name:             "Marcos Souza",
			Avatar:           "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			Status:           "descanso",
			Score:            68,
			TripsCount:       34,
			IncidentsCount:   6,
			NextScale:        "Escalado em 8h",
			Role:             "Motorista Interestadual",
			LicenseExpiry:    "2026-05-28", // CNH quase vencendo
			ToxicologyExpiry: "2026-06-10",
			TrainingExpiry:   "2026-07-01",
		},
		{
			ID:               "DRV-004",
			Name:             "João Santos",
			Avatar:           "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			Status:           "offline",
			Score:            72,
			TripsCount:       19,
			IncidentsCount:   2,
			NextScale:        "Férias",
			Role:             "Motorista Regional",
			LicenseExpiry:    "2029-01-20",
			ToxicologyExpiry: "2026-12-15",
			TrainingExpiry:   "2026-10-30",
		},
	}

	// Initial seed trips
	r.trips = []model.Trip{
		{
			ID:           "TR-990",
			Origin:       "CD São Paulo - SP",
			Destination:  "CD Rio de Janeiro - RJ",
			Status:       "em_transito",
			DriverName:   "Carlos Alberto",
			DriverAvatar: "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			VehiclePlaca: "BRA-2E19",
			VehicleModel: "Mercedes-Benz Actros",
			CargoType:    "Eletrônicos Premium",
			CargoValue:   450000.00,
			CargoWeight:  8500,
			EstimatedTime: "12:30",
			Speed:        82,
			FuelLevel:    75,
			LastSignalTime: "Faz 1 min",
			CurrentLocation: "Resende - RJ",
			Checkpoints: []model.Checkpoint{
				{Name: "CD São Paulo - SP", Timestamp: "2026-05-25T08:00:00Z", Type: "origin"},
				{Name: "Pedágio Jacareí", Timestamp: "2026-05-25T09:30:00Z", Type: "checkpoint"},
				{Name: "Checkpoint Resende", PlannedTime: "12:15", Type: "current"},
				{Name: "CD Rio de Janeiro - RJ", PlannedTime: "15:00", Type: "destination"},
			},
		},
		{
			ID:           "VT-422",
			Origin:       "CD Curitiba - PR",
			Destination:  "CD Porto Alegre - RS",
			Status:       "em_transito",
			DriverName:   "Marcos Souza",
			DriverAvatar: "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			VehiclePlaca: "KGB-8840",
			VehicleModel: "Volvo FH 540",
			CargoType:    "Vacinas Climatizadas",
			CargoValue:   1200000.00,
			CargoWeight:  4200,
			TemperatureRequired: "-18°C a -22°C",
			EstimatedTime: "14:45",
			Speed:        78,
			FuelLevel:    64,
			LastSignalTime: "Faz 3 min",
			CurrentLocation: "Joinville - SC",
			Checkpoints: []model.Checkpoint{
				{Name: "CD Curitiba - PR", Timestamp: "2026-05-25T07:15:00Z", Type: "origin"},
				{Name: "Pedágio Garuva", Timestamp: "2026-05-25T08:45:00Z", Type: "checkpoint"},
				{Name: "CD Porto Alegre - RS", PlannedTime: "16:30", Type: "destination"},
			},
		},
		{
			ID:           "TR-8820",
			Origin:       "CD Belo Horizonte - MG",
			Destination:  "CD Salvador - BA",
			Status:       "atrasada",
			DriverName:   "João Santos",
			DriverAvatar: "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiNWKZN5WSOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
			VehiclePlaca: "MLX-9018",
			VehicleModel: "Scania R 450",
			CargoType:    "Carga Seca Geral",
			CargoValue:   150000.00,
			CargoWeight:  12000,
			EstimatedTime: "Atrasado (+45m)",
			Speed:        0,
			FuelLevel:    22,
			LastSignalTime: "Faz 12 min",
			CurrentLocation: "Teófilo Otoni - MG",
			Checkpoints: []model.Checkpoint{
				{Name: "CD Belo Horizonte - MG", Timestamp: "2026-05-24T22:00:00Z", Type: "origin"},
				{Name: "Teófilo Otoni - MG", Timestamp: "2026-05-25T04:30:00Z", Type: "checkpoint"},
				{Name: "CD Salvador - BA", PlannedTime: "18:00", Type: "destination"},
			},
		},
	}

	// Initial seed incidents
	r.incidents = []model.Incident{
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
	}
}

func (r *Repository) GetTrips() []model.Trip {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.trips
}

func (r *Repository) GetTripByID(id string) (model.Trip, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.trips {
		if t.ID == id {
			return t, true
		}
	}
	return model.Trip{}, false
}

func (r *Repository) UpdateTrip(id string, updates model.Trip) (model.Trip, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, t := range r.trips {
		if t.ID == id {
			// Apply updates
			if updates.EstimatedTime != "" {
				r.trips[i].EstimatedTime = updates.EstimatedTime
			}
			if updates.Status != "" {
				r.trips[i].Status = updates.Status
			}
			return r.trips[i], true
		}
	}
	return model.Trip{}, false
}

func (r *Repository) GetDrivers() []model.Driver {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.drivers
}

func (r *Repository) CreateDriver(driver model.Driver) model.Driver {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	newDriver := driver
	newDriver.ID = fmt.Sprintf("DRV-%03d", rand.Intn(100)+5)
	r.drivers = append(r.drivers, newDriver)
	return newDriver
}

func (r *Repository) GetIncidents() []model.Incident {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.incidents
}

func (r *Repository) UpdateIncident(id string, updates model.Incident) (model.Incident, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, inc := range r.incidents {
		if inc.ID == id {
			if updates.Status != "" {
				r.incidents[i].Status = updates.Status
			}
			return r.incidents[i], true
		}
	}
	return model.Incident{}, false
}

func (r *Repository) CreateIncident(inc model.Incident) model.Incident {
	r.mu.Lock()
	defer r.mu.Unlock()
	newInc := inc
	newInc.ID = fmt.Sprintf("INC-%03d", rand.Intn(100)+5)
	newInc.Timestamp = time.Now().Format(time.RFC3339)
	r.incidents = append(r.incidents, newInc)
	return newInc
}
