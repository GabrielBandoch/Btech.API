package handler

import (
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HealthHandler returns the API running status.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:  "online",
		Message: "FleetControl API is running",
	}
	response.OK(w, resp)
}
