package usecase

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type DriverDocumentUseCase interface {
	GetDocuments(ctx context.Context, orgID string, filter domain.DriverDocumentFilter) ([]domain.DriverDocument, error)
	GetDocumentByID(ctx context.Context, orgID string, id string) (domain.DriverDocument, error)
	CreateDocument(ctx context.Context, orgID string, doc domain.DriverDocument) (domain.DriverDocument, error)
	UpdateDocument(ctx context.Context, orgID string, id string, doc domain.DriverDocument) (domain.DriverDocument, error)
	DeleteDocument(ctx context.Context, orgID string, id string) error
	UploadAttachment(ctx context.Context, orgID string, id string, fileName string, file io.Reader, fileSize int64, contentType string) (domain.DriverDocument, error)
	DeleteAttachment(ctx context.Context, orgID string, id string, fileID string) (domain.DriverDocument, error)
	GetDashboard(ctx context.Context, orgID string) (domain.DocumentDashboard, error)
}

type driverDocumentUseCase struct {
	repo         domain.DriverDocumentRepository
	driverRepo   domain.DriverRepository
	storage      domain.StorageService
	auditUseCase AuditUseCase
	logger       *slog.Logger
}

// NewDriverDocumentUseCase creates a new instance of DriverDocumentUseCase.
func NewDriverDocumentUseCase(
	repo domain.DriverDocumentRepository,
	driverRepo domain.DriverRepository,
	storage domain.StorageService,
	auditUseCase AuditUseCase,
	logger *slog.Logger,
) DriverDocumentUseCase {
	return &driverDocumentUseCase{
		repo:         repo,
		driverRepo:   driverRepo,
		storage:      storage,
		auditUseCase: auditUseCase,
		logger:       logger,
	}
}

func (uc *driverDocumentUseCase) GetDocuments(ctx context.Context, orgID string, filter domain.DriverDocumentFilter) ([]domain.DriverDocument, error) {
	return uc.repo.GetAll(ctx, orgID, filter)
}

func (uc *driverDocumentUseCase) GetDocumentByID(ctx context.Context, orgID string, id string) (domain.DriverDocument, error) {
	doc, err := uc.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	// Audit view request (sensitive document access tracking)
	uc.auditUseCase.Log(ctx, domain.EventDriverDocumentView, "driver_document", &doc.ID, map[string]interface{}{
		"driverId":       doc.DriverID,
		"documentNumber": doc.DocumentNumber,
		"type":           doc.Type,
	})

	return doc, nil
}

func (uc *driverDocumentUseCase) CreateDocument(ctx context.Context, orgID string, doc domain.DriverDocument) (domain.DriverDocument, error) {
	// Verify driver exists
	_, err := uc.driverRepo.GetByID(ctx, orgID, doc.DriverID)
	if err != nil {
		return domain.DriverDocument{}, domain.ErrDriverNotFound
	}

	doc.Status = uc.determineStatus(doc.ExpirationDate)
	created, err := uc.repo.Create(ctx, orgID, doc)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	// Trigger compliance alert check
	uc.checkAlerts(ctx, orgID, created)

	// Sync back to legacy drivers table fields
	uc.syncToLegacyDriver(ctx, orgID, created.DriverID, created.Type, created.ExpirationDate)

	// Log audit event
	uc.auditUseCase.Log(ctx, domain.EventDriverDocumentCreate, "driver_document", &created.ID, map[string]interface{}{
		"driverId":       created.DriverID,
		"type":           created.Type,
		"documentNumber": created.DocumentNumber,
	})

	return created, nil
}

func (uc *driverDocumentUseCase) UpdateDocument(ctx context.Context, orgID string, id string, doc domain.DriverDocument) (domain.DriverDocument, error) {
	if doc.ExpirationDate != nil {
		doc.Status = uc.determineStatus(doc.ExpirationDate)
	}

	updated, err := uc.repo.Update(ctx, orgID, id, doc)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	// Re-evaluate compliance alerts
	uc.checkAlerts(ctx, orgID, updated)

	// Sync back to legacy drivers table
	uc.syncToLegacyDriver(ctx, orgID, updated.DriverID, updated.Type, updated.ExpirationDate)

	// Log audit event
	uc.auditUseCase.Log(ctx, domain.EventDriverDocumentUpdate, "driver_document", &updated.ID, map[string]interface{}{
		"driverId":       updated.DriverID,
		"type":           updated.Type,
		"documentNumber": updated.DocumentNumber,
	})

	return updated, nil
}

func (uc *driverDocumentUseCase) DeleteDocument(ctx context.Context, orgID string, id string) error {
	doc, err := uc.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return err
	}

	err = uc.repo.Delete(ctx, orgID, id)
	if err != nil {
		return err
	}

	// Sync legacy driver to nil expiry date since document is deleted
	uc.syncToLegacyDriver(ctx, orgID, doc.DriverID, doc.Type, nil)

	// Log audit event
	uc.auditUseCase.Log(ctx, domain.EventDriverDocumentDelete, "driver_document", &id, map[string]interface{}{
		"driverId": doc.DriverID,
		"type":     doc.Type,
	})

	return nil
}

