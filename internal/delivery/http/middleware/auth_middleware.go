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

const userContextKey contextKey = "user"

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
				response.Error(w, http.StatusUnauthorized, "authorization header must be in Format Bearer <token>")
				return
			}

			tokenStr := parts[1]
			user, err := authUseCase.ValidateToken(r.Context(), tokenStr)
			if err != nil {
				// Normalize validation error for client safety
				response.Error(w, http.StatusUnauthorized, "invalid or expired authorization token")
				return
			}

			// Inject user into context
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext extracts the authenticated user from the context.
func UserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(*domain.User)
	return user, ok
}
