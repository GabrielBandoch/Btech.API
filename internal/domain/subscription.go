package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

const (
	SubscriptionStatusTrialing = "trialing"
	SubscriptionStatusActive   = "active"
	SubscriptionStatusPastDue  = "past_due"
	SubscriptionStatusCanceled = "canceled"
	SubscriptionStatusExpired  = "expired"
)

type Subscription struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	PlanID         string     `json:"planId"`
	Status         string     `json:"status"`
	StartsAt       time.Time  `json:"startsAt"`
	EndsAt         time.Time  `json:"endsAt"`
	TrialEndsAt    *time.Time `json:"trialEndsAt"`
	CanceledAt     *time.Time `json:"canceledAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type SubscriptionEventHistory struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organizationId"`
	SubscriptionID string                 `json:"subscriptionId"`
	EventType      string                 `json:"eventType"` // "subscription.created", "subscription.updated", etc.
	FromStatus     *string                `json:"fromStatus"`
	ToStatus       string                 `json:"toStatus"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"createdAt"`
}

type SubscriptionRepository interface {
	GetByOrganizationID(ctx context.Context, orgID string) (*Subscription, error)
	GetByID(ctx context.Context, id string) (*Subscription, error)
	Create(ctx context.Context, sub *Subscription) error
	Update(ctx context.Context, sub *Subscription) error
	CreateEventHistory(ctx context.Context, event *SubscriptionEventHistory) error
	GetEventHistory(ctx context.Context, orgID string) ([]*SubscriptionEventHistory, error)
}
