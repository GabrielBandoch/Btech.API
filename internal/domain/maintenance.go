package domain

import (
	"context"
	"time"
)

// Maintenance Types, Priorities, and Statuses
const (
	MaintenanceTypePreventive = "preventive"
	MaintenanceTypeCorrective = "corrective"

	MaintenancePriorityLow      = "low"
	MaintenancePriorityMedium   = "medium"
	MaintenancePriorityHigh     = "high"
	MaintenancePriorityCritical = "critical"

	MaintenanceStatusScheduled  = "scheduled"
	MaintenanceStatusInProgress = "in_progress"
	MaintenanceStatusCompleted  = "completed"
	MaintenanceStatusCanceled   = "canceled"

	MaintenanceAlertTypeMileage = "mileage_due"
	MaintenanceAlertTypeDate    = "date_due"

	MaintenanceAlertStatusActive   = "active"
	MaintenanceAlertStatusResolved = "resolved"
)

// Permission Constants for Maintenance
const (
	PermissionMaintenanceCreate = "maintenance:create"
	PermissionMaintenanceRead   = "maintenance:read"
	PermissionMaintenanceUpdate = "maintenance:update"
	PermissionMaintenanceDelete = "maintenance:delete"
)

// MaintenanceSupplier represents a workshop/supplier
type MaintenanceSupplier struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	Name           string     `json:"name"`
	Phone          string     `json:"phone,omitempty"`
	Email          string     `json:"email,omitempty"`
	Address        string     `json:"address,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}

// MaintenancePlan represents a preventive plan configuration
type MaintenancePlan struct {
	ID                  string     `json:"id"`
	OrganizationID      string     `json:"organizationId"`
	VehicleID           string     `json:"vehicleId"`
	Name                string     `json:"name"`
	IntervalKM          *int       `json:"intervalKm,omitempty"`
	IntervalMonths      *int       `json:"intervalMonths,omitempty"`
	LastMaintenanceKM   *int       `json:"lastMaintenanceKm,omitempty"`
	LastMaintenanceDate *time.Time `json:"lastMaintenanceDate,omitempty"`
	NextDueKM           *int       `json:"nextDueKm,omitempty"`
	NextDueDate         *time.Time `json:"nextDueDate,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	DeletedAt           *time.Time `json:"deletedAt,omitempty"`
}

// Maintenance represents a corrective/preventive record
type Maintenance struct {
	ID                string     `json:"id"`
	OrganizationID    string     `json:"organizationId"`
	VehicleID         string     `json:"vehicleId"`
	MaintenancePlanID *string    `json:"maintenancePlanId,omitempty"`
	SupplierID        *string    `json:"supplierId,omitempty"`
	Type              string     `json:"type"` // preventive, corrective
	Priority          string     `json:"priority"` // low, medium, high, critical
	Status            string     `json:"status"` // scheduled, in_progress, completed, canceled
	Date              time.Time  `json:"date"`
	OdometerAtService int        `json:"odometerAtService"`
	DowntimeHours     float64    `json:"downtimeHours"`
	Cost              float64    `json:"cost"`
	Description       string     `json:"description,omitempty"`
	Attachments       []string   `json:"attachments"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	DeletedAt         *time.Time `json:"deletedAt,omitempty"`
}

// MaintenanceAlert represents a notification for due/overdue preventive plans
type MaintenanceAlert struct {
	ID                string     `json:"id"`
	OrganizationID    string     `json:"organizationId"`
	VehicleID         string     `json:"vehicleId"`
	MaintenancePlanID string     `json:"maintenancePlanId"`
	Type              string     `json:"type"` // mileage_due, date_due
	Title             string     `json:"title"`
	Message           string     `json:"message"`
	Status            string     `json:"status"` // active, resolved
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	DeletedAt         *time.Time `json:"deletedAt,omitempty"`
}

// MaintenanceFilter specifies search parameters
type MaintenanceFilter struct {
	VehicleID  string
	Type       string
	Status     string
	SupplierID string
	Priority   string
	StartDate  *time.Time
	EndDate    *time.Time
}

// Repository Interfaces
type MaintenanceSupplierRepository interface {
	GetAll(ctx context.Context, orgID string) ([]MaintenanceSupplier, error)
	GetByID(ctx context.Context, orgID string, id string) (MaintenanceSupplier, error)
	Create(ctx context.Context, orgID string, supplier MaintenanceSupplier) (MaintenanceSupplier, error)
	Update(ctx context.Context, orgID string, id string, supplier MaintenanceSupplier) (MaintenanceSupplier, error)
	Delete(ctx context.Context, orgID string, id string) error
}

type MaintenancePlanRepository interface {
	GetAll(ctx context.Context, orgID string, vehicleID string) ([]MaintenancePlan, error)
	GetByID(ctx context.Context, orgID string, id string) (MaintenancePlan, error)
	Create(ctx context.Context, orgID string, plan MaintenancePlan) (MaintenancePlan, error)
	Update(ctx context.Context, orgID string, id string, plan MaintenancePlan) (MaintenancePlan, error)
	Delete(ctx context.Context, orgID string, id string) error
}

type MaintenanceRepository interface {
	GetAll(ctx context.Context, orgID string, filter MaintenanceFilter) ([]Maintenance, error)
	GetByID(ctx context.Context, orgID string, id string) (Maintenance, error)
	Create(ctx context.Context, orgID string, m Maintenance) (Maintenance, error)
	Update(ctx context.Context, orgID string, id string, m Maintenance) (Maintenance, error)
	Delete(ctx context.Context, orgID string, id string) error
	GetCostReport(ctx context.Context, orgID string, filter MaintenanceFilter) (map[string]interface{}, error)
}

type MaintenanceAlertRepository interface {
	GetAll(ctx context.Context, orgID string, status string) ([]MaintenanceAlert, error)
	GetByID(ctx context.Context, orgID string, id string) (MaintenanceAlert, error)
	Create(ctx context.Context, orgID string, alert MaintenanceAlert) (MaintenanceAlert, error)
	UpdateStatus(ctx context.Context, orgID string, id string, status string) error
	ResolveByPlanID(ctx context.Context, orgID string, planID string) error
	GetActiveByPlanID(ctx context.Context, orgID string, planID string) ([]MaintenanceAlert, error)
}
