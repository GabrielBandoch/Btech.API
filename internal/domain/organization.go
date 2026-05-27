package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrOrgUserNotFound      = errors.New("organization user mapping not found")
)

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type OrganizationUser struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	UserID         string    `json:"userId"`
	Role           string    `json:"role"` // "owner", "admin", "operator", "viewer"
	CreatedAt      time.Time `json:"createdAt"`
}

type OrganizationRepository interface {
	Create(ctx context.Context, org *Organization) error
	GetByID(ctx context.Context, id string) (*Organization, error)
	GetBySlug(ctx context.Context, slug string) (*Organization, error)
	CreateOrganizationUser(ctx context.Context, orgUser *OrganizationUser) error
	GetOrganizationUser(ctx context.Context, orgID, userID string) (*OrganizationUser, error)
}
