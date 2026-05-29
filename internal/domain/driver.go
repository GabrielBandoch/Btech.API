package domain

import (
	"context"
	"time"
)

const (
	DriverStatusActive   = "active"
	DriverStatusInactive = "inactive"
	DriverStatusBlocked  = "blocked"
)

type Driver struct {
	ID               string
	OrganizationID   string
	Name             string
	Avatar           string
	Status           string // active, inactive, blocked
	Score            int
	TripsCount       int
	IncidentsCount   int
	NextScale        string
	Role             string
	LicenseExpiry    string
	ToxicologyExpiry string
	TrainingExpiry   string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

type DriverRepository interface {
	GetAll(ctx context.Context, orgID string) ([]Driver, error)
	GetByID(ctx context.Context, orgID string, id string) (Driver, error)
	Create(ctx context.Context, orgID string, driver Driver) (Driver, error)
	Count(ctx context.Context, orgID string) (int, error)
}

