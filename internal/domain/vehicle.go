package domain

import (
	"context"
	"time"
)

type Vehicle struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	Placa          string     `json:"placa"`
	Brand          string     `json:"brand"`
	Model          string     `json:"model"`
	Year           int        `json:"year"`
	Type           string     `json:"type"`
	Mileage        int        `json:"mileage"`
	Status         string     `json:"status"` // disponivel, manutencao, em_viagem, etc.
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}

type VehicleRepository interface {
	GetAll(ctx context.Context, orgID string) ([]Vehicle, error)
	GetByID(ctx context.Context, orgID string, id string) (Vehicle, error)
	Create(ctx context.Context, orgID string, vehicle Vehicle) (Vehicle, error)
}
