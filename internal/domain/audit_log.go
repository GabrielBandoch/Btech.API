package domain

import (
	"context"
	"time"
)

// Audit event taxonomy constants
const (
	EventUserLogin        = "user.login"
	EventUserRegister     = "user.register"
	EventDriverCreate     = "driver.create"
	EventDriverUpdate     = "driver.update"
	EventTripUpdate       = "trip.update"
	EventIncidentCreate   = "incident.create"
	EventPermissionDenied = "permission.denied"
	EventSettingsUpdate   = "settings.update"
)

type ContextKey string

const (
	UserIDContextKey         ContextKey = "user_id"
	OrganizationIDContextKey ContextKey = "organization_id"
	ClientIPContextKey       ContextKey = "client_ip"
	UserAgentContextKey      ContextKey = "user_agent"
)

type AuditLog struct {
	ID             string                 `json:"id"`
	ActorUserID    *string                `json:"actorUserId"` // nil for unauthenticated / system actions
	OrganizationID *string                `json:"organizationId"` // nil for global system actions
	Action         string                 `json:"action"`
	EntityType     string                 `json:"entityType"`
	EntityID       *string                `json:"entityId"`
	Metadata       map[string]interface{} `json:"metadata"`
	IPAddress      string                 `json:"ipAddress"`
	UserAgent      *string                `json:"userAgent"`
	CreatedAt      time.Time              `json:"createdAt"`
}

type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	GetByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*AuditLog, error)
}
