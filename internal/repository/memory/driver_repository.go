package memory

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type memoryDriverRepository struct {
	drivers []domain.Driver
}

// NewMemoryDriverRepository returns a new instance of DriverRepository backed by memory.
func NewMemoryDriverRepository() domain.DriverRepository {
	return &memoryDriverRepository{
		drivers: []domain.Driver{
			{
				ID:           "1",
				Name:         "João Silva",
				Status:       "active",
				LicensePlate: "ABC-1234",
			},
			{
				ID:           "2",
				Name:         "Maria Santos",
				Status:       "inactive",
				LicensePlate: "XYZ-5678",
			},
			{
				ID:           "3",
				Name:         "Pedro Oliveira",
				Status:       "active",
				LicensePlate: "MNO-9012",
			},
		},
	}
}

// GetAll returns all drivers, respecting the context lifecycle.
func (r *memoryDriverRepository) GetAll(ctx context.Context) ([]domain.Driver, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return r.drivers, nil
	}
}
