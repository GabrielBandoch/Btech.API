package dto

import "github.com/btech/fleetcontrol-api/internal/domain"

type DriverResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	LicensePlate string `json:"licensePlate"`
}

// FromDomain maps a Driver domain entity to a DriverResponse DTO.
func FromDomain(d domain.Driver) DriverResponse {
	return DriverResponse{
		ID:           d.ID,
		Name:         d.Name,
		Status:       d.Status,
		LicensePlate: d.LicensePlate,
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
