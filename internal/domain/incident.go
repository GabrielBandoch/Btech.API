package domain

import "context"

type Incident struct {
	ID             string
	OrganizationID string
	TripID         string
	VehiclePlaca   string
	DriverName     string
	Type           string
	Severity       string
	Description    string
	Timestamp      string
	Location       string
	Status         string
}

type IncidentRepository interface {
	GetAll(ctx context.Context, orgID string) ([]Incident, error)
	Create(ctx context.Context, orgID string, incident Incident) (Incident, error)
	Update(ctx context.Context, orgID string, id string, incident Incident) (Incident, error)
}
