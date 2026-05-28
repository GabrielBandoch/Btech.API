package middleware

import (
	"context"
	"net/http"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

// AuditContextMiddleware extracts Client IP and User-Agent from HTTP headers and injects them into request context.
func AuditContextMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1. Extract Client IP Address
			ip := getClientIP(r)
			ctx = context.WithValue(ctx, domain.ClientIPContextKey, ip)

			// 2. Extract User-Agent
			ua := r.Header.Get("User-Agent")
			ctx = context.WithValue(ctx, domain.UserAgentContextKey, ua)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
