package handler

import (
	"encoding/json"
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/usecase"
	"github.com/go-chi/chi/v5"
)

type IncidentHandler struct {
	useCase usecase.IncidentUseCase
}

// NewIncidentHandler instantiates a new IncidentHandler.
func NewIncidentHandler(useCase usecase.IncidentUseCase) *IncidentHandler {
	return &IncidentHandler{
		useCase: useCase,
	}
}

// GetIncidents handles fetching all incidents.
func (h *IncidentHandler) GetIncidents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	incidents, err := h.useCase.GetIncidents(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve incidents")
		return
	}

	response.OK(w, dto.IncidentFromDomainList(incidents))
}

// CreateIncident handles registering a new incident.
func (h *IncidentHandler) CreateIncident(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.useCase.CreateIncident(ctx, req.ToDomain())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to create incident")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.IncidentFromDomain(created), "")
}

// UpdateIncident handles updating an incident.
func (h *IncidentHandler) UpdateIncident(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "incident ID is required")
		return
	}

	var req dto.UpdateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.useCase.UpdateIncident(ctx, id, req.ToDomain())
	if err != nil {
		response.Error(w, http.StatusNotFound, "incident not found")
		return
	}

	response.OK(w, dto.IncidentFromDomain(updated))
}
