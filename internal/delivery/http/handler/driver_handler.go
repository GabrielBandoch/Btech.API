package handler

import (
	"encoding/json"
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type DriverHandler struct {
	useCase usecase.DriverUseCase
}

// NewDriverHandler instantiates a new DriverHandler with dependencies injected.
func NewDriverHandler(useCase usecase.DriverUseCase) *DriverHandler {
	return &DriverHandler{
		useCase: useCase,
	}
}

// GetDrivers handles fetching all drivers.
func (h *DriverHandler) GetDrivers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	drivers, err := h.useCase.GetDrivers(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve drivers")
		return
	}

	driverDTOs := dto.FromDomainList(drivers)
	response.OK(w, driverDTOs)
}

// GetDriverByID handles fetching a single driver by their ID.
func (h *DriverHandler) GetDriverByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "driver ID is required")
		return
	}

	drv, err := h.useCase.GetDriverByID(ctx, id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "driver not found")
		return
	}

	response.OK(w, dto.FromDomain(drv))
}

// CreateDriver handles registering a new driver.
func (h *DriverHandler) CreateDriver(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	created, err := h.useCase.CreateDriver(ctx, req.ToDomain())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create driver")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.FromDomain(created), "")
}
