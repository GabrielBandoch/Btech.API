package usecase

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type mockPlanRepository struct {
	plans        map[string]*domain.Plan
	entitlements map[string][]*domain.PlanEntitlement
}

func newMockPlanRepository() *mockPlanRepository {
	m := &mockPlanRepository{
		plans:        make(map[string]*domain.Plan),
		entitlements: make(map[string][]*domain.PlanEntitlement),
	}
	// Seed standard plans for testing
	m.plans["plan-free-id"] = &domain.Plan{
		ID:           "plan-free-id",
		Code:         "free",
		Name:         "Free Plan",
		MonthlyPrice: 0,
		YearlyPrice:  0,
		IsActive:     true,
	}
	m.plans["plan-pro-id"] = &domain.Plan{
		ID:           "plan-pro-id",
		Code:         "pro",
		Name:         "Pro Plan",
		MonthlyPrice: 99.00,
		YearlyPrice:  990.00,
		IsActive:     true,
	}
	return m
}

func (m *mockPlanRepository) GetByID(ctx context.Context, id string) (*domain.Plan, error) {
	p, ok := m.plans[id]
	if !ok {
		return nil, domain.ErrPlanNotFound
	}
	return p, nil
}

func (m *mockPlanRepository) GetByCode(ctx context.Context, code string) (*domain.Plan, error) {
	for _, p := range m.plans {
		if p.Code == code {
			return p, nil
		}
	}
	return nil, domain.ErrPlanNotFound
}

func (m *mockPlanRepository) ListActive(ctx context.Context) ([]*domain.Plan, error) {
	var list []*domain.Plan
	for _, p := range m.plans {
		if p.IsActive {
			list = append(list, p)
		}
	}
	return list, nil
}

func (m *mockPlanRepository) GetEntitlements(ctx context.Context, planID string) ([]*domain.PlanEntitlement, error) {
	return m.entitlements[planID], nil
}

func (m *mockPlanRepository) Create(ctx context.Context, plan *domain.Plan) error {
	m.plans[plan.ID] = plan
	return nil
}

func (m *mockPlanRepository) CreateEntitlement(ctx context.Context, ent *domain.PlanEntitlement) error {
	m.entitlements[ent.PlanID] = append(m.entitlements[ent.PlanID], ent)
	return nil
}

type mockSubscriptionRepository struct {
	subs    map[string]*domain.Subscription
	history map[string][]*domain.SubscriptionEventHistory
}

func newMockSubscriptionRepository() *mockSubscriptionRepository {
	return &mockSubscriptionRepository{
		subs:    make(map[string]*domain.Subscription),
		history: make(map[string][]*domain.SubscriptionEventHistory),
	}
}

func (m *mockSubscriptionRepository) GetByOrganizationID(ctx context.Context, orgID string) (*domain.Subscription, error) {
	for _, s := range m.subs {
		if s.OrganizationID == orgID {
			return s, nil
		}
	}
	return nil, domain.ErrSubscriptionNotFound
}

func (m *mockSubscriptionRepository) GetByID(ctx context.Context, id string) (*domain.Subscription, error) {
	s, ok := m.subs[id]
	if !ok {
		return nil, domain.ErrSubscriptionNotFound
	}
	return s, nil
}

func (m *mockSubscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) error {
	m.subs[sub.ID] = sub
	return nil
}

func (m *mockSubscriptionRepository) Update(ctx context.Context, sub *domain.Subscription) error {
	m.subs[sub.ID] = sub
	return nil
}

func (m *mockSubscriptionRepository) CreateEventHistory(ctx context.Context, event *domain.SubscriptionEventHistory) error {
	m.history[event.OrganizationID] = append(m.history[event.OrganizationID], event)
	return nil
}

func (m *mockSubscriptionRepository) GetEventHistory(ctx context.Context, orgID string) ([]*domain.SubscriptionEventHistory, error) {
	return m.history[orgID], nil
}

func TestBillingUseCase_CreateSubscription(t *testing.T) {
	planRepo := newMockPlanRepository()
	subRepo := newMockSubscriptionRepository()
	audit := &mockAuditUseCase{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := NewBillingUseCase(subRepo, planRepo, audit, logger)

	ctx := context.Background()
	orgID := "test-org-123"

	// Create a new pro subscription (non-trial)
	sub, err := uc.CreateSubscription(ctx, orgID, "pro", 0)
	if err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}

	if sub.OrganizationID != orgID {
		t.Errorf("expected org ID %s, got %s", orgID, sub.OrganizationID)
	}

	if sub.Status != domain.SubscriptionStatusActive {
		t.Errorf("expected status %s, got %s", domain.SubscriptionStatusActive, sub.Status)
	}

	if sub.PlanID != "plan-pro-id" {
		t.Errorf("expected plan ID plan-pro-id, got %s", sub.PlanID)
	}

	// Verify history event was created
	histories, _ := subRepo.GetEventHistory(ctx, orgID)
	if len(histories) != 1 {
		t.Fatalf("expected 1 history record, got %d", len(histories))
	}
	if histories[0].EventType != domain.EventSubscriptionCreated {
		t.Errorf("expected event type %s, got %s", domain.EventSubscriptionCreated, histories[0].EventType)
	}
}

func TestBillingUseCase_ResolveActiveSubscription_Fallback(t *testing.T) {
	planRepo := newMockPlanRepository()
	subRepo := newMockSubscriptionRepository()
	audit := &mockAuditUseCase{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := NewBillingUseCase(subRepo, planRepo, audit, logger)

	ctx := context.Background()
	orgID := "test-org-456"

	// Resolve subscription for organization without any active subscription
	sub, plan, err := uc.ResolveActiveSubscription(ctx, orgID)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if sub == nil || plan == nil {
		t.Fatal("expected auto-provisioned subscription and plan, got nil")
	}

	if plan.Code != "free" {
		t.Errorf("expected auto-provisioned plan to be 'free', got '%s'", plan.Code)
	}

	if sub.Status != domain.SubscriptionStatusActive {
		t.Errorf("expected active fallback status, got %s", sub.Status)
	}
}

func TestBillingUseCase_UpgradeAndCancel(t *testing.T) {
	planRepo := newMockPlanRepository()
	subRepo := newMockSubscriptionRepository()
	audit := &mockAuditUseCase{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	uc := NewBillingUseCase(subRepo, planRepo, audit, logger)

	ctx := context.Background()
	orgID := "test-org-789"

	// 1. Initial creation (free)
	_, _ = uc.CreateSubscription(ctx, orgID, "free", 0)

	// 2. Upgrade to pro
	sub, err := uc.UpdateSubscription(ctx, orgID, "pro")
	if err != nil {
		t.Fatalf("failed to upgrade subscription: %v", err)
	}

	if sub.PlanID != "plan-pro-id" {
		t.Errorf("expected upgraded plan ID to be plan-pro-id, got %s", sub.PlanID)
	}

	// 3. Cancel
	sub, err = uc.CancelSubscription(ctx, orgID)
	if err != nil {
		t.Fatalf("failed to cancel subscription: %v", err)
	}

	if sub.Status != domain.SubscriptionStatusCanceled {
		t.Errorf("expected status %s, got %s", domain.SubscriptionStatusCanceled, sub.Status)
	}

	if sub.CanceledAt == nil {
		t.Error("expected CanceledAt timestamp to be set")
	}
}
