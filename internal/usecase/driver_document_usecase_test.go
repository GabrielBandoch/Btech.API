package usecase_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

// Mock repos
type mockDocRepo struct {
	docs   map[string]domain.DriverDocument
	alerts []domain.DriverDocumentAlert
	files  map[string]domain.DriverDocumentFile
}

func (m *mockDocRepo) GetAll(ctx context.Context, orgID string, filter domain.DriverDocumentFilter) ([]domain.DriverDocument, error) {
	var list []domain.DriverDocument
	for _, d := range m.docs {
		if d.DriverID == filter.DriverID {
			list = append(list, d)
		}
	}
	return list, nil
}

func (m *mockDocRepo) GetByID(ctx context.Context, orgID string, id string) (domain.DriverDocument, error) {
	d, ok := m.docs[id]
	if !ok {
		return domain.DriverDocument{}, domain.ErrDriverDocumentNotFound
	}
	return d, nil
}

func (m *mockDocRepo) Create(ctx context.Context, orgID string, doc domain.DriverDocument) (domain.DriverDocument, error) {
	doc.ID = "DOC-TEST"
	m.docs[doc.ID] = doc
	return doc, nil
}

func (m *mockDocRepo) Update(ctx context.Context, orgID string, id string, doc domain.DriverDocument) (domain.DriverDocument, error) {
	existing := m.docs[id]
	if doc.DocumentNumber != "" {
		existing.DocumentNumber = doc.DocumentNumber
	}
	if doc.ExpirationDate != nil {
		existing.ExpirationDate = doc.ExpirationDate
	}
	if doc.Status != "" {
		existing.Status = doc.Status
	}
	m.docs[id] = existing
	return existing, nil
}

func (m *mockDocRepo) Delete(ctx context.Context, orgID string, id string) error {
	delete(m.docs, id)
	return nil
}

func (m *mockDocRepo) GetDashboardStats(ctx context.Context, orgID string, mandatoryTypes []string) (domain.DocumentDashboard, error) {
	return domain.DocumentDashboard{}, nil
}

func (m *mockDocRepo) CreateAlert(ctx context.Context, orgID string, alert domain.DriverDocumentAlert) (domain.DriverDocumentAlert, error) {
	alert.ID = "ALT-TEST"
	m.alerts = append(m.alerts, alert)
	return alert, nil
}

func (m *mockDocRepo) GetActiveAlerts(ctx context.Context, orgID string) ([]domain.DriverDocumentAlert, error) {
	return m.alerts, nil
}

func (m *mockDocRepo) ResolveAlertsForDocument(ctx context.Context, orgID string, docID string) error {
	m.alerts = nil
	return nil
}

func (m *mockDocRepo) CreateFile(ctx context.Context, orgID string, file domain.DriverDocumentFile) (domain.DriverDocumentFile, error) {
	file.ID = "FIL-TEST"
	m.files[file.ID] = file
	return file, nil
}

func (m *mockDocRepo) GetFileByID(ctx context.Context, orgID string, id string) (domain.DriverDocumentFile, error) {
	f, ok := m.files[id]
	if !ok {
		return domain.DriverDocumentFile{}, domain.ErrFileNotFound
	}
	return f, nil
}

func (m *mockDocRepo) DeleteFile(ctx context.Context, orgID string, id string) error {
	delete(m.files, id)
	return nil
}

type mockDriverRepo struct {
	drivers map[string]domain.Driver
}

func (m *mockDriverRepo) GetAll(ctx context.Context, orgID string) ([]domain.Driver, error) {
	return nil, nil
}

func (m *mockDriverRepo) GetByID(ctx context.Context, orgID string, id string) (domain.Driver, error) {
	d, ok := m.drivers[id]
	if !ok {
		return domain.Driver{}, domain.ErrDriverNotFound
	}
	return d, nil
}

func (m *mockDriverRepo) Create(ctx context.Context, orgID string, driver domain.Driver) (domain.Driver, error) {
	return driver, nil
}

func (m *mockDriverRepo) Count(ctx context.Context, orgID string) (int, error) {
	return 0, nil
}

func (m *mockDriverRepo) Update(ctx context.Context, orgID string, id string, driver domain.Driver) (domain.Driver, error) {
	existing := m.drivers[id]
	if driver.LicenseExpiry != "" {
		existing.LicenseExpiry = driver.LicenseExpiry
	}
	if driver.ToxicologyExpiry != "" {
		existing.ToxicologyExpiry = driver.ToxicologyExpiry
	}
	if driver.TrainingExpiry != "" {
		existing.TrainingExpiry = driver.TrainingExpiry
	}
	m.drivers[id] = existing
	return existing, nil
}

