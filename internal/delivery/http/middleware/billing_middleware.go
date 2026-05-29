package middleware

import (
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

// RequireEntitlement checks if the active organization has the specified billing feature entitlement.
func RequireEntitlement(entitlementUseCase usecase.EntitlementUseCase, entitlementKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := OrganizationIDFromContext(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "unauthorized: missing organization context")
				return
			}

			allowed, err := entitlementUseCase.EvaluateFeature(r.Context(), orgID, entitlementKey)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, "internal server error evaluating billing entitlements")
				return
			}

			if !allowed {
				response.Error(w, http.StatusForbidden, "forbidden: resource not available on current plan")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
