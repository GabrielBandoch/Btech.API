package handler

import (
	"encoding/json"
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/model"
	"github.com/btech/fleetcontrol-api/internal/repository"
)

type Handler struct {
	repo *repository.Repository
}

func NewHandler(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

// EnableCORS sets necessary CORS headers and handles preflight OPTIONS requests
func (h *Handler) EnableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

// GetTrips handles GET /api/trips
func (h *Handler) GetTrips(w http.ResponseWriter, r *http.Request) {
	trips := h.repo.GetTrips()
	h.respondJSON(w, http.StatusOK, trips)
}

// GetTripByID handles GET /api/trips/{id}
func (h *Handler) GetTripByID(w http.ResponseWriter, r *http.Request) {
	// Standard path parsing since we are using stdlib router
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Missing trip ID")
		return
	}

	trip, found := h.repo.GetTripByID(id)
	if !found {
		h.respondError(w, http.StatusNotFound, "Trip not found")
		return
	}

	h.respondJSON(w, http.StatusOK, trip)
}

// UpdateTrip handles PUT /api/trips/{id}
func (h *Handler) UpdateTrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Missing trip ID")
		return
	}

	var updates model.Trip
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	trip, updated := h.repo.UpdateTrip(id, updates)
	if !updated {
		h.respondError(w, http.StatusNotFound, "Trip not found")
		return
	}

	h.respondJSON(w, http.StatusOK, trip)
}

// GetDrivers handles GET /api/drivers
func (h *Handler) GetDrivers(w http.ResponseWriter, r *http.Request) {
	drivers := h.repo.GetDrivers()
	h.respondJSON(w, http.StatusOK, drivers)
}

// CreateDriver handles POST /api/drivers
func (h *Handler) CreateDriver(w http.ResponseWriter, r *http.Request) {
	var newDriver model.Driver
	if err := json.NewDecoder(r.Body).Decode(&newDriver); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if newDriver.Name == "" {
		h.respondError(w, http.StatusBadRequest, "Name is required")
		return
	}

	created := h.repo.CreateDriver(newDriver)
	h.respondJSON(w, http.StatusCreated, created)
}

// GetIncidents handles GET /api/incidents
func (h *Handler) GetIncidents(w http.ResponseWriter, r *http.Request) {
	incidents := h.repo.GetIncidents()
	h.respondJSON(w, http.StatusOK, incidents)
}

// UpdateIncident handles PUT /api/incidents/{id}
func (h *Handler) UpdateIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "Missing incident ID")
		return
	}

	var updates model.Incident
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	inc, updated := h.repo.UpdateIncident(id, updates)
	if !updated {
		h.respondError(w, http.StatusNotFound, "Incident not found")
		return
	}

	h.respondJSON(w, http.StatusOK, inc)
}
