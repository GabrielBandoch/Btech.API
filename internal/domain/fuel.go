package domain

import (
	"context"
	"errors"
	"time"
)

// FuelType constants
const (
	FuelTypeGasoline = "gasoline"
	FuelTypeEthanol  = "ethanol"
	FuelTypeDiesel   = "diesel"
	FuelTypeGNV      = "gnv"
	FuelTypeElectric = "electric"
)

// Permission constants for the Fuel module
const (
	PermissionFuelCreate = "fuel:create"
	PermissionFuelRead   = "fuel:read"
	PermissionFuelUpdate = "fuel:update"
	PermissionFuelDelete = "fuel:delete"
)

// Domain errors
var (
	ErrFuelRecordNotFound          = errors.New("fuel record not found")
	ErrFuelOdometerRegression      = errors.New("odometer reading cannot be lower than the previous recorded value")
	ErrFuelEditForbiddenAfter24h   = errors.New("fuel records can only be edited by owners or admins after 24 hours")
	ErrFuelVehicleNotInOrg         = errors.New("vehicle does not belong to this organization")
	ErrFuelDriverNotInOrg          = errors.New("driver does not belong to this organization")
	ErrFuelInvalidFuelType         = errors.New("invalid fuel type: must be one of gasoline, ethanol, diesel, gnv, electric")
	ErrFuelInvalidLiters           = errors.New("liters must be greater than zero")
	ErrFuelInvalidPricePerLiter    = errors.New("price per liter must be greater than zero")
	ErrFuelInvalidOdometerReading  = errors.New("odometer reading must be zero or greater")
	ErrFuelFutureDate              = errors.New("fuel record date cannot be more than 24 hours in the future")
)

// FuelRecord is the core transactional entity for a single fuel fill-up.
type FuelRecord struct {
	ID              string     `json:"id"`
	OrganizationID  string     `json:"organizationId"`
	VehicleID       string     `json:"vehicleId"`
	DriverID        *string    `json:"driverId,omitempty"`
	Date            time.Time  `json:"date"`
	Liters          float64    `json:"liters"`
	PricePerLiter   float64    `json:"pricePerLiter"`
	TotalCost       float64    `json:"totalCost"`
	OdometerReading int        `json:"odometerReading"`
	FuelType        string     `json:"fuelType"`
	StationName     *string    `json:"stationName,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	IsAnomaly       bool       `json:"isAnomaly"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       *time.Time `json:"deletedAt,omitempty"`
}

// FuelFilter holds optional query parameters for listing fuel records.
type FuelFilter struct {
	VehicleID string
	DriverID  string
	FuelType  string
	StartDate *time.Time
	EndDate   *time.Time
}

// FuelDashboard holds the KPI summary returned by GET /fuel/dashboard.
type FuelDashboard struct {
	TotalRecordsThisMonth  int     `json:"totalRecordsThisMonth"`
	TotalCostThisMonth     float64 `json:"totalCostThisMonth"`
	TotalLitersThisMonth   float64 `json:"totalLitersThisMonth"`
	AvgEfficiencyKmL       float64 `json:"avgEfficiencyKmL"`
	AvgCostPerLiter        float64 `json:"avgCostPerLiter"`
	AnomalyCount           int     `json:"anomalyCount"`
	CostVsLastMonth        float64 `json:"costVsLastMonth"` // percentage delta; positive = more expensive
	TopConsumingVehicleID  *string `json:"topConsumingVehicleId,omitempty"`
}

// FuelEfficiencyReport is a per-vehicle aggregation returned by GET /fuel/reports/efficiency.
// It is a computed value object — never persisted.
type FuelEfficiencyReport struct {
	VehicleID        string  `json:"vehicleId"`
	TotalLiters      float64 `json:"totalLiters"`
	TotalCost        float64 `json:"totalCost"`
	TotalKm          int     `json:"totalKm"`
	AvgEfficiencyKmL float64 `json:"avgEfficiencyKmL"` // 0 when < 2 records exist
	AvgCostPerKm     float64 `json:"avgCostPerKm"`     // 0 when < 2 records exist
	RecordCount      int     `json:"recordCount"`
}

// FuelRepository defines the data access contract for the fuel module.
type FuelRepository interface {
	// CRUD
	GetAll(ctx context.Context, orgID string, filter FuelFilter) ([]FuelRecord, error)
	GetByID(ctx context.Context, orgID string, id string) (FuelRecord, error)
	Create(ctx context.Context, orgID string, r FuelRecord) (FuelRecord, error)
	Update(ctx context.Context, orgID string, id string, r FuelRecord) (FuelRecord, error)
	Delete(ctx context.Context, orgID string, id string) error

	// Anomaly detection: return the last N records for a vehicle, ordered by date DESC
	GetLastNByVehicle(ctx context.Context, orgID string, vehicleID string, n int) ([]FuelRecord, error)

	// Efficiency report: per-vehicle aggregations within filter period
	GetEfficiencyReport(ctx context.Context, orgID string, filter FuelFilter) ([]FuelEfficiencyReport, error)

	// Dashboard: aggregated KPIs for the current and previous calendar month
	GetDashboardStats(ctx context.Context, orgID string) (FuelDashboard, error)
}
