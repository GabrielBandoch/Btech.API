package domain

import "context"

type Driver struct {
	ID               string
	OrganizationID   string
	Name             string
	Avatar           string
	Status           string
	Score            int
	TripsCount       int
	IncidentsCount   int
	NextScale        string
	Role             string
	LicenseExpiry    string
	ToxicologyExpiry string
	TrainingExpiry   string
}

type DriverRepository interface {
	GetAll(ctx context.Context, orgID string) ([]Driver, error)
	GetByID(ctx context.Context, orgID string, id string) (Driver, error)
	Create(ctx context.Context, orgID string, driver Driver) (Driver, error)
}
