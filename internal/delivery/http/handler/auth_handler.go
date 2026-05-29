package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/domain"
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

	user, token, refreshToken, err := h.authUseCase.LoginUser(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to authenticate")
		return
	}

	// Set refresh token cookie (HttpOnly, Secure, SameSite=Lax)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   int(7 * 24 * time.Hour / time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

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

// Refresh handles token rotation using the refresh token cookie.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		response.Error(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	ip := ""
	if val, ok := r.Context().Value(domain.ClientIPContextKey).(string); ok {
		ip = val
	}
	ua := ""
	if val, ok := r.Context().Value(domain.UserAgentContextKey).(string); ok {
		ua = val
	}

	user, newAccessToken, newRefreshToken, err := h.authUseCase.RefreshTokenRotation(r.Context(), cookie.Value, ip, ua)
	if err != nil {
		// Clear cookie on error (especially compromise) to prevent retry loops
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		response.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	// Set rotated refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Path:     "/",
		MaxAge:   int(7 * 24 * time.Hour / time.Second),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	org, err := h.authUseCase.GetOrganizationByID(r.Context(), user.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to retrieve organization details")
		return
	}

	resp := dto.AuthResponse{
		User:  dto.UserToResponse(user, org.Name, org.Slug),
		Token: newAccessToken,
	}
	response.OK(w, resp)
}

// Logout terminates the user's session and clears the cookie.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	var tokenVal string
	if err == nil {
		tokenVal = cookie.Value
	}

	// Perform logout (resilient: returns success even if token is missing/invalid)
	_ = h.authUseCase.LogoutUser(r.Context(), tokenVal)

	// Clear the refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	response.JSON(w, http.StatusOK, true, map[string]string{"message": "successfully logged out"}, "")
}

// ListSessions lists active sessions for the authenticated user.
func (h *AuthHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Try to identify current session ID by parsing refresh token
	currentSessionID := ""
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		if claims, err := h.authUseCase.ParseRefreshToken(r.Context(), cookie.Value); err == nil {
			currentSessionID = claims.SessionID
		}
	}

	sessions, err := h.authUseCase.ListSessions(r.Context(), user.ID, user.OrganizationID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	response.OK(w, dto.SessionsToResponseList(sessions, currentSessionID))
}

// RevokeSession terminates a specific session.
func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		response.Error(w, http.StatusBadRequest, "missing session ID")
		return
	}

	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.authUseCase.RevokeSession(r.Context(), sessionID, user.ID, user.OrganizationID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, true, map[string]string{"message": "session successfully revoked"}, "")
}
