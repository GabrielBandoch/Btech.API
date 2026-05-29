package domain

import (
	"context"
	"time"
)

// Permission constants for type safety in code check references
const (
	PermissionDriversCreate   = "drivers:create"
	PermissionDriversRead     = "drivers:read"
	PermissionDriversDelete   = "drivers:delete"
	PermissionTripsCreate     = "trips:create"
	PermissionTripsRead       = "trips:read"
	PermissionTripsUpdate     = "trips:update"
	PermissionTripsDelete     = "trips:delete"
	PermissionIncidentsCreate = "incidents:create"
	PermissionIncidentsRead   = "incidents:read"
	PermissionSettingsManage  = "settings:manage"
	PermissionVehiclesCreate  = "vehicles:create"
	PermissionVehiclesRead    = "vehicles:read"
	PermissionVehiclesUpdate  = "vehicles:update"
	PermissionVehiclesDelete  = "vehicles:delete"
)

type Permission struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type RolePermission struct {
	Role         string    `json:"role"`
	PermissionID string    `json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type PermissionRepository interface {
	GetPermissionsByRole(ctx context.Context, role string) ([]string, error)
}
