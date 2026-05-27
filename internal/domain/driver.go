package domain

import "context"

type Driver struct {
	ID               string
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
	GetAll(ctx context.Context) ([]Driver, error)
	GetByID(ctx context.Context, id string) (Driver, error)
	Create(ctx context.Context, driver Driver) (Driver, error)
}
