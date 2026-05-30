package domain

import (
	"context"
	"errors"
	"time"
)

// Document Types
const (
	DocTypeCNH           = "cnh"
	DocTypeMOPP          = "mopp"
	DocTypeASO           = "aso"
	DocTypeToxicology    = "toxicology"
	DocTypeCertification = "certification"
	DocTypeCustom        = "custom"
)

// Document Statuses
const (
	DocStatusValid        = "valid"
	DocStatusExpiringSoon = "expiring_soon"
	DocStatusExpired      = "expired"
)

// Alert Statuses
const (
	AlertStatusActive   = "active"
	AlertStatusResolved = "resolved"
)

// Permission constants for Driver Documents module
const (
	PermissionDriverDocumentsCreate = "driver_documents:create"
	PermissionDriverDocumentsRead   = "driver_documents:read"
	PermissionDriverDocumentsUpdate = "driver_documents:update"
	PermissionDriverDocumentsDelete = "driver_documents:delete"
)

// Domain Errors
var (
	ErrDriverDocumentNotFound = errors.New("driver document not found")
	ErrDriverNotFound         = errors.New("driver not found")
	ErrInvalidDocumentType    = errors.New("invalid document type")
	ErrInvalidDocumentStatus  = errors.New("invalid document status")
	ErrFileNotFound           = errors.New("document file not found")
)

// DriverDocument represents a driver document's metadata.
type DriverDocument struct {
	ID               string               `json:"id"`
	OrganizationID   string               `json:"organizationId"`
	DriverID         string               `json:"driverId"`
	Type             string               `json:"type"` // cnh, mopp, aso, toxicology, certification, custom
	DocumentNumber   string               `json:"documentNumber"`
	IssuingAuthority string               `json:"issuingAuthority"`
	IssueDate        *time.Time           `json:"issueDate,omitempty"`
	ExpirationDate   *time.Time           `json:"expirationDate,omitempty"`
	Notes            string               `json:"notes"`
	Status           string               `json:"status"` // valid, expiring_soon, expired
	Files            []DriverDocumentFile `json:"files"`
	CreatedAt        time.Time            `json:"createdAt"`
	UpdatedAt        time.Time            `json:"updatedAt"`
	DeletedAt        *time.Time           `json:"deletedAt,omitempty"`
}

// DriverDocumentFile represents an uploaded attachment details.
type DriverDocumentFile struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	DocumentID     string     `json:"documentId"`
	FilePath       string     `json:"filePath"`
	FileName       string     `json:"fileName"`
	FileSize       int64      `json:"fileSize"`
	ContentType    string     `json:"contentType"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt,omitempty"`
}

// DriverDocumentAlert represents a computed threshold alert.
type DriverDocumentAlert struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organizationId"`
	DocumentID     string     `json:"documentId"`
	DaysRemaining  int        `json:"daysRemaining"` // 30, 15, 7, 0 (expired)
	Status         string     `json:"status"`        // active, resolved
	CreatedAt      time.Time  `json:"createdAt"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
}

// DriverDocumentFilter holds criteria to query documents.
type DriverDocumentFilter struct {
	DriverID        string
	Type            string
	Status          string
	ExpirationStart *time.Time
	ExpirationEnd   *time.Time
}

// DocumentDashboard stores statistics for organization metrics.
type DocumentDashboard struct {
	TotalDocuments       int     `json:"totalDocuments"`
	ValidDocuments       int     `json:"validDocuments"`
	ExpiringDocuments    int     `json:"expiringDocuments"`
	ExpiredDocuments     int     `json:"expiredDocuments"`
	CompliancePercentage float64 `json:"compliancePercentage"`
}

// DriverDocumentRepository defines the database interface.
type DriverDocumentRepository interface {
	GetAll(ctx context.Context, orgID string, filter DriverDocumentFilter) ([]DriverDocument, error)
	GetByID(ctx context.Context, orgID string, id string) (DriverDocument, error)
	Create(ctx context.Context, orgID string, doc DriverDocument) (DriverDocument, error)
	Update(ctx context.Context, orgID string, id string, doc DriverDocument) (DriverDocument, error)
	Delete(ctx context.Context, orgID string, id string) error

	// Dashboard & Compliance
	GetDashboardStats(ctx context.Context, orgID string, mandatoryTypes []string) (DocumentDashboard, error)

	// Expiration Alert Operations
	CreateAlert(ctx context.Context, orgID string, alert DriverDocumentAlert) (DriverDocumentAlert, error)
	GetActiveAlerts(ctx context.Context, orgID string) ([]DriverDocumentAlert, error)
	ResolveAlertsForDocument(ctx context.Context, orgID string, docID string) error

	// File attachments operations
	CreateFile(ctx context.Context, orgID string, file DriverDocumentFile) (DriverDocumentFile, error)
	GetFileByID(ctx context.Context, orgID string, id string) (DriverDocumentFile, error)
	DeleteFile(ctx context.Context, orgID string, id string) error
}
