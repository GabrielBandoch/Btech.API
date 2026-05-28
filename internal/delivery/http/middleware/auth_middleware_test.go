package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/domain"
)

func TestRequirePermission(t *testing.T) {
	// Setup dummy handler
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, "success")
	})

	t.Run("MissingUserContext", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		RequirePermission("drivers:create")(dummyHandler).ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("UserLacksPermission", func(t *testing.T) {
		user := &domain.User{
			ID:          "user-1",
			Role:        "operator",
			Permissions: []string{"drivers:read", "trips:read"},
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), UserContextKey, user)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		RequirePermission("drivers:create")(dummyHandler).ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("UserHasPermission", func(t *testing.T) {
		user := &domain.User{
			ID:          "user-1",
			Role:        "operator",
			Permissions: []string{"drivers:read", "drivers:create"},
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), UserContextKey, user)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		RequirePermission("drivers:create")(dummyHandler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
	})
}

func TestRequireAnyPermission(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, "success")
	})

	t.Run("UserLacksAllPermissions", func(t *testing.T) {
		user := &domain.User{
			ID:          "user-1",
			Role:        "viewer",
			Permissions: []string{"drivers:read"},
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), UserContextKey, user)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		RequireAnyPermission("drivers:create", "trips:update")(dummyHandler).ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("UserHasOneOfRequiredPermissions", func(t *testing.T) {
		user := &domain.User{
			ID:          "user-1",
			Role:        "operator",
			Permissions: []string{"drivers:read", "trips:update"},
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), UserContextKey, user)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		RequireAnyPermission("drivers:create", "trips:update")(dummyHandler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
	})
}
