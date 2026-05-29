package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type UsageTrackingUseCase interface {
	GetUsage(ctx context.Context, orgID string, metricKey string) (int, error)
	IncrementUsage(ctx context.Context, orgID string, metricKey string, delta int) (int, error)
	CheckAndIncrement(ctx context.Context, orgID string, metricKey string, quotaKey string, delta int) (bool, error)
}

type usageTrackingUseCase struct {
	usageRepo          domain.UsageCounterRepository
	entitlementUseCase EntitlementUseCase
	auditUseCase       AuditUseCase
}

func NewUsageTrackingUseCase(
	usageRepo domain.UsageCounterRepository,
	entitlementUseCase EntitlementUseCase,
	auditUseCase AuditUseCase,
) UsageTrackingUseCase {
	return &usageTrackingUseCase{
		usageRepo:          usageRepo,
		entitlementUseCase: entitlementUseCase,
		auditUseCase:       auditUseCase,
	}
}

func (uc *usageTrackingUseCase) GetUsage(ctx context.Context, orgID string, metricKey string) (int, error) {
	period := time.Now().Format("2006-01")
	counter, err := uc.usageRepo.Get(ctx, orgID, metricKey, period)
	if err != nil {
		return 0, err
	}
	return counter.CurrentValue, nil
}

func (uc *usageTrackingUseCase) IncrementUsage(ctx context.Context, orgID string, metricKey string, delta int) (int, error) {
	period := time.Now().Format("2006-01")
	counter, err := uc.usageRepo.Increment(ctx, orgID, metricKey, period, delta)
	if err != nil {
		return 0, err
	}
	return counter.CurrentValue, nil
}

func (uc *usageTrackingUseCase) CheckAndIncrement(ctx context.Context, orgID string, metricKey string, quotaKey string, delta int) (bool, error) {
	quotaVal, err := uc.entitlementUseCase.GetEntitlementValue(ctx, orgID, quotaKey)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate quota limit: %w", err)
	}

	// Unlimited check
	if quotaVal == "unlimited" {
		_, err = uc.IncrementUsage(ctx, orgID, metricKey, delta)
		return err == nil, err
	}

	limit, err := strconv.Atoi(quotaVal)
	if err != nil {
		limit = 0
	}

	current, err := uc.GetUsage(ctx, orgID, metricKey)
	if err != nil {
		return false, fmt.Errorf("failed to get current usage: %w", err)
	}

	if current+delta > limit {
		// Log quota exceeded
		uc.auditUseCase.Log(ctx, domain.EventQuotaExceeded, "quota", &quotaKey, map[string]interface{}{
			"organization_id": orgID,
			"quota_key":       quotaKey,
			"current_usage":   current,
			"requested_delta": delta,
			"limit":           limit,
		})
		return false, nil
	}

	_, err = uc.IncrementUsage(ctx, orgID, metricKey, delta)
	return err == nil, err
}
