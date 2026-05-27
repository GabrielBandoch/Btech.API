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

type TripHandler struct {
	useCase usecase.TripUseCase
}

// NewTripHandler instantiates a new TripHandler.
func NewTripHandler(useCase usecase.TripUseCase) *TripHandler {
	return &TripHandler{
		useCase: useCase,
	}
}

// GetTrips handles fetching all trips.
func (h *TripHandler) GetTrips(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	trips, err := h.useCase.GetTrips(ctx, orgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve trips")
		return
	}

	response.OK(w, dto.TripFromDomainList(trips))
}

// GetTripByID handles fetching a single trip by ID.
func (h *TripHandler) GetTripByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "trip ID is required")
		return
	}

	trip, err := h.useCase.GetTripByID(ctx, orgID, id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "trip not found")
		return
	}

	response.OK(w, dto.TripFromDomain(trip))
}

// UpdateTrip handles updating a trip.
func (h *TripHandler) UpdateTrip(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, ok := middleware.OrganizationIDFromContext(ctx)
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized: organization context missing")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "trip ID is required")
		return
	}

	var req dto.UpdateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.useCase.UpdateTrip(ctx, orgID, id, req.ToDomain())
	if err != nil {
		response.Error(w, http.StatusNotFound, "trip not found")
		return
	}

	response.OK(w, dto.TripFromDomain(updated))
}
