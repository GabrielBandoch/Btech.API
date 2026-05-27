package memory

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type memoryDriverRepository struct {
	mu      sync.RWMutex
	drivers []domain.Driver
}

// NewMemoryDriverRepository returns a new instance of DriverRepository backed by memory.
func NewMemoryDriverRepository() domain.DriverRepository {
	return &memoryDriverRepository{
		drivers: []domain.Driver{
			{
				ID:               "DRV-002",
				Name:             "Carlos Alberto",
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
				Avatar:           "https://lh3.googleusercontent.com/aida-public/AB6AXuChHfHqAEjLxYDvHMsyZBqsFBhiGEOvsmCXwMwYgSHWE196AKvfqU0ifrCsUK8dH4w07C28G6vt_8Yy_CBvwRJ0AuXnukWHCXrPeeE9nFUkV96laFjKV6ljqN6MD24AyXX_wdlX_YNZ3Eo1y4rVOqrq9F-qhiWPlVrbAYyUMbTWYsqEp-uIDx7m0X52JX6zYflxuFW5OQGbP85aiK3nxwjGgaAt3GlkBt3UgCoy6AuyqvwNDozCyJWel0MF-Z4vDMFskV4yA",
				Status:           "descanso",
				Score:            68,
				TripsCount:       34,
				IncidentsCount:   6,
				NextScale:        "Escalado em 8h",
				Role:             "Motorista Interestadual",
				LicenseExpiry:    "2026-05-28",
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
		},
	}
}

func (r *memoryDriverRepository) GetAll(ctx context.Context) ([]domain.Driver, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		r.mu.RLock()
		defer r.mu.RUnlock()
		return r.drivers, nil
	}
}

func (r *memoryDriverRepository) GetByID(ctx context.Context, id string) (domain.Driver, error) {
	select {
	case <-ctx.Done():
		return domain.Driver{}, ctx.Err()
	default:
		r.mu.RLock()
		defer r.mu.RUnlock()
		for _, d := range r.drivers {
			if d.ID == id {
				return d, nil
			}
		}
		return domain.Driver{}, fmt.Errorf("driver not found")
	}
}

func (r *memoryDriverRepository) Create(ctx context.Context, driver domain.Driver) (domain.Driver, error) {
	select {
	case <-ctx.Done():
		return domain.Driver{}, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()
		newDriver := driver
		newDriver.ID = fmt.Sprintf("DRV-%03d", rand.Intn(100)+5)
		r.drivers = append(r.drivers, newDriver)
		return newDriver, nil
	}
}
