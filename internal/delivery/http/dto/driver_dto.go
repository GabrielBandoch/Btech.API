package dto

import "github.com/btech/fleetcontrol-api/internal/domain"

type DriverResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Avatar           string `json:"avatar"`
	Status           string `json:"status"`
	Score            int    `json:"score"`
	TripsCount       int    `json:"tripsCount"`
	IncidentsCount   int    `json:"incidentsCount"`
	NextScale        string `json:"nextScale,omitempty"`
	Role             string `json:"role"`
	LicenseExpiry    string `json:"licenseExpiry"`
	ToxicologyExpiry string `json:"toxicologyExpiry"`
	TrainingExpiry   string `json:"trainingExpiry"`
}

type CreateDriverRequest struct {
	Name             string `json:"name"`
	Avatar           string `json:"avatar"`
	Status           string `json:"status"`
	Role             string `json:"role"`
	LicenseExpiry    string `json:"licenseExpiry"`
	ToxicologyExpiry string `json:"toxicologyExpiry"`
	TrainingExpiry   string `json:"trainingExpiry"`
}

// FromDomain maps a Driver domain entity to a DriverResponse DTO.
func FromDomain(d domain.Driver) DriverResponse {
	return DriverResponse{
		ID:               d.ID,
		Name:             d.Name,
		Avatar:           d.Avatar,
		Status:           d.Status,
		Score:            d.Score,
		TripsCount:       d.TripsCount,
		IncidentsCount:   d.IncidentsCount,
		NextScale:        d.NextScale,
		Role:             d.Role,
		LicenseExpiry:    d.LicenseExpiry,
		ToxicologyExpiry: d.ToxicologyExpiry,
		TrainingExpiry:   d.TrainingExpiry,
	}
}

// FromDomainList converts a slice of Driver domain entities to a slice of DriverResponse DTOs.
func FromDomainList(drivers []domain.Driver) []DriverResponse {
	dtos := make([]DriverResponse, len(drivers))
	for i, d := range drivers {
		dtos[i] = FromDomain(d)
	}
	return dtos
}

// ToDomain maps a CreateDriverRequest DTO to a Driver domain entity.
func (r *CreateDriverRequest) ToDomain() domain.Driver {
	return domain.Driver{
		Name:             r.Name,
		Avatar:           r.Avatar,
		Status:           r.Status,
		Role:             r.Role,
		LicenseExpiry:    r.LicenseExpiry,
		ToxicologyExpiry: r.ToxicologyExpiry,
		TrainingExpiry:   r.TrainingExpiry,
	}
}
