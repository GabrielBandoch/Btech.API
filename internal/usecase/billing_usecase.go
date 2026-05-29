package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type BillingUseCase interface {
	ResolveActiveSubscription(ctx context.Context, orgID string) (*domain.Subscription, *domain.Plan, error)
	CreateSubscription(ctx context.Context, orgID, planCode string, trialDays int) (*domain.Subscription, error)
	UpdateSubscription(ctx context.Context, orgID, planCode string) (*domain.Subscription, error)
	CancelSubscription(ctx context.Context, orgID string) (*domain.Subscription, error)
}

type billingUseCase struct {
	subscriptionRepo domain.SubscriptionRepository
	planRepo         domain.PlanRepository
	auditUseCase     AuditUseCase
	logger           *slog.Logger
}

func NewBillingUseCase(
	subscriptionRepo domain.SubscriptionRepository,
	planRepo domain.PlanRepository,
	auditUseCase AuditUseCase,
	logger *slog.Logger,
) BillingUseCase {
	return &billingUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		auditUseCase:     auditUseCase,
		logger:           logger,
	}
}

func (uc *billingUseCase) ResolveActiveSubscription(ctx context.Context, orgID string) (*domain.Subscription, *domain.Plan, error) {
	sub, err := uc.subscriptionRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			// Auto-provision Free subscription if not found to prevent system disruption
			uc.logger.Info("No subscription found for organization, auto-provisioning Free plan", slog.String("org_id", orgID))
			newSub, createErr := uc.CreateSubscription(ctx, orgID, "free", 0)
			if createErr != nil {
				return nil, nil, fmt.Errorf("failed to auto-provision free plan: %w", createErr)
			}
			sub = newSub
		} else {
			return nil, nil, fmt.Errorf("failed to get subscription: %w", err)
		}
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get plan for subscription: %w", err)
	}

	return sub, plan, nil
}

func (uc *billingUseCase) CreateSubscription(ctx context.Context, orgID, planCode string, trialDays int) (*domain.Subscription, error) {
	plan, err := uc.planRepo.GetByCode(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve plan: %w", err)
	}

	now := time.Now()
	status := domain.SubscriptionStatusActive
	var trialEndsAt *time.Time
	if trialDays > 0 {
		status = domain.SubscriptionStatusTrialing
		ends := now.AddDate(0, 0, trialDays)
		trialEndsAt = &ends
	}

	sub := &domain.Subscription{
		ID:             newBillingUUID(),
		OrganizationID: orgID,
		PlanID:         plan.ID,
		Status:         status,
		StartsAt:       now,
		EndsAt:         now.AddDate(1, 0, 0), // Default 1 year validity
		TrialEndsAt:    trialEndsAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.subscriptionRepo.Create(ctx, sub); err != nil {
		return nil, err
	}

	// Insert into history
	history := &domain.SubscriptionEventHistory{
		ID:             newBillingUUID(),
		OrganizationID: orgID,
		SubscriptionID: sub.ID,
		EventType:      domain.EventSubscriptionCreated,
		ToStatus:       status,
		Metadata: map[string]interface{}{
			"plan_code":  planCode,
			"trial_days": trialDays,
		},
		CreatedAt: now,
	}
	_ = uc.subscriptionRepo.CreateEventHistory(ctx, history)

	// Audit Log
	uc.auditUseCase.Log(ctx, domain.EventSubscriptionCreated, "subscription", &sub.ID, map[string]interface{}{
		"organization_id": orgID,
		"plan_code":       planCode,
		"status":          status,
	})

	return sub, nil
}

func (uc *billingUseCase) UpdateSubscription(ctx context.Context, orgID, planCode string) (*domain.Subscription, error) {
	sub, err := uc.subscriptionRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve current subscription: %w", err)
	}

	newPlan, err := uc.planRepo.GetByCode(ctx, planCode)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve new plan: %w", err)
	}

	oldStatus := sub.Status
	oldPlanID := sub.PlanID

	now := time.Now()
	sub.PlanID = newPlan.ID
	sub.Status = domain.SubscriptionStatusActive
	sub.StartsAt = now
	sub.EndsAt = now.AddDate(1, 0, 0)
	sub.UpdatedAt = now

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		return nil, err
	}

	// Record history
	history := &domain.SubscriptionEventHistory{
		ID:             newBillingUUID(),
		OrganizationID: orgID,
		SubscriptionID: sub.ID,
		EventType:      domain.EventSubscriptionUpdated,
		FromStatus:     &oldStatus,
		ToStatus:       domain.SubscriptionStatusActive,
		Metadata: map[string]interface{}{
			"old_plan_id": oldPlanID,
			"new_plan_id": newPlan.ID,
			"plan_code":   planCode,
		},
		CreatedAt: now,
	}
	_ = uc.subscriptionRepo.CreateEventHistory(ctx, history)

	// Audit Log
	uc.auditUseCase.Log(ctx, domain.EventSubscriptionUpdated, "subscription", &sub.ID, map[string]interface{}{
		"organization_id": orgID,
		"old_plan_id":     oldPlanID,
		"new_plan_id":     newPlan.ID,
		"plan_code":       planCode,
	})

	return sub, nil
}

func (uc *billingUseCase) CancelSubscription(ctx context.Context, orgID string) (*domain.Subscription, error) {
	sub, err := uc.subscriptionRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subscription: %w", err)
	}

	oldStatus := sub.Status
	now := time.Now()
	sub.Status = domain.SubscriptionStatusCanceled
	sub.CanceledAt = &now
	sub.UpdatedAt = now

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		return nil, err
	}

	// Record history
	history := &domain.SubscriptionEventHistory{
		ID:             newBillingUUID(),
		OrganizationID: orgID,
		SubscriptionID: sub.ID,
		EventType:      domain.EventSubscriptionCanceled,
		FromStatus:     &oldStatus,
		ToStatus:       domain.SubscriptionStatusCanceled,
		Metadata: map[string]interface{}{
			"reason": "user_request",
		},
		CreatedAt: now,
	}
	_ = uc.subscriptionRepo.CreateEventHistory(ctx, history)

	// Audit Log
	uc.auditUseCase.Log(ctx, domain.EventSubscriptionCanceled, "subscription", &sub.ID, map[string]interface{}{
		"organization_id": orgID,
		"status":          sub.Status,
	})

	return sub, nil
}

func newBillingUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
