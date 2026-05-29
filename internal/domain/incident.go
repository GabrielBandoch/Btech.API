package domain

import (
	"context"
	"time"
)

const (
	IncidentSeverityLow      = "low"
	IncidentSeverityMedium   = "medium"
	IncidentSeverityHigh     = "high"
	IncidentSeverityCritical = "critical"
)

type Incident struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	TripID         string     `json:"tripId"`
	VehiclePlaca   string     `json:"vehiclePlaca"`
	DriverName     string     `json:"driverName"`
	Type           string     `json:"type"`
	Severity       string     `json:"severity"` // low, medium, high, critical
	Description    string     `json:"description"`
	Timestamp      string     `json:"timestamp"`
	Location       string     `json:"location"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}

type IncidentRepository interface {
	GetAll(ctx context.Context, orgID string) ([]Incident, error)
	Create(ctx context.Context, orgID string, incident Incident) (Incident, error)
	Update(ctx context.Context, orgID string, id string, incident Incident) (Incident, error)
}