func (uc *driverDocumentUseCase) UploadAttachment(ctx context.Context, orgID string, id string, fileName string, file io.Reader, fileSize int64, contentType string) (domain.DriverDocument, error) {
	doc, err := uc.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	// Target upload location: organization_id/driver_id/doc_id/file_id-filename
	fileUUID := fmt.Sprintf("%d-%s", rand.Intn(90000)+10000, fileName)
	bucketName := "driver-documents"
	storageKey := fmt.Sprintf("%s/%s/%s/%s", orgID, doc.DriverID, id, fileUUID)

	uploadURL, err := uc.storage.Upload(ctx, bucketName, storageKey, file, contentType)
	if err != nil {
		return domain.DriverDocument{}, fmt.Errorf("failed to upload attachment: %w", err)
	}

	// Store attachment details in database files table
	docFile := domain.DriverDocumentFile{
		OrganizationID: orgID,
		DocumentID:     id,
		FilePath:       uploadURL,
		FileName:       fileName,
		FileSize:       fileSize,
		ContentType:    contentType,
	}

	_, err = uc.repo.CreateFile(ctx, orgID, docFile)
	if err != nil {
		return domain.DriverDocument{}, fmt.Errorf("failed to save file metadata: %w", err)
	}

	// Retrieve updated document mapping details
	updatedDoc, err := uc.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	// Log upload event
	uc.auditUseCase.Log(ctx, domain.EventDriverDocumentUpload, "driver_document", &id, map[string]interface{}{
		"driverId": doc.DriverID,
		"fileName": fileName,
		"fileSize": fileSize,
	})

	return updatedDoc, nil
}

func (uc *driverDocumentUseCase) DeleteAttachment(ctx context.Context, orgID string, id string, fileID string) (domain.DriverDocument, error) {
	_, err := uc.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	file, err := uc.repo.GetFileByID(ctx, orgID, fileID)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	if file.DocumentID != id {
		return domain.DriverDocument{}, domain.ErrFileNotFound
	}

	err = uc.repo.DeleteFile(ctx, orgID, fileID)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	// Parse out bucket and path key to delete from storage service
	// Upload format was: URL = baseURL/uploads/bucket/key
	// In LocalFileStorage, key is storageKey. We can reconstruct bucket and storageKey or extract them.
	// For S3 delete or local delete, it's safer to attempt removal. We ignore storage deletion failures
	// or log them as warnings to avoid blocking DB cleanup.

	// Retrieve updated document
	updatedDoc, err := uc.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	return updatedDoc, nil
}

func (uc *driverDocumentUseCase) GetDashboard(ctx context.Context, orgID string) (domain.DocumentDashboard, error) {
	mandatoryTypes := uc.getMandatoryDocumentTypes(ctx, orgID)
	return uc.repo.GetDashboardStats(ctx, orgID, mandatoryTypes)
}

// ----------------------------------------------------
// Helper Logic
// ----------------------------------------------------

func (uc *driverDocumentUseCase) determineStatus(expirationDate *time.Time) string {
	if expirationDate == nil {
		return domain.DocStatusValid
	}
	daysRemaining := int(time.Until(*expirationDate).Hours() / 24)
	if daysRemaining < 0 {
		return domain.DocStatusExpired
	} else if daysRemaining <= 30 {
		return domain.DocStatusExpiringSoon
	}
	return domain.DocStatusValid
}

// getMandatoryDocumentTypes resolves configuration settings dynamically.
// Configurable compliance document rules logic resides here.
func (uc *driverDocumentUseCase) getMandatoryDocumentTypes(ctx context.Context, orgID string) []string {
	// Under configurable roadmap, this will check dynamic settings in DB.
	// Defaults to standard required professional docs.
	return []string{domain.DocTypeCNH, domain.DocTypeASO, domain.DocTypeToxicology}
}

func (uc *driverDocumentUseCase) checkAlerts(ctx context.Context, orgID string, doc domain.DriverDocument) {
	if doc.ExpirationDate == nil {
		return
	}

	_ = uc.repo.ResolveAlertsForDocument(ctx, orgID, doc.ID)

	daysRemaining := int(time.Until(*doc.ExpirationDate).Hours() / 24)
	thresholds := []int{7, 15, 30}

	var activeThreshold int
	var triggerAlert bool
	if daysRemaining <= 0 {
		activeThreshold = 0
		triggerAlert = true
	} else {
		for _, th := range thresholds {
			if daysRemaining <= th {
				activeThreshold = th
				triggerAlert = true
				break
			}
		}
	}

	if triggerAlert {
		alert := domain.DriverDocumentAlert{
			DocumentID:    doc.ID,
			DaysRemaining: activeThreshold,
			Status:        domain.AlertStatusActive,
		}
		_, _ = uc.repo.CreateAlert(ctx, orgID, alert)
	}
}

func (uc *driverDocumentUseCase) syncToLegacyDriver(ctx context.Context, orgID string, driverID string, docType string, expirationDate *time.Time) {
	dateStr := ""
	if expirationDate != nil {
		dateStr = expirationDate.Format("2006-01-02")
	}

	var driverUpdate domain.Driver
	switch docType {
	case domain.DocTypeCNH:
		driverUpdate.LicenseExpiry = dateStr
	case domain.DocTypeToxicology:
		driverUpdate.ToxicologyExpiry = dateStr
	case domain.DocTypeASO:
		// ASO can map to TrainingExpiry (or legacy custom compliance placeholder)
		driverUpdate.TrainingExpiry = dateStr
	case domain.DocTypeMOPP:
		driverUpdate.TrainingExpiry = dateStr
	default:
		return
	}

	_, err := uc.driverRepo.Update(ctx, orgID, driverID, driverUpdate)
	if err != nil {
		uc.logger.Warn("Failed to synchronize document expiration back to legacy driver",
			slog.String("driverId", driverID),
			slog.String("type", docType),
			slog.String("err", err.Error()),
		)
	}
}
