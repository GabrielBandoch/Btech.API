package domain

import "context"

type Checkpoint struct {
	Name        string
	PlannedTime string
	Timestamp   string
	Type        string
}

type Trip struct {
	ID                  string
	OrganizationID      string
	Origin              string
	Destination         string
	Status              string
	DriverName          string
	DriverAvatar        string
	VehiclePlaca        string
	VehicleModel        string
	CargoType           string
	CargoValue          float64
	CargoWeight         int
	TemperatureRequired string
	EstimatedTime       string
	Speed               int
	FuelLevel           int
	LastSignalTime      string
	CurrentLocation     string
	Checkpoints         []Checkpoint
}

type TripRepository interface {
	GetAll(ctx context.Context, orgID string) ([]Trip, error)
	GetByID(ctx context.Context, orgID string, id string) (Trip, error)
	Update(ctx context.Context, orgID string, id string, trip Trip) (Trip, error)
}
