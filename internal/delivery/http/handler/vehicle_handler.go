package handler

import (
	"encoding/json"
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type VehicleHandler struct {
	useCase usecase.VehicleUseCase
}

// NewVehicleHandler instantiates a new VehicleHandler.
func NewVehicleHandler(useCase usecase.VehicleUseCase) *VehicleHandler {
	return &VehicleHandler{
		useCase: useCase,
	}
}

// GetVehicles handles fetching all vehicles for an organization.
func (h *VehicleHandler) GetVehicles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	vehicles, err := h.useCase.GetVehicles(ctx, orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve vehicles")
		return
	}

	response.OK(w, dto.VehicleFromDomainList(vehicles))
}

// GetVehicleByID handles fetching a single vehicle by ID.
func (h *VehicleHandler) GetVehicleByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "vehicle ID is required")
		return
	}

	vehicle, err := h.useCase.GetVehicleByID(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "vehicle not found")
		return
	}

	response.OK(w, dto.VehicleFromDomain(vehicle))
}

// CreateVehicle handles registering a new vehicle.
func (h *VehicleHandler) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	var req dto.CreateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Brand == "" || req.Model == "" {
		response.Error(w, http.StatusBadRequest, "brand and model are required")
		return
	}

	created, err := h.useCase.CreateVehicle(ctx, orgID, req.ToDomain())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create vehicle")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.VehicleFromDomain(created), "")
}

// UpdateVehicle handles modifying an existing vehicle's properties.
func (h *VehicleHandler) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "vehicle ID is required")
		return
	}

	var req dto.UpdateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.useCase.UpdateVehicle(ctx, orgID, id, req.ToDomain())
	if err != nil {
		response.Error(w, http.StatusNotFound, "vehicle not found")
		return
	}

	response.OK(w, dto.VehicleFromDomain(updated))
}
