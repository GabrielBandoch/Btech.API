package domain

import (
	"context"
	"time"
)

type UsageCounter struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	MetricKey      string    `json:"metricKey"`
	CurrentValue   int       `json:"currentValue"`
	BillingPeriod  string    `json:"billingPeriod"` // format YYYY-MM
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type UsageCounterRepository interface {
	Get(ctx context.Context, orgID string, metricKey string, billingPeriod string) (*UsageCounter, error)
	Increment(ctx context.Context, orgID string, metricKey string, billingPeriod string, delta int) (*UsageCounter, error)
	Reset(ctx context.Context, orgID string, metricKey string, billingPeriod string) error
}
