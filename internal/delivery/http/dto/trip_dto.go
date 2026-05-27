package dto

import "github.com/btech/fleetcontrol-api/internal/domain"

type CheckpointResponse struct {
	Name        string `json:"name"`
	PlannedTime string `json:"plannedTime,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	Type        string `json:"type"`
}

type TripResponse struct {
	ID                  string               `json:"id"`
	Origin              string               `json:"origin"`
	Destination         string               `json:"destination"`
	Status              string               `json:"status"`
	DriverName          string               `json:"driverName"`
	DriverAvatar        string               `json:"driverAvatar"`
	VehiclePlaca        string               `json:"vehiclePlaca"`
	VehicleModel        string               `json:"vehicleModel"`
	CargoType           string               `json:"cargoType"`
	CargoValue          float64              `json:"cargoValue"`
	CargoWeight         int                  `json:"cargoWeight"`
	TemperatureRequired string               `json:"temperatureRequired,omitempty"`
	EstimatedTime       string               `json:"estimatedTime"`
	Speed               int                  `json:"speed"`
	FuelLevel           int                  `json:"fuelLevel"`
	LastSignalTime      string               `json:"lastSignalTime"`
	CurrentLocation     string               `json:"currentLocation"`
	Checkpoints         []CheckpointResponse `json:"checkpoints"`
}

type UpdateTripRequest struct {
	Status        string `json:"status"`
	EstimatedTime string `json:"estimatedTime"`
}

// TripFromDomain maps a Trip domain entity to a TripResponse DTO.
func TripFromDomain(t domain.Trip) TripResponse {
	checkpoints := make([]CheckpointResponse, len(t.Checkpoints))
	for i, c := range t.Checkpoints {
		checkpoints[i] = CheckpointResponse{
			Name:        c.Name,
			PlannedTime: c.PlannedTime,
			Timestamp:   c.Timestamp,
			Type:        c.Type,
		}
	}

	return TripResponse{
		ID:                  t.ID,
		Origin:              t.Origin,
		Destination:         t.Destination,
		Status:              t.Status,
		DriverName:          t.DriverName,
		DriverAvatar:        t.DriverAvatar,
		VehiclePlaca:        t.VehiclePlaca,
		VehicleModel:        t.VehicleModel,
		CargoType:           t.CargoType,
		CargoValue:          t.CargoValue,
		CargoWeight:         t.CargoWeight,
		TemperatureRequired: t.TemperatureRequired,
		EstimatedTime:       t.EstimatedTime,
		Speed:               t.Speed,
		FuelLevel:           t.FuelLevel,
		LastSignalTime:      t.LastSignalTime,
		CurrentLocation:     t.CurrentLocation,
		Checkpoints:         checkpoints,
	}
}

// TripFromDomainList converts a slice of Trip domain entities to a slice of TripResponse DTOs.
func TripFromDomainList(trips []domain.Trip) []TripResponse {
	dtos := make([]TripResponse, len(trips))
	for i, t := range trips {
		dtos[i] = TripFromDomain(t)
	}
	return dtos
}

// ToDomain maps an UpdateTripRequest DTO to a Trip domain entity.
func (r *UpdateTripRequest) ToDomain() domain.Trip {
	return domain.Trip{
		Status:        r.Status,
		EstimatedTime: r.EstimatedTime,
	}
}
