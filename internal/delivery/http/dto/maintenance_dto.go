package dto

import (
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

// Suppliers
type CreateSupplierRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone,omitempty"`
	Email   string `json:"email,omitempty"`
	Address string `json:"address,omitempty"`
}

type UpdateSupplierRequest struct {
	Name    string `json:"name,omitempty"`
	Phone   string `json:"phone,omitempty"`
	Email   string `json:"email,omitempty"`
	Address string `json:"address,omitempty"`
}

type SupplierResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone,omitempty"`
	Email     string    `json:"email,omitempty"`
	Address   string    `json:"address,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func SupplierToResponse(s domain.MaintenanceSupplier) SupplierResponse {
	return SupplierResponse{
		ID:        s.ID,
		Name:      s.Name,
		Phone:     s.Phone,
		Email:     s.Email,
		Address:   s.Address,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func SupplierToResponseList(list []domain.MaintenanceSupplier) []SupplierResponse {
	res := make([]SupplierResponse, len(list))
	for i, s := range list {
		res[i] = SupplierToResponse(s)
	}
	return res
}

// Plans
type CreatePlanRequest struct {
	VehicleID      string     `json:"vehicleId"`
	Name           string     `json:"name"`
	IntervalKM     *int       `json:"intervalKm,omitempty"`
	IntervalMonths *int       `json:"intervalMonths,omitempty"`
	LastKM         *int       `json:"lastMaintenanceKm,omitempty"`
	LastDate       *time.Time `json:"lastMaintenanceDate,omitempty"`
}

type UpdatePlanRequest struct {
	Name           string     `json:"name,omitempty"`
	IntervalKM     *int       `json:"intervalKm,omitempty"`
	IntervalMonths *int       `json:"intervalMonths,omitempty"`
	LastKM         *int       `json:"lastMaintenanceKm,omitempty"`
	LastDate       *time.Time `json:"lastMaintenanceDate,omitempty"`
}

type PlanResponse struct {
	ID                  string     `json:"id"`
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
}

func PlanToResponse(p domain.MaintenancePlan) PlanResponse {
	return PlanResponse{
		ID:                  p.ID,
		VehicleID:           p.VehicleID,
		Name:                p.Name,
		IntervalKM:          p.IntervalKM,
		IntervalMonths:      p.IntervalMonths,
		LastMaintenanceKM:   p.LastMaintenanceKM,
		LastMaintenanceDate: p.LastMaintenanceDate,
		NextDueKM:           p.NextDueKM,
		NextDueDate:         p.NextDueDate,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
}

func PlanToResponseList(list []domain.MaintenancePlan) []PlanResponse {
	res := make([]PlanResponse, len(list))
	for i, p := range list {
		res[i] = PlanToResponse(p)
	}
	return res
}

// Maintenances
type CreateMaintenanceRequest struct {
	VehicleID         string    `json:"vehicleId"`
	MaintenancePlanID *string   `json:"maintenancePlanId,omitempty"`
	SupplierID        *string   `json:"supplierId,omitempty"`
	Type              string    `json:"type"` // preventive, corrective
	Priority          string    `json:"priority"` // low, medium, high, critical
	Status            string    `json:"status"` // scheduled, in_progress, completed, canceled
	Date              time.Time `json:"date"`
	OdometerAtService int       `json:"odometerAtService"`
	DowntimeHours     float64   `json:"downtimeHours"`
	Cost              float64   `json:"cost"`
	Description       string    `json:"description,omitempty"`
	Attachments       []string  `json:"attachments,omitempty"`
}

type UpdateMaintenanceRequest struct {
	VehicleID         string     `json:"vehicleId,omitempty"`
	MaintenancePlanID *string    `json:"maintenancePlanId,omitempty"`
	SupplierID        *string    `json:"supplierId,omitempty"`
	Type              string     `json:"type,omitempty"`
	Priority          string     `json:"priority,omitempty"`
	Status            string     `json:"status,omitempty"`
	Date              *time.Time `json:"date,omitempty"`
	OdometerAtService *int       `json:"odometerAtService,omitempty"`
	DowntimeHours     *float64   `json:"downtimeHours,omitempty"`
	Cost              *float64   `json:"cost,omitempty"`
	Description       string     `json:"description,omitempty"`
	Attachments       []string   `json:"attachments,omitempty"`
}

type MaintenanceResponse struct {
	ID                string     `json:"id"`
	VehicleID         string     `json:"vehicleId"`
	MaintenancePlanID *string    `json:"maintenancePlanId,omitempty"`
	SupplierID        *string    `json:"supplierId,omitempty"`
	Type              string     `json:"type"`
	Priority          string     `json:"priority"`
	Status            string     `json:"status"`
	Date              time.Time  `json:"date"`
	OdometerAtService int        `json:"odometerAtService"`
	DowntimeHours     float64    `json:"downtimeHours"`
	Cost              float64    `json:"cost"`
	Description       string     `json:"description,omitempty"`
	Attachments       []string   `json:"attachments"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func MaintenanceToResponse(m domain.Maintenance) MaintenanceResponse {
	attachments := m.Attachments
	if attachments == nil {
		attachments = []string{}
	}
	return MaintenanceResponse{
		ID:                m.ID,
		VehicleID:         m.VehicleID,
		MaintenancePlanID: m.MaintenancePlanID,
		SupplierID:        m.SupplierID,
		Type:              m.Type,
		Priority:          m.Priority,
		Status:            m.Status,
		Date:              m.Date,
		OdometerAtService: m.OdometerAtService,
		DowntimeHours:     m.DowntimeHours,
		Cost:              m.Cost,
		Description:       m.Description,
		Attachments:       attachments,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func MaintenanceToResponseList(list []domain.Maintenance) []MaintenanceResponse {
	res := make([]MaintenanceResponse, len(list))
	for i, m := range list {
		res[i] = MaintenanceToResponse(m)
	}
	return res
}

// Alerts
type AlertResponse struct {
	ID                string    `json:"id"`
	VehicleID         string    `json:"vehicleId"`
	MaintenancePlanID string    `json:"maintenancePlanId"`
	Type              string    `json:"type"`
	Title             string    `json:"title"`
	Message           string    `json:"message"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"createdAt"`
}

func AlertToResponse(a domain.MaintenanceAlert) AlertResponse {
	return AlertResponse{
		ID:                a.ID,
		VehicleID:         a.VehicleID,
		MaintenancePlanID: a.MaintenancePlanID,
		Type:              a.Type,
		Title:             a.Title,
		Message:           a.Message,
		Status:            a.Status,
		CreatedAt:         a.CreatedAt,
	}
}

func AlertToResponseList(list []domain.MaintenanceAlert) []AlertResponse {
	res := make([]AlertResponse, len(list))
	for i, a := range list {
		res[i] = AlertToResponse(a)
	}
	return res
}
