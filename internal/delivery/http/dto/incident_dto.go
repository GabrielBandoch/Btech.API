package dto

import "github.com/btech/fleetcontrol-api/internal/domain"

type IncidentResponse struct {
	ID           string `json:"id"`
	TripID       string `json:"tripId,omitempty"`
	VehiclePlaca string `json:"vehiclePlaca"`
	DriverName   string `json:"driverName"`
	Type         string `json:"type"`
	Severity     string `json:"severity"`
	Description  string `json:"description"`
	Timestamp    string `json:"timestamp"`
	Location     string `json:"location"`
	Status       string `json:"status"`
}

type CreateIncidentRequest struct {
	TripID       string `json:"tripId,omitempty"`
	VehiclePlaca string `json:"vehiclePlaca"`
	DriverName   string `json:"driverName"`
	Type         string `json:"type"`
	Severity     string `json:"severity"`
	Description  string `json:"description"`
	Location     string `json:"location"`
	Status       string `json:"status"`
}

type UpdateIncidentRequest struct {
	Status string `json:"status"`
}

// IncidentFromDomain maps an Incident domain entity to an IncidentResponse DTO.
func IncidentFromDomain(i domain.Incident) IncidentResponse {
	return IncidentResponse{
		ID:           i.ID,
		TripID:       i.TripID,
		VehiclePlaca: i.VehiclePlaca,
		DriverName:   i.DriverName,
		Type:         i.Type,
		Severity:     i.Severity,
		Description:  i.Description,
		Timestamp:    i.Timestamp,
		Location:     i.Location,
		Status:       i.Status,
	}
}

// IncidentFromDomainList converts a slice of Incident domain entities to a slice of IncidentResponse DTOs.
func IncidentFromDomainList(incidents []domain.Incident) []IncidentResponse {
	dtos := make([]IncidentResponse, len(incidents))
	for idx, i := range incidents {
		dtos[idx] = IncidentFromDomain(i)
	}
	return dtos
}

// ToDomain maps a CreateIncidentRequest DTO to an Incident domain entity.
func (r *CreateIncidentRequest) ToDomain() domain.Incident {
	return domain.Incident{
		TripID:       r.TripID,
		VehiclePlaca: r.VehiclePlaca,
		DriverName:   r.DriverName,
		Type:         r.Type,
		Severity:     r.Severity,
		Description:  r.Description,
		Location:     r.Location,
		Status:       r.Status,
	}
}

// ToDomain maps an UpdateIncidentRequest DTO to an Incident domain entity.
func (r *UpdateIncidentRequest) ToDomain() domain.Incident {
	return domain.Incident{
		Status: r.Status,
	}
}
