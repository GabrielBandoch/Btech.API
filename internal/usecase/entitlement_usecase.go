package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type EntitlementUseCase interface {
	EvaluateFeature(ctx context.Context, orgID string, featureKey string) (bool, error)
	EvaluateQuota(ctx context.Context, orgID string, quotaKey string, currentUsage int) (bool, error)
	GetEntitlementValue(ctx context.Context, orgID string, key string) (string, error)
}

type entitlementUseCase struct {
	subscriptionRepo domain.SubscriptionRepository
	planRepo         domain.PlanRepository
	entitlementRepo  domain.EntitlementRepository
	auditUseCase     AuditUseCase
}

func NewEntitlementUseCase(
	subscriptionRepo domain.SubscriptionRepository,
	planRepo domain.PlanRepository,
	entitlementRepo domain.EntitlementRepository,
	auditUseCase AuditUseCase,
) EntitlementUseCase {
	return &entitlementUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		entitlementRepo:  entitlementRepo,
		auditUseCase:     auditUseCase,
	}
}

func (uc *entitlementUseCase) GetEntitlementValue(ctx context.Context, orgID string, key string) (string, error) {
	// 1. Check override
	override, err := uc.entitlementRepo.GetActiveOverride(ctx, orgID, key)
	if err == nil && override != nil {
		return override.Value, nil
	}

	// 2. Fetch active subscription
	sub, err := uc.subscriptionRepo.GetByOrganizationID(ctx, orgID)
	var planID string
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			// Fallback to free plan
			freePlan, planErr := uc.planRepo.GetByCode(ctx, "free")
			if planErr != nil {
				return "", fmt.Errorf("failed to retrieve fallback free plan: %w", planErr)
			}
			planID = freePlan.ID
		} else {
			return "", fmt.Errorf("failed to retrieve subscription: %w", err)
		}
	} else {
		// Verify if subscription status is expired
		if sub.Status == domain.SubscriptionStatusExpired {
			// If subscription is expired, all entitlements default to restricted/blocked
			return "false", nil
		}
		planID = sub.PlanID
	}

	// 3. Fetch plan entitlements
	ents, err := uc.planRepo.GetEntitlements(ctx, planID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve plan entitlements: %w", err)
	}

	for _, ent := range ents {
		if ent.Key == key {
			return ent.Value, nil
		}
	}

	return "", nil
}

func (uc *entitlementUseCase) EvaluateFeature(ctx context.Context, orgID string, featureKey string) (bool, error) {
	val, err := uc.GetEntitlementValue(ctx, orgID, featureKey)
	if err != nil {
		return false, err
	}

	allowed := val == "true"

	if !allowed {
		// Log feature access denied
		uc.auditUseCase.Log(ctx, domain.EventFeatureAccessDenied, "feature", &featureKey, map[string]interface{}{
			"organization_id": orgID,
			"feature_key":     featureKey,
		})
	}

	return allowed, nil
}

func (uc *entitlementUseCase) EvaluateQuota(ctx context.Context, orgID string, quotaKey string, currentUsage int) (bool, error) {
	val, err := uc.GetEntitlementValue(ctx, orgID, quotaKey)
	if err != nil {
		return false, err
	}

	// Unlimited bypass
	if val == "unlimited" {
		return true, nil
	}

	limit, err := strconv.Atoi(val)
	if err != nil {
		// If it's empty or invalid, fallback to 0 limit (blocked)
		limit = 0
	}

	allowed := currentUsage < limit

	if !allowed {
		// Log quota exceeded
		uc.auditUseCase.Log(ctx, domain.EventQuotaExceeded, "quota", &quotaKey, map[string]interface{}{
			"organization_id": orgID,
			"quota_key":       quotaKey,
			"current_usage":   currentUsage,
			"limit":           limit,
		})
	}

	return allowed, nil
}
