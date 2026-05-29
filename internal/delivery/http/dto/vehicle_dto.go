package dto

import "github.com/btech/fleetcontrol-api/internal/domain"

type VehicleResponse struct {
	ID        string `json:"id"`
	Brand     string `json:"brand"`
	Model     string `json:"model"`
	Year      int    `json:"year"`
	Type      string `json:"type"`
	Mileage   int    `json:"mileage"`
	Status    string `json:"status"`
}

type CreateVehicleRequest struct {
	Brand   string `json:"brand"`
	Model   string `json:"model"`
	Year    int    `json:"year"`
	Type    string `json:"type"`
	Mileage int    `json:"mileage"`
	Status  string `json:"status"`
}

// VehicleFromDomain maps a Vehicle domain entity to a VehicleResponse DTO.
func VehicleFromDomain(v domain.Vehicle) VehicleResponse {
	return VehicleResponse{
		ID:      v.ID,
		Brand:   v.Brand,
		Model:   v.Model,
		Year:    v.Year,
		Type:    v.Type,
		Mileage: v.Mileage,
		Status:  v.Status,
	}
}

// VehicleFromDomainList converts a slice of Vehicle domain entities to a slice of VehicleResponse DTOs.
func VehicleFromDomainList(vehicles []domain.Vehicle) []VehicleResponse {
	dtos := make([]VehicleResponse, len(vehicles))
	for i, v := range vehicles {
		dtos[i] = VehicleFromDomain(v)
	}
	return dtos
}

// ToDomain maps a CreateVehicleRequest DTO to a Vehicle domain entity.
func (r *CreateVehicleRequest) ToDomain() domain.Vehicle {
	return domain.Vehicle{
		Brand:   r.Brand,
		Model:   r.Model,
		Year:    r.Year,
		Type:    r.Type,
		Mileage: r.Mileage,
		Status:  r.Status,
	}
}
