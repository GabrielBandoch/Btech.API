package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
	ErrSessionRevoked  = errors.New("session has been revoked")
)

type UserSession struct {
	ID             string     `json:"id"`
	UserID         string     `json:"userId"`
	OrganizationID string     `json:"organizationId"`
	TokenHash      string     `json:"-"`
	TokenVersion   int        `json:"tokenVersion"`
	UserAgent      string     `json:"userAgent"`
	IPAddress      string     `json:"ipAddress"`
	IsRevoked      bool       `json:"isRevoked"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	LastSeenAt     *time.Time `json:"lastSeenAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type UserSessionRepository interface {
	GetByID(ctx context.Context, id string) (*UserSession, error)
	Create(ctx context.Context, session *UserSession) error
	Update(ctx context.Context, session *UserSession) error
	Delete(ctx context.Context, id string) error
	ListByUserID(ctx context.Context, userID, orgID string) ([]*UserSession, error)
	RevokeAllByUserID(ctx context.Context, userID string) error
}
