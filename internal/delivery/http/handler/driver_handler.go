package handler

import (
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/usecase"
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

// GetDrivers handles fetching all drivers, converting them to DTOs and replying with standard envelope.
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
