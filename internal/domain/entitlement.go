package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrQuotaExceeded = errors.New("organization quota exceeded")
)

const (
	EntitlementDriversMax             = "drivers.max"
	EntitlementTripsMonthly           = "trips.monthly"
	EntitlementUsersMax               = "users.max"
	EntitlementFeatureAuditLogs       = "feature.audit_logs"
	EntitlementFeatureAPIAccess       = "feature.api_access"
	EntitlementFeatureAdvancedReports = "feature.advanced_reports"
)

type OrganizationEntitlementOverride struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	Key            string     `json:"key"`
	Value          string     `json:"value"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type EntitlementRepository interface {
	GetActiveOverride(ctx context.Context, orgID string, key string) (*OrganizationEntitlementOverride, error)
	ListActiveOverrides(ctx context.Context, orgID string) ([]*OrganizationEntitlementOverride, error)
	SetOverride(ctx context.Context, override *OrganizationEntitlementOverride) error
	DeleteOverride(ctx context.Context, orgID string, key string) error
}
