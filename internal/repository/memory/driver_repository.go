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
		drivers: []domain.Driver{},
	}
}

func (r *memoryDriverRepository) seedDriversForOrg(orgID string) []domain.Driver {
	return []domain.Driver{
		{
			ID:               "DRV-002",
			OrganizationID:   orgID,
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
			OrganizationID:   orgID,
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
			OrganizationID:   orgID,
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
}

func (r *memoryDriverRepository) GetAll(ctx context.Context, orgID string) ([]domain.Driver, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()

		var orgDrivers []domain.Driver
		for _, d := range r.drivers {
			if d.OrganizationID == orgID {
				orgDrivers = append(orgDrivers, d)
			}
		}

		if len(orgDrivers) == 0 {
			seeded := r.seedDriversForOrg(orgID)
			r.drivers = append(r.drivers, seeded...)
			orgDrivers = seeded
		}

		return orgDrivers, nil
	}
}

func (r *memoryDriverRepository) GetByID(ctx context.Context, orgID string, id string) (domain.Driver, error) {
	select {
	case <-ctx.Done():
		return domain.Driver{}, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()

		var orgHasDrivers bool
		for _, d := range r.drivers {
			if d.OrganizationID == orgID {
				orgHasDrivers = true
				break
			}
		}
		if !orgHasDrivers {
			seeded := r.seedDriversForOrg(orgID)
			r.drivers = append(r.drivers, seeded...)
		}

		for _, d := range r.drivers {
			if d.OrganizationID == orgID && d.ID == id {
				return d, nil
			}
		}
		return domain.Driver{}, fmt.Errorf("driver not found")
	}
}

func (r *memoryDriverRepository) Create(ctx context.Context, orgID string, driver domain.Driver) (domain.Driver, error) {
	select {
	case <-ctx.Done():
		return domain.Driver{}, ctx.Err()
	default:
		r.mu.Lock()
		defer r.mu.Unlock()
		newDriver := driver
		newDriver.OrganizationID = orgID
		newDriver.ID = fmt.Sprintf("DRV-%03d", rand.Intn(100)+5)
		r.drivers = append(r.drivers, newDriver)
		return newDriver, nil
	}
}
