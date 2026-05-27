package domain

import "context"

type Incident struct {
	ID           string
	TripID       string
	VehiclePlaca string
	DriverName   string
	Type         string
	Severity     string
	Description  string
	Timestamp    string
	Location     string
	Status       string
}

type IncidentRepository interface {
	GetAll(ctx context.Context) ([]Incident, error)
	Create(ctx context.Context, incident Incident) (Incident, error)
	Update(ctx context.Context, id string, incident Incident) (Incident, error)
}
