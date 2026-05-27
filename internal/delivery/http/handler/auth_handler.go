package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

type AuthHandler struct {
	authUseCase usecase.AuthUseCase
}

// NewAuthHandler instantiates a new AuthHandler.
func NewAuthHandler(authUseCase usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

// Register handles registering a new user.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authUseCase.RegisterUser(r.Context(), req.Name, req.Email, req.Password, req.Role)
	if err != nil {
		if errors.Is(err, usecase.ErrEmailExists) || errors.Is(err, usecase.ErrInvalidRole) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	org, err := h.authUseCase.GetOrganizationByID(r.Context(), user.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve organization details")
		return
	}

	response.JSON(w, http.StatusCreated, true, dto.UserToResponse(user, org.Name, org.Slug), "")
}

// Login handles authenticating a user.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user, token, err := h.authUseCase.LoginUser(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}

	org, err := h.authUseCase.GetOrganizationByID(r.Context(), user.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve organization details")
		return
	}

	resp := dto.AuthResponse{
		User:  dto.UserToResponse(user, org.Name, org.Slug),
		Token: token,
	}
	response.OK(w, resp)
}

// Me retrieves the authenticated user details from request context.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	org, err := h.authUseCase.GetOrganizationByID(r.Context(), user.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve organization details")
		return
	}

	response.OK(w, dto.UserToResponse(user, org.Name, org.Slug))
}
