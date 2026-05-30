package dto

import (
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

// --------------------------------------------------------------------------
// Request DTOs
// --------------------------------------------------------------------------

// CreateFuelRecordRequest is the body accepted by POST /fuel/records.
// TotalCost is intentionally absent — it is computed by the use case.
type CreateFuelRecordRequest struct {
	VehicleID       string  `json:"vehicleId"`              // required
	DriverID        string  `json:"driverId,omitempty"`     // optional
	Date            string  `json:"date"`                   // required, RFC3339
	Liters          float64 `json:"liters"`                 // required, > 0
	PricePerLiter   float64 `json:"pricePerLiter"`          // required, > 0
	OdometerReading int     `json:"odometerReading"`        // required, >= 0
	FuelType        string  `json:"fuelType"`               // required, see FuelType constants
	StationName     string  `json:"stationName,omitempty"`  // optional
	Notes           string  `json:"notes,omitempty"`        // optional
}

// UpdateFuelRecordRequest is the body accepted by PUT /fuel/records/{id}.
// All fields are omitempty — any provided field will replace the existing value.
type UpdateFuelRecordRequest struct {
	VehicleID       string  `json:"vehicleId,omitempty"`
	DriverID        string  `json:"driverId,omitempty"`
	Date            string  `json:"date,omitempty"`
	Liters          float64 `json:"liters,omitempty"`
	PricePerLiter   float64 `json:"pricePerLiter,omitempty"`
	OdometerReading int     `json:"odometerReading,omitempty"`
	FuelType        string  `json:"fuelType,omitempty"`
	StationName     string  `json:"stationName,omitempty"`
	Notes           string  `json:"notes,omitempty"`
}

// --------------------------------------------------------------------------
// Response DTOs
// --------------------------------------------------------------------------

// FuelRecordResponse is the shape returned for a single fuel record.
type FuelRecordResponse struct {
	ID              string    `json:"id"`
	VehicleID       string    `json:"vehicleId"`
	DriverID        *string   `json:"driverId,omitempty"`
	Date            time.Time `json:"date"`
	Liters          float64   `json:"liters"`
	PricePerLiter   float64   `json:"pricePerLiter"`
	TotalCost       float64   `json:"totalCost"`
	OdometerReading int       `json:"odometerReading"`
	FuelType        string    `json:"fuelType"`
	StationName     *string   `json:"stationName,omitempty"`
	Notes           *string   `json:"notes,omitempty"`
	IsAnomaly       bool      `json:"isAnomaly"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// FuelDashboardResponse is the shape returned by GET /fuel/dashboard.
type FuelDashboardResponse struct {
	TotalRecordsThisMonth int     `json:"totalRecordsThisMonth"`
	TotalCostThisMonth    float64 `json:"totalCostThisMonth"`
	TotalLitersThisMonth  float64 `json:"totalLitersThisMonth"`
	AvgEfficiencyKmL      float64 `json:"avgEfficiencyKmL"`
	AvgCostPerLiter       float64 `json:"avgCostPerLiter"`
	AnomalyCount          int     `json:"anomalyCount"`
	CostVsLastMonth       float64 `json:"costVsLastMonth"`
	TopConsumingVehicleID *string `json:"topConsumingVehicleId,omitempty"`
}

// FuelEfficiencyReportResponse is the per-vehicle entry in GET /fuel/reports/efficiency.
// AvgEfficiencyKmL and AvgCostPerKm are 0 when insufficient data exists (< 2 records).
// The frontend must render 0 as "—" rather than "0.0 km/L" to avoid misleading users.
type FuelEfficiencyReportResponse struct {
	VehicleID        string  `json:"vehicleId"`
	TotalLiters      float64 `json:"totalLiters"`
	TotalCost        float64 `json:"totalCost"`
	TotalKm          int     `json:"totalKm"`
	AvgEfficiencyKmL float64 `json:"avgEfficiencyKmL"`
	AvgCostPerKm     float64 `json:"avgCostPerKm"`
	RecordCount      int     `json:"recordCount"`
}

// --------------------------------------------------------------------------
// Mappers
// --------------------------------------------------------------------------

// FuelRecordToResponse maps a domain.FuelRecord to FuelRecordResponse.
func FuelRecordToResponse(r domain.FuelRecord) FuelRecordResponse {
	return FuelRecordResponse{
		ID:              r.ID,
		VehicleID:       r.VehicleID,
		DriverID:        r.DriverID,
		Date:            r.Date,
		Liters:          r.Liters,
		PricePerLiter:   r.PricePerLiter,
		TotalCost:       r.TotalCost,
		OdometerReading: r.OdometerReading,
		FuelType:        r.FuelType,
		StationName:     r.StationName,
		Notes:           r.Notes,
		IsAnomaly:       r.IsAnomaly,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
}

// FuelRecordToResponseList maps a slice of domain.FuelRecord to []FuelRecordResponse.
func FuelRecordToResponseList(records []domain.FuelRecord) []FuelRecordResponse {
	result := make([]FuelRecordResponse, 0, len(records))
	for _, r := range records {
		result = append(result, FuelRecordToResponse(r))
	}
	return result
}

// FuelDashboardToResponse maps domain.FuelDashboard to FuelDashboardResponse.
func FuelDashboardToResponse(d domain.FuelDashboard) FuelDashboardResponse {
	return FuelDashboardResponse{
		TotalRecordsThisMonth: d.TotalRecordsThisMonth,
		TotalCostThisMonth:    d.TotalCostThisMonth,
		TotalLitersThisMonth:  d.TotalLitersThisMonth,
		AvgEfficiencyKmL:      d.AvgEfficiencyKmL,
		AvgCostPerLiter:       d.AvgCostPerLiter,
		AnomalyCount:          d.AnomalyCount,
		CostVsLastMonth:       d.CostVsLastMonth,
		TopConsumingVehicleID: d.TopConsumingVehicleID,
	}
}

// FuelEfficiencyReportToResponse maps domain.FuelEfficiencyReport to FuelEfficiencyReportResponse.
func FuelEfficiencyReportToResponse(r domain.FuelEfficiencyReport) FuelEfficiencyReportResponse {
	return FuelEfficiencyReportResponse{
		VehicleID:        r.VehicleID,
		TotalLiters:      r.TotalLiters,
		TotalCost:        r.TotalCost,
		TotalKm:          r.TotalKm,
		AvgEfficiencyKmL: r.AvgEfficiencyKmL,
		AvgCostPerKm:     r.AvgCostPerKm,
		RecordCount:      r.RecordCount,
	}
}

// FuelEfficiencyReportToResponseList maps a slice of domain.FuelEfficiencyReport.
func FuelEfficiencyReportToResponseList(reports []domain.FuelEfficiencyReport) []FuelEfficiencyReportResponse {
	result := make([]FuelEfficiencyReportResponse, 0, len(reports))
	for _, r := range reports {
		result = append(result, FuelEfficiencyReportToResponse(r))
	}
	return result
}
