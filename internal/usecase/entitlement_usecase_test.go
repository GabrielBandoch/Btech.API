package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type mockEntitlementRepository struct {
	overrides map[string]*domain.OrganizationEntitlementOverride
}

func newMockEntitlementRepository() *mockEntitlementRepository {
	return &mockEntitlementRepository{
		overrides: make(map[string]*domain.OrganizationEntitlementOverride),
	}
}

func (m *mockEntitlementRepository) GetActiveOverride(ctx context.Context, orgID string, key string) (*domain.OrganizationEntitlementOverride, error) {
	k := orgID + ":" + key
	override, ok := m.overrides[k]
	if !ok {
		return nil, nil
	}
	if override.ExpiresAt != nil && override.ExpiresAt.Before(time.Now()) {
		return nil, nil // expired
	}
	return override, nil
}

func (m *mockEntitlementRepository) ListActiveOverrides(ctx context.Context, orgID string) ([]*domain.OrganizationEntitlementOverride, error) {
	var list []*domain.OrganizationEntitlementOverride
	for _, o := range m.overrides {
		if o.OrganizationID == orgID {
			if o.ExpiresAt == nil || o.ExpiresAt.After(time.Now()) {
				list = append(list, o)
			}
		}
	}
	return list, nil
}

func (m *mockEntitlementRepository) SetOverride(ctx context.Context, override *domain.OrganizationEntitlementOverride) error {
	k := override.OrganizationID + ":" + override.Key
	m.overrides[k] = override
	return nil
}

func (m *mockEntitlementRepository) DeleteOverride(ctx context.Context, orgID string, key string) error {
	k := orgID + ":" + key
	delete(m.overrides, k)
	return nil
}

func TestEntitlementUseCase_EvaluateFeature(t *testing.T) {
	planRepo := newMockPlanRepository()
	subRepo := newMockSubscriptionRepository()
	entRepo := newMockEntitlementRepository()
	audit := &mockAuditUseCase{}

	uc := NewEntitlementUseCase(subRepo, planRepo, entRepo, audit)

	ctx := context.Background()
	orgID := "org-feat-test"

	// Setup plan entitlements for 'free' and 'pro'
	planRepo.entitlements["plan-free-id"] = []*domain.PlanEntitlement{
		{PlanID: "plan-free-id", Key: "feature.audit_logs", Value: "false"},
	}
	planRepo.entitlements["plan-pro-id"] = []*domain.PlanEntitlement{
		{PlanID: "plan-pro-id", Key: "feature.audit_logs", Value: "true"},
	}

	// 1. Unsubscribed / Free plan (feature disabled)
	allowed, err := uc.EvaluateFeature(ctx, orgID, "feature.audit_logs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected feature to be disabled on free plan")
	}

	// 2. Active Subscription (pro plan - feature enabled)
	subRepo.subs[orgID] = &domain.Subscription{
		ID:             "sub-1",
		OrganizationID: orgID,
		PlanID:         "plan-pro-id",
		Status:         domain.SubscriptionStatusActive,
	}

	allowed, err = uc.EvaluateFeature(ctx, orgID, "feature.audit_logs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected feature to be enabled on pro plan")
	}

	// 3. Override takes precedence (override to false for pro tenant)
	err = entRepo.SetOverride(ctx, &domain.OrganizationEntitlementOverride{
		ID:             "over-1",
		OrganizationID: orgID,
		Key:            "feature.audit_logs",
		Value:          "false",
		CreatedAt:      time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	allowed, err = uc.EvaluateFeature(ctx, orgID, "feature.audit_logs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected override value 'false' to override plan entitlement 'true'")
	}
}

func TestEntitlementUseCase_EvaluateQuota(t *testing.T) {
	planRepo := newMockPlanRepository()
	subRepo := newMockSubscriptionRepository()
	entRepo := newMockEntitlementRepository()
	audit := &mockAuditUseCase{}

	uc := NewEntitlementUseCase(subRepo, planRepo, entRepo, audit)

	ctx := context.Background()
	orgID := "org-quota-test"

	planRepo.entitlements["plan-free-id"] = []*domain.PlanEntitlement{
		{PlanID: "plan-free-id", Key: "drivers.max", Value: "5"},
	}
	planRepo.entitlements["plan-pro-id"] = []*domain.PlanEntitlement{
		{PlanID: "plan-pro-id", Key: "drivers.max", Value: "unlimited"},
	}

	// 1. Free plan limit = 5
	// Usage = 4 -> Allowed
	allowed, err := uc.EvaluateQuota(ctx, orgID, "drivers.max", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected usage 4 to be within limit 5")
	}

	// Usage = 5 -> Blocked
	allowed, _ = uc.EvaluateQuota(ctx, orgID, "drivers.max", 5)
	if allowed {
		t.Error("expected usage 5 to be blocked by limit 5")
	}

	// 2. Pro plan limit = unlimited
	subRepo.subs[orgID] = &domain.Subscription{
		ID:             "sub-2",
		OrganizationID: orgID,
		PlanID:         "plan-pro-id",
		Status:         domain.SubscriptionStatusActive,
	}

	// Usage = 1000 -> Allowed because it's unlimited
	allowed, err = uc.EvaluateQuota(ctx, orgID, "drivers.max", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected unlimited limit to bypass quota checks")
	}
}
