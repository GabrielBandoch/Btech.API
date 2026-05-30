package dto

import (
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type CreateDriverDocumentRequest struct {
	Type             string     `json:"type"` // required
	DocumentNumber   string     `json:"documentNumber"` // required
	IssuingAuthority string     `json:"issuingAuthority,omitempty"`
	IssueDate        *time.Time `json:"issueDate,omitempty"`
	ExpirationDate   *time.Time `json:"expirationDate,omitempty"`
	Notes            string     `json:"notes,omitempty"`
}

type UpdateDriverDocumentRequest struct {
	DocumentNumber   string     `json:"documentNumber,omitempty"`
	IssuingAuthority string     `json:"issuingAuthority,omitempty"`
	IssueDate        *time.Time `json:"issueDate,omitempty"`
	ExpirationDate   *time.Time `json:"expirationDate,omitempty"`
	Notes            string     `json:"notes,omitempty"`
}

type DriverDocumentFileResponse struct {
	ID          string    `json:"id"`
	DocumentID  string    `json:"documentId"`
	FilePath    string    `json:"filePath"`
	FileName    string    `json:"fileName"`
	FileSize    int64     `json:"fileSize"`
	ContentType string    `json:"contentType"`
	CreatedAt   time.Time `json:"createdAt"`
}

type DriverDocumentResponse struct {
	ID               string                       `json:"id"`
	DriverID         string                       `json:"driverId"`
	Type             string                       `json:"type"`
	DocumentNumber   string                       `json:"documentNumber"`
	IssuingAuthority string                       `json:"issuingAuthority,omitempty"`
	IssueDate        *time.Time                   `json:"issueDate,omitempty"`
	ExpirationDate   *time.Time                   `json:"expirationDate,omitempty"`
	Notes            string                       `json:"notes,omitempty"`
	Status           string                       `json:"status"` // valid, expiring_soon, expired
	Files            []DriverDocumentFileResponse `json:"files"`
	CreatedAt        time.Time                    `json:"createdAt"`
	UpdatedAt        time.Time                    `json:"updatedAt"`
}

type DocumentDashboardResponse struct {
	TotalDocuments       int     `json:"totalDocuments"`
	ValidDocuments       int     `json:"validDocuments"`
	ExpiringDocuments    int     `json:"expiringDocuments"`
	ExpiredDocuments     int     `json:"expiredDocuments"`
	CompliancePercentage float64 `json:"compliancePercentage"`
}

func DriverDocumentFileToResponse(f domain.DriverDocumentFile) DriverDocumentFileResponse {
	return DriverDocumentFileResponse{
		ID:          f.ID,
		DocumentID:  f.DocumentID,
		FilePath:    f.FilePath,
		FileName:    f.FileName,
		FileSize:    f.FileSize,
		ContentType: f.ContentType,
		CreatedAt:   f.CreatedAt,
	}
}

func DriverDocumentToResponse(d domain.DriverDocument) DriverDocumentResponse {
	filesRes := []DriverDocumentFileResponse{}
	for _, f := range d.Files {
		filesRes = append(filesRes, DriverDocumentFileToResponse(f))
	}

	return DriverDocumentResponse{
		ID:               d.ID,
		DriverID:         d.DriverID,
		Type:             d.Type,
		DocumentNumber:   d.DocumentNumber,
		IssuingAuthority: d.IssuingAuthority,
		IssueDate:        d.IssueDate,
		ExpirationDate:   d.ExpirationDate,
		Notes:            d.Notes,
		Status:           d.Status,
		Files:            filesRes,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
}

func DriverDocumentToResponseList(list []domain.DriverDocument) []DriverDocumentResponse {
	res := make([]DriverDocumentResponse, 0, len(list))
	for _, d := range list {
		res = append(res, DriverDocumentToResponse(d))
	}
	return res
}

func DocumentDashboardToResponse(d domain.DocumentDashboard) DocumentDashboardResponse {
	return DocumentDashboardResponse{
		TotalDocuments:       d.TotalDocuments,
		ValidDocuments:       d.ValidDocuments,
		ExpiringDocuments:    d.ExpiringDocuments,
		ExpiredDocuments:     d.ExpiredDocuments,
		CompliancePercentage: d.CompliancePercentage,
	}
}
