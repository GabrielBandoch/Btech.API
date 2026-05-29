package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/btech/fleetcontrol-api/internal/delivery/http/response"
	"github.com/btech/fleetcontrol-api/internal/domain"
)

type mockEntitlementUseCase struct {
	allowedFeatures map[string]bool
}

func (m *mockEntitlementUseCase) EvaluateFeature(ctx context.Context, orgID string, featureKey string) (bool, error) {
	k := orgID + ":" + featureKey
	return m.allowedFeatures[k], nil
}

func (m *mockEntitlementUseCase) EvaluateQuota(ctx context.Context, orgID string, quotaKey string, currentUsage int) (bool, error) {
	return true, nil
}

func (m *mockEntitlementUseCase) GetEntitlementValue(ctx context.Context, orgID string, key string) (string, error) {
	return "", nil
}

func TestRequireEntitlement(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, "success")
	})

	t.Run("MissingOrgContext", func(t *testing.T) {
		mockUC := &mockEntitlementUseCase{}
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		RequireEntitlement(mockUC, domain.EntitlementFeatureAuditLogs)(dummyHandler).ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("OrgLacksEntitlement", func(t *testing.T) {
		mockUC := &mockEntitlementUseCase{
			allowedFeatures: make(map[string]bool),
		}
		// Deny feature.audit_logs for 'org-1'
		mockUC.allowedFeatures["org-1:feature.audit_logs"] = false

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), OrganizationIDContextKey, "org-1")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		RequireEntitlement(mockUC, domain.EntitlementFeatureAuditLogs)(dummyHandler).ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", w.Code)
		}
	})

	t.Run("OrgHasEntitlement", func(t *testing.T) {
		mockUC := &mockEntitlementUseCase{
			allowedFeatures: make(map[string]bool),
		}
		// Allow feature.audit_logs for 'org-1'
		mockUC.allowedFeatures["org-1:feature.audit_logs"] = true

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := context.WithValue(req.Context(), OrganizationIDContextKey, "org-1")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		RequireEntitlement(mockUC, domain.EntitlementFeatureAuditLogs)(dummyHandler).ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
	})
}