type mockAudit struct {
	actions []string
}

func (m *mockAudit) Log(ctx context.Context, action string, entityType string, entityID *string, metadata map[string]interface{}) {
	m.actions = append(m.actions, action)
}

func (m *mockAudit) GetLogsByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*domain.AuditLog, error) {
	return nil, nil
}

type mockStorage struct{}

func (m *mockStorage) Upload(ctx context.Context, bucket string, key string, file io.Reader, contentType string) (string, error) {
	return "http://localhost/files/" + key, nil
}

func (m *mockStorage) GetURL(ctx context.Context, bucket string, key string) (string, error) {
	return "http://localhost/files/" + key, nil
}

func (m *mockStorage) Delete(ctx context.Context, bucket string, key string) error {
	return nil
}

func TestDriverDocumentUseCase(t *testing.T) {
	docRepo := &mockDocRepo{
		docs:  make(map[string]domain.DriverDocument),
		files: make(map[string]domain.DriverDocumentFile),
	}
	driverRepo := &mockDriverRepo{
		drivers: map[string]domain.Driver{
			"DRV-1": {ID: "DRV-1", Name: "Carlos"},
		},
	}
	audit := &mockAudit{}
	store := &mockStorage{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	uc := usecase.NewDriverDocumentUseCase(docRepo, driverRepo, store, audit, logger)

	ctx := context.Background()
	orgID := "org-1"

	t.Run("Create_Validates_And_Recalculates_Status", func(t *testing.T) {
		expiry := time.Now().AddDate(0, 0, 45) // 45 days in future -> valid
		doc := domain.DriverDocument{
			DriverID:       "DRV-1",
			Type:           domain.DocTypeCNH,
			DocumentNumber: "12345",
			ExpirationDate: &expiry,
		}

		created, err := uc.CreateDocument(ctx, orgID, doc)
		if err != nil {
			t.Fatalf("unexpected error creating document: %v", err)
		}

		if created.Status != domain.DocStatusValid {
			t.Errorf("expected status 'valid', got '%s'", created.Status)
		}

		// Verify legacy driver sync
		d, _ := driverRepo.GetByID(ctx, orgID, "DRV-1")
		expectedDateStr := expiry.Format("2006-01-02")
		if d.LicenseExpiry != expectedDateStr {
			t.Errorf("expected legacy license_expiry to be '%s', got '%s'", expectedDateStr, d.LicenseExpiry)
		}
	})

	t.Run("Create_Triggers_Alert_Thresholds", func(t *testing.T) {
		// Clear alerts
		docRepo.alerts = nil
		expiryExpiring := time.Now().AddDate(0, 0, 10) // 10 days left -> status expiring_soon, triggers <=15 and <=30 alerts
		doc := domain.DriverDocument{
			DriverID:       "DRV-1",
			Type:           domain.DocTypeToxicology,
			DocumentNumber: "55555",
			ExpirationDate: &expiryExpiring,
		}

		created, err := uc.CreateDocument(ctx, orgID, doc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if created.Status != domain.DocStatusExpiringSoon {
			t.Errorf("expected status 'expiring_soon', got '%s'", created.Status)
		}

		if len(docRepo.alerts) == 0 {
			t.Error("expected compliance alerts to be triggered")
		} else {
			if docRepo.alerts[0].DaysRemaining != 15 {
				t.Errorf("expected alert threshold 15, got %d", docRepo.alerts[0].DaysRemaining)
			}
		}

		// Verify legacy toxicology sync
		d, _ := driverRepo.GetByID(ctx, orgID, "DRV-1")
		expectedDateStr := expiryExpiring.Format("2006-01-02")
		if d.ToxicologyExpiry != expectedDateStr {
			t.Errorf("expected legacy toxicology_expiry to be '%s', got '%s'", expectedDateStr, d.ToxicologyExpiry)
		}
	})

	t.Run("GetByID_Audit_Log_Event", func(t *testing.T) {
		audit.actions = nil
		_, err := uc.GetDocumentByID(ctx, orgID, "DOC-TEST")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasViewEvent := false
		for _, action := range audit.actions {
			if action == domain.EventDriverDocumentView {
				hasViewEvent = true
				break
			}
		}

		if !hasViewEvent {
			t.Error("expected driver_document.view audit event to be logged")
		}
	})
}
