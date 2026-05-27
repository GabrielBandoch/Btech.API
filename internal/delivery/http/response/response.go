package response

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// JSON sends a standardized API response with appropriate HTTP headers and status code.
func JSON(w http.ResponseWriter, status int, success bool, data interface{}, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := APIResponse{
		Success: success,
		Data:    data,
		Error:   errMsg,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// OK is a helper to write a successful JSON response with a standard envelope.
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, true, data, "")
}

// Error is a helper to write a failure JSON response with standard error envelope.
func Error(w http.ResponseWriter, status int, errMsg string) {
	JSON(w, status, false, nil, errMsg)
}
