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

type spyAuditUseCase struct {
	loggedActions []string
}

func (s *spyAuditUseCase) Log(ctx context.Context, action string, entityType string, entityID *string, metadata map[string]interface{}) {
	s.loggedActions = append(s.loggedActions, action)
}

func (s *spyAuditUseCase) GetLogsByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*domain.AuditLog, error) {
	return nil, nil
}

func TestRequirePermission_AuditLog(t *testing.T) {
	spy := &spyAuditUseCase{}
	SetAuditUseCase(spy)
	defer SetAuditUseCase(nil) // Reset after test

	user := &domain.User{
		ID:          "user-1",
		Role:        "viewer",
		Permissions: []string{"drivers:read"},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	RequirePermission("drivers:create")(dummyHandler).ServeHTTP(w, req)

	if len(spy.loggedActions) != 1 {
		t.Fatalf("expected 1 logged audit action, got %d", len(spy.loggedActions))
	}

	if spy.loggedActions[0] != domain.EventPermissionDenied {
		t.Errorf("expected logged action %s, got %s", domain.EventPermissionDenied, spy.loggedActions[0])
	}
}

