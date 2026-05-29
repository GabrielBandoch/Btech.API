package domain

import (
	"context"
	"time"
)

type Checkpoint struct {
	ID        string    `json:"id"`
	TripID    string    `json:"tripId"`
	Sequence  int       `json:"sequence"`
	Name      string    `json:"name"`
	Latitude  *float64  `json:"latitude"`
	Longitude *float64  `json:"longitude"`
	PlannedAt *string   `json:"plannedAt"`
	ArrivedAt *string   `json:"arrivedAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type Trip struct {
	ID                  string     `json:"id"`
	OrganizationID      string     `json:"organizationId"`
	Origin              string     `json:"origin"`
	Destination         string     `json:"destination"`
	Status              string     `json:"status"`
	DriverName          string     `json:"driverName"`
	DriverAvatar        string     `json:"driverAvatar"`
	VehiclePlaca        string     `json:"vehiclePlaca"`
	VehicleModel        string     `json:"vehicleModel"`
	CargoType           string     `json:"cargoType"`
	CargoValue          float64    `json:"cargoValue"`
	CargoWeight         int        `json:"cargoWeight"`
	TemperatureRequired string     `json:"temperatureRequired"`
	EstimatedTime       string     `json:"estimatedTime"`
	Speed               int        `json:"speed"`
	FuelLevel           int        `json:"fuelLevel"`
	LastSignalTime      string     `json:"lastSignalTime"`
	CurrentLocation     string     `json:"currentLocation"`
	Checkpoints         []Checkpoint `json:"checkpoints"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	DeletedAt           *time.Time `json:"deletedAt,omitempty"`
}

type TripRepository interface {
	GetAll(ctx context.Context, orgID string) ([]Trip, error)
	GetByID(ctx context.Context, orgID string, id string) (Trip, error)
	Update(ctx context.Context, orgID string, id string, trip Trip) (Trip, error)
}

