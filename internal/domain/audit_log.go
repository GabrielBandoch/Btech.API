package domain

import (
	"context"
	"time"
)

// Audit event taxonomy constants
const (
	EventUserLogin          = "user.login"
	EventUserRegister       = "user.register"
	EventUserLogout         = "user.logout"
	EventSessionRefresh     = "session.refresh"
	EventSessionRevoke      = "session.revoke"
	EventSessionCompromised = "session.compromised"
	EventDriverCreate       = "driver.create"
	EventDriverUpdate       = "driver.update"
	EventTripUpdate         = "trip.update"
	EventIncidentCreate     = "incident.create"
	EventPermissionDenied   = "permission.denied"
	EventSettingsUpdate     = "settings.update"
	EventSubscriptionCreated = "subscription.created"
	EventSubscriptionUpdated = "subscription.updated"
	EventSubscriptionCanceled = "subscription.canceled"
	EventEntitlementChanged = "entitlement.changed"
	EventQuotaExceeded      = "quota.exceeded"
	EventFeatureAccessDenied = "feature.access_denied"
	EventVehicleCreate      = "vehicle.create"
	EventVehicleUpdate      = "vehicle.update"
	EventVehicleDelete      = "vehicle.delete"

	EventMaintenanceSupplierCreate = "maintenance_supplier.create"
	EventMaintenanceSupplierUpdate = "maintenance_supplier.update"
	EventMaintenanceSupplierDelete = "maintenance_supplier.delete"

	EventMaintenancePlanCreate = "maintenance_plan.create"
	EventMaintenancePlanUpdate = "maintenance_plan.update"
	EventMaintenancePlanDelete = "maintenance_plan.delete"

	EventMaintenanceCreate   = "maintenance.create"
	EventMaintenanceUpdate   = "maintenance.update"
	EventMaintenanceDelete   = "maintenance.delete"
	EventMaintenanceComplete = "maintenance.complete"

	EventMaintenanceAlertCreated  = "maintenance.alert.created"
	EventMaintenanceAlertResolved = "maintenance.alert.resolved"

	// Fuel events
	EventFuelCreate  = "fuel.create"
	EventFuelUpdate  = "fuel.update"
	EventFuelDelete  = "fuel.delete"
	EventFuelAnomaly = "fuel.anomaly" // generated alongside fuel.create when anomaly detected
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
