package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type MaintenanceHandler struct {
	useCase usecase.MaintenanceUseCase
}

func NewMaintenanceHandler(useCase usecase.MaintenanceUseCase) *MaintenanceHandler {
	return &MaintenanceHandler{
		useCase: useCase,
	}
}

// ----------------------------------------------------
// Suppliers Endpoints
// ----------------------------------------------------

func (h *MaintenanceHandler) GetSuppliers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	suppliers, err := h.useCase.GetSuppliers(ctx, orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve suppliers")
		return
	}

	response.OK(w, dto.SupplierToResponseList(suppliers))
}

func (h *MaintenanceHandler) GetSupplierByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "supplier ID is required")
		return
	}

	supplier, err := h.useCase.GetSupplierByID(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "supplier not found")
		return
	}

	response.OK(w, dto.SupplierToResponse(supplier))
}

func (h *MaintenanceHandler) CreateSupplier(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	var req dto.CreateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	domainSupplier := domain.MaintenanceSupplier{
		Name:    req.Name,
		Phone:   req.Phone,
		Email:   req.Email,
		Address: req.Address,
	}

	created, err := h.useCase.CreateSupplier(ctx, orgID, domainSupplier)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create supplier")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.SupplierToResponse(created), "supplier created successfully")
}

func (h *MaintenanceHandler) UpdateSupplier(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "supplier ID is required")
		return
	}

	var req dto.UpdateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	domainSupplier := domain.MaintenanceSupplier{
		Name:    req.Name,
		Phone:   req.Phone,
		Email:   req.Email,
		Address: req.Address,
	}

	updated, err := h.useCase.UpdateSupplier(ctx, orgID, id, domainSupplier)
	if err != nil {
		response.Error(w, http.StatusNotFound, "supplier not found")
		return
	}

	response.OK(w, dto.SupplierToResponse(updated))
}

func (h *MaintenanceHandler) DeleteSupplier(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "supplier ID is required")
		return
	}

	err := h.useCase.DeleteSupplier(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete supplier")
		return
	}

	response.OK(w, map[string]string{"message": "supplier deleted successfully"})
}

// ----------------------------------------------------
// Plans Endpoints
// ----------------------------------------------------

func (h *MaintenanceHandler) GetPlans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	vehicleID := r.URL.Query().Get("vehicleId")

	plans, err := h.useCase.GetPlans(ctx, orgID, vehicleID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve plans")
		return
	}

	response.OK(w, dto.PlanToResponseList(plans))
}

func (h *MaintenanceHandler) GetPlanByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "plan ID is required")
		return
	}

	plan, err := h.useCase.GetPlanByID(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "plan not found")
		return
	}

	response.OK(w, dto.PlanToResponse(plan))
}

func (h *MaintenanceHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	var req dto.CreatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.VehicleID == "" || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "vehicleId and name are required")
		return
	}

	domainPlan := domain.MaintenancePlan{
		VehicleID:           req.VehicleID,
		Name:                req.Name,
		IntervalKM:          req.IntervalKM,
		IntervalMonths:      req.IntervalMonths,
		LastMaintenanceKM:   req.LastKM,
		LastMaintenanceDate: req.LastDate,
	}

	created, err := h.useCase.CreatePlan(ctx, orgID, domainPlan)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create maintenance plan")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.PlanToResponse(created), "plan created successfully")
}

func (h *MaintenanceHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "plan ID is required")
		return
	}

	var req dto.UpdatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	domainPlan := domain.MaintenancePlan{
		Name:                req.Name,
		IntervalKM:          req.IntervalKM,
		IntervalMonths:      req.IntervalMonths,
		LastMaintenanceKM:   req.LastKM,
		LastMaintenanceDate: req.LastDate,
	}

	updated, err := h.useCase.UpdatePlan(ctx, orgID, id, domainPlan)
	if err != nil {
		response.Error(w, http.StatusNotFound, "plan not found")
		return
	}

	response.OK(w, dto.PlanToResponse(updated))
}

func (h *MaintenanceHandler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "plan ID is required")
		return
	}

	err := h.useCase.DeletePlan(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete plan")
		return
	}

	response.OK(w, map[string]string{"message": "plan deleted successfully"})
}

// ----------------------------------------------------
// Records Endpoints
// ----------------------------------------------------

