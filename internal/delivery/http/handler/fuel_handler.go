package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/usecase"
	"github.com/go-chi/chi/v5"
)

// FuelHandler holds the HTTP handlers for all fuel module endpoints.
type FuelHandler struct {
	useCase usecase.FuelUseCase
}

func NewFuelHandler(useCase usecase.FuelUseCase) *FuelHandler {
	return &FuelHandler{useCase: useCase}
}

// --------------------------------------------------------------------------
// GET /fuel/records
// --------------------------------------------------------------------------
func (h *FuelHandler) GetRecords(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	filter := parseFuelFilter(r)

	records, err := h.useCase.GetRecords(ctx, orgID, filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve fuel records")
		return
	}

	response.OK(w, dto.FuelRecordToResponseList(records))
}

// --------------------------------------------------------------------------
// GET /fuel/records/{id}
// --------------------------------------------------------------------------
func (h *FuelHandler) GetRecordByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "fuel record ID is required")
		return
	}

	record, err := h.useCase.GetByID(ctx, orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrFuelRecordNotFound) {
			response.Error(w, http.StatusNotFound, "fuel record not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to retrieve fuel record")
		return
	}

	response.OK(w, dto.FuelRecordToResponse(record))
}

// --------------------------------------------------------------------------
// POST /fuel/records
// --------------------------------------------------------------------------
func (h *FuelHandler) CreateRecord(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	var req dto.CreateFuelRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Required field validation (handler-level: presence only)
	if req.VehicleID == "" {
		response.Error(w, http.StatusBadRequest, "vehicleId is required")
		return
	}
	if req.Date == "" {
		response.Error(w, http.StatusBadRequest, "date is required")
		return
	}
	if req.FuelType == "" {
		response.Error(w, http.StatusBadRequest, "fuelType is required")
		return
	}

	// Parse date string
	parsedDate, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "date must be a valid RFC3339 timestamp")
		return
	}

	// Map to domain struct
	rec := domain.FuelRecord{
		VehicleID:       req.VehicleID,
		Date:            parsedDate,
		Liters:          req.Liters,
		PricePerLiter:   req.PricePerLiter,
		OdometerReading: req.OdometerReading,
		FuelType:        req.FuelType,
	}
	if req.DriverID != "" {
		rec.DriverID = &req.DriverID
	}
	if req.StationName != "" {
		rec.StationName = &req.StationName
	}
	if req.Notes != "" {
		rec.Notes = &req.Notes
	}

	created, err := h.useCase.CreateRecord(ctx, orgID, rec)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrFuelInvalidFuelType):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelInvalidLiters):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelInvalidPricePerLiter):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelInvalidOdometerReading):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelFutureDate):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelOdometerRegression):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelVehicleNotInOrg):
			response.Error(w, http.StatusBadRequest, "vehicle not found in this organization")
		case errors.Is(err, domain.ErrFuelDriverNotInOrg):
			response.Error(w, http.StatusBadRequest, "driver not found in this organization")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to create fuel record")
		}
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.FuelRecordToResponse(created), "fuel record created successfully")
}

// --------------------------------------------------------------------------
// PUT /fuel/records/{id}
// --------------------------------------------------------------------------
func (h *FuelHandler) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "fuel record ID is required")
		return
	}

	// Extract user role for the 24h edit restriction
	userRole, _ := middleware.RoleFromContext(ctx)

	var req dto.UpdateFuelRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Parse date if provided
	var parsedDate time.Time
	if req.Date != "" {
		d, err := time.Parse(time.RFC3339, req.Date)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "date must be a valid RFC3339 timestamp")
			return
		}
		parsedDate = d
	}

	// Map to domain struct (use case reads existing values for unset fields)
	rec := domain.FuelRecord{
		VehicleID:       req.VehicleID,
		Date:            parsedDate,
		Liters:          req.Liters,
		PricePerLiter:   req.PricePerLiter,
		OdometerReading: req.OdometerReading,
		FuelType:        req.FuelType,
	}
	if req.DriverID != "" {
		rec.DriverID = &req.DriverID
	}
	if req.StationName != "" {
		rec.StationName = &req.StationName
	}
	if req.Notes != "" {
		rec.Notes = &req.Notes
	}

	updated, err := h.useCase.UpdateRecord(ctx, orgID, id, rec, userRole)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrFuelRecordNotFound):
			response.Error(w, http.StatusNotFound, "fuel record not found")
		case errors.Is(err, domain.ErrFuelEditForbiddenAfter24h):
			response.Error(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrFuelOdometerRegression):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelInvalidFuelType):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelInvalidLiters):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelInvalidPricePerLiter):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelFutureDate):
			response.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrFuelVehicleNotInOrg):
			response.Error(w, http.StatusBadRequest, "vehicle not found in this organization")
		case errors.Is(err, domain.ErrFuelDriverNotInOrg):
			response.Error(w, http.StatusBadRequest, "driver not found in this organization")
		default:
			response.Error(w, http.StatusInternalServerError, "failed to update fuel record")
		}
		return
	}

	response.OK(w, dto.FuelRecordToResponse(updated))
}

// --------------------------------------------------------------------------
// DELETE /fuel/records/{id}
// --------------------------------------------------------------------------
func (h *FuelHandler) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "fuel record ID is required")
		return
	}

	if err := h.useCase.DeleteRecord(ctx, orgID, id); err != nil {
		if errors.Is(err, domain.ErrFuelRecordNotFound) {
			response.Error(w, http.StatusNotFound, "fuel record not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to delete fuel record")
		return
	}

	response.OK(w, map[string]string{"message": "fuel record deleted successfully"})
}

// --------------------------------------------------------------------------
// GET /fuel/dashboard
// --------------------------------------------------------------------------
func (h *FuelHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	dashboard, err := h.useCase.GetDashboard(ctx, orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve fuel dashboard")
		return
	}

	response.OK(w, dto.FuelDashboardToResponse(dashboard))
}

// --------------------------------------------------------------------------
// GET /fuel/reports/efficiency
// --------------------------------------------------------------------------
func (h *FuelHandler) GetEfficiencyReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	filter := parseFuelFilter(r)

	reports, err := h.useCase.GetEfficiencyReport(ctx, orgID, filter)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve efficiency report")
		return
	}

	response.OK(w, dto.FuelEfficiencyReportToResponseList(reports))
}

// --------------------------------------------------------------------------
// parseFuelFilter extracts optional query parameters into a FuelFilter.
// All parameters are camelCase per API standards.
// --------------------------------------------------------------------------
func parseFuelFilter(r *http.Request) domain.FuelFilter {
	q := r.URL.Query()
	filter := domain.FuelFilter{
		VehicleID: q.Get("vehicleId"),
		DriverID:  q.Get("driverId"),
		FuelType:  q.Get("fuelType"),
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

	return filter
}
