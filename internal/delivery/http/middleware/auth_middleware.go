package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

type contextKey string

const (
	UserContextKey           contextKey = "user"
	UserIDContextKey         contextKey = "user_id"
	OrganizationIDContextKey contextKey = "organization_id"
	RoleContextKey           contextKey = "role"
)

// AuthMiddleware creates a JWT authentication middleware that validates access tokens.
func AuthMiddleware(authUseCase usecase.AuthUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "authorization token is required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(w, http.StatusUnauthorized, "authorization header must be in format Bearer <token>")
				return
			}

			tokenStr := parts[1]
			user, claims, err := authUseCase.ValidateToken(r.Context(), tokenStr)
			if err != nil {
				// Normalize validation error for client safety
				response.Error(w, http.StatusUnauthorized, "invalid or expired authorization token")
				return
			}

			// Inject values into request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserContextKey, user)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, OrganizationIDContextKey, claims.OrganizationID)
			ctx = context.WithValue(ctx, RoleContextKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext extracts the authenticated user from the context.
func UserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*domain.User)
	return user, ok
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}

// OrganizationIDFromContext extracts the organization ID from the context.
func OrganizationIDFromContext(ctx context.Context) (string, bool) {
	orgID, ok := ctx.Value(OrganizationIDContextKey).(string)
	return orgID, ok
}

// RoleFromContext extracts the role from the context.
func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(RoleContextKey).(string)
	return role, ok
}

// RequireRole checks if the authenticated request user's role is one of the allowed roles.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "unauthorized: missing security role context")
				return
			}

			roleAllowed := false
			for _, allowed := range allowedRoles {
				if strings.EqualFold(role, allowed) {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				response.Error(w, http.StatusForbidden, "forbidden: insufficient role access permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole is an alias for RequireRole for semantic clarity.
func RequireAnyRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return RequireRole(allowedRoles...)
}
