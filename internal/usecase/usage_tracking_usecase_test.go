package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type mockUsageCounterRepository struct {
	counters map[string]*domain.UsageCounter
}

func newMockUsageCounterRepository() *mockUsageCounterRepository {
	return &mockUsageCounterRepository{
		counters: make(map[string]*domain.UsageCounter),
	}
}

func (m *mockUsageCounterRepository) Get(ctx context.Context, orgID string, metricKey string, billingPeriod string) (*domain.UsageCounter, error) {
	k := orgID + ":" + metricKey + ":" + billingPeriod
	c, ok := m.counters[k]
	if !ok {
		return &domain.UsageCounter{
			OrganizationID: orgID,
			MetricKey:      metricKey,
			CurrentValue:   0,
			BillingPeriod:  billingPeriod,
		}, nil
	}
	return c, nil
}

func (m *mockUsageCounterRepository) Increment(ctx context.Context, orgID string, metricKey string, billingPeriod string, delta int) (*domain.UsageCounter, error) {
	k := orgID + ":" + metricKey + ":" + billingPeriod
	c, ok := m.counters[k]
	if !ok {
		c = &domain.UsageCounter{
			ID:             "uc-" + k,
			OrganizationID: orgID,
			MetricKey:      metricKey,
			CurrentValue:   0,
			BillingPeriod:  billingPeriod,
		}
		m.counters[k] = c
	}
	c.CurrentValue += delta
	return c, nil
}

func (m *mockUsageCounterRepository) Reset(ctx context.Context, orgID string, metricKey string, billingPeriod string) error {
	k := orgID + ":" + metricKey + ":" + billingPeriod
	if c, ok := m.counters[k]; ok {
		c.CurrentValue = 0
	}
	return nil
}

func TestUsageTrackingUseCase_CheckAndIncrement(t *testing.T) {
	planRepo := newMockPlanRepository()
	subRepo := newMockSubscriptionRepository()
	entRepo := newMockEntitlementRepository()
	audit := &mockAuditUseCase{}
	usageRepo := newMockUsageCounterRepository()

	entUC := NewEntitlementUseCase(subRepo, planRepo, entRepo, audit)
	uc := NewUsageTrackingUseCase(usageRepo, entUC, audit)

	ctx := context.Background()
	orgID := "org-usage-test"
	period := time.Now().Format("2006-01")

	// Free plan: trips.monthly limit = 50
	planRepo.entitlements["plan-free-id"] = []*domain.PlanEntitlement{
		{PlanID: "plan-free-id", Key: "trips.monthly", Value: "50"},
	}

	// 1. First increment (delta 10) -> Allowed
	allowed, err := uc.CheckAndIncrement(ctx, orgID, "trips.created", "trips.monthly", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected first increment to be allowed")
	}

	// Verify current usage is 10
	val, _ := uc.GetUsage(ctx, orgID, "trips.created")
	if val != 10 {
		t.Errorf("expected usage to be 10, got %d", val)
	}

	// 2. Large increment (delta 41) -> Blocked (10 + 41 = 51 > 50)
	allowed, err = uc.CheckAndIncrement(ctx, orgID, "trips.created", "trips.monthly", 41)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected increment over limit to be blocked")
	}

	// Verify usage remains 10 (not updated on block)
	val, _ = uc.GetUsage(ctx, orgID, "trips.created")
	if val != 10 {
		t.Errorf("expected usage to remain 10, got %d", val)
	}

	// 3. Increment exactly to limit (delta 40 -> 10 + 40 = 50) -> Allowed
	allowed, err = uc.CheckAndIncrement(ctx, orgID, "trips.created", "trips.monthly", 40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected increment exactly up to limit to be allowed")
	}

	val, _ = uc.GetUsage(ctx, orgID, "trips.created")
	if val != 50 {
		t.Errorf("expected usage to be 50, got %d", val)
	}

	// 4. Reset counter -> Usage = 0
	err = usageRepo.Reset(ctx, orgID, "trips.created", period)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val, _ = uc.GetUsage(ctx, orgID, "trips.created")
	if val != 0 {
		t.Errorf("expected usage to be reset to 0, got %d", val)
	}
}