func (h *MaintenanceHandler) GetMaintenances(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	q := r.URL.Query()
	filter := domain.MaintenanceFilter{
		VehicleID:  q.Get("vehicleId"),
		Type:       q.Get("type"),
		Status:     q.Get("status"),
		SupplierID: q.Get("supplierId"),
		Priority:   q.Get("priority"),
	}

	if startStr := q.Get("startDate"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartDate = &t
		}
	}
	if endStr := q.Get("endDate"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndDate = &t
		}
	}

	records, err := h.useCase.GetMaintenances(ctx, orgID, filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve maintenance records")
		return
	}

	response.OK(w, dto.MaintenanceToResponseList(records))
}

func (h *MaintenanceHandler) GetMaintenanceByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "maintenance ID is required")
		return
	}

	m, err := h.useCase.GetMaintenanceByID(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "maintenance record not found")
		return
	}

	response.OK(w, dto.MaintenanceToResponse(m))
}

func (h *MaintenanceHandler) CreateMaintenance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	var req dto.CreateMaintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.VehicleID == "" || req.Type == "" || req.Status == "" || req.Priority == "" {
		response.Error(w, http.StatusBadRequest, "vehicleId, type, priority, and status are required")
		return
	}

	if req.Date.IsZero() {
		req.Date = time.Now()
	}

	domainM := domain.Maintenance{
		VehicleID:         req.VehicleID,
		MaintenancePlanID: req.MaintenancePlanID,
		SupplierID:        req.SupplierID,
		Type:              req.Type,
		Priority:          req.Priority,
		Status:            req.Status,
		Date:              req.Date,
		OdometerAtService: req.OdometerAtService,
		DowntimeHours:     req.DowntimeHours,
		Cost:              req.Cost,
		Description:       req.Description,
		Attachments:       req.Attachments,
	}

	created, err := h.useCase.CreateMaintenance(ctx, orgID, domainM)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create maintenance record")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.MaintenanceToResponse(created), "maintenance recorded successfully")
}

func (h *MaintenanceHandler) UpdateMaintenance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "maintenance ID is required")
		return
	}

	var req dto.UpdateMaintenanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	domainM := domain.Maintenance{
		VehicleID:         req.VehicleID,
		MaintenancePlanID: req.MaintenancePlanID,
		SupplierID:        req.SupplierID,
		Type:              req.Type,
		Priority:          req.Priority,
		Status:            req.Status,
		Description:       req.Description,
		Attachments:       req.Attachments,
	}

	if req.Date != nil {
		domainM.Date = *req.Date
	}
	if req.OdometerAtService != nil {
		domainM.OdometerAtService = *req.OdometerAtService
	}
	if req.DowntimeHours != nil {
		domainM.DowntimeHours = *req.DowntimeHours
	}
	if req.Cost != nil {
		domainM.Cost = *req.Cost
	}

	updated, err := h.useCase.UpdateMaintenance(ctx, orgID, id, domainM)
	if err != nil {
		response.Error(w, http.StatusNotFound, "maintenance not found")
		return
	}

	response.OK(w, dto.MaintenanceToResponse(updated))
}

func (h *MaintenanceHandler) DeleteMaintenance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "maintenance ID is required")
		return
	}

	err := h.useCase.DeleteMaintenance(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to delete maintenance")
		return
	}

	response.OK(w, map[string]string{"message": "maintenance deleted successfully"})
}

// ----------------------------------------------------
// Alerts Endpoints
// ----------------------------------------------------

func (h *MaintenanceHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	status := r.URL.Query().Get("status")

	alerts, err := h.useCase.GetAlerts(ctx, orgID, status)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve alerts")
		return
	}

	response.OK(w, dto.AlertToResponseList(alerts))
}

func (h *MaintenanceHandler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "alert ID is required")
		return
	}

	err := h.useCase.ResolveAlert(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to resolve alert")
		return
	}

	response.OK(w, map[string]string{"message": "alert resolved successfully"})
}

// ----------------------------------------------------
// Dashboard & Cost Reports
// ----------------------------------------------------

func (h *MaintenanceHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	dash, err := h.useCase.GetDashboard(ctx, orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to load dashboard")
		return
	}

	response.OK(w, dash)
}

func (h *MaintenanceHandler) GetCostReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	q := r.URL.Query()
	filter := domain.MaintenanceFilter{
		VehicleID:  q.Get("vehicleId"),
		SupplierID: q.Get("supplierId"),
	}

	if startStr := q.Get("startDate"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartDate = &t
		}
	}
	if endStr := q.Get("endDate"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndDate = &t
		}
	}

	report, err := h.useCase.GetCostReport(ctx, orgID, filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to generate cost report")
		return
	}

	response.OK(w, report)
}
