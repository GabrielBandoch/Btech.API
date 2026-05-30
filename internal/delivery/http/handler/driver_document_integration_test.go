package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/btech/fleetcontrol-api/internal/config"
	delivery "github.com/btech/fleetcontrol-api/internal/delivery/http"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/handler"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/platform/database"
	"github.com/btech/fleetcontrol-api/internal/platform/logger"
	"github.com/btech/fleetcontrol-api/internal/platform/storage"
	"github.com/btech/fleetcontrol-api/internal/repository/postgres"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

type documentResponseEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

func TestDriverDocumentIntegration(t *testing.T) {
	// Setup test database environment
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/fleetcontrol?sslmode=disable")
	os.Setenv("JWT_SECRET", "supersecretchangeinproduction")
	os.Setenv("JWT_EXPIRES_IN", "15m")
	os.Setenv("BCRYPT_COST", "4")

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping driver document integration test: DATABASE_URL not set")
	}

	log := logger.New("development")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := database.NewPostgresDB(ctx, cfg.DatabaseURL, log)
	cancel()
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(cfg.DatabaseURL, "../../../../migrations", log); err != nil {
		t.Fatalf("failed to run database migrations: %v", err)
	}

	// Truncate tables for a clean test environment
	_, err = db.Pool.Exec(context.Background(), `
		TRUNCATE TABLE users, organizations, user_sessions, subscriptions, 
		drivers, driver_documents, driver_document_files, driver_document_alerts CASCADE`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	// Setup repositories
	userRepo := postgres.NewPostgresUserRepository(db.Pool)
	orgRepo := postgres.NewPostgresOrganizationRepository(db.Pool)
	permissionRepo := postgres.NewPostgresPermissionRepository(db.Pool)
	auditLogRepo := postgres.NewPostgresAuditLogRepository(db.Pool)
	sessionRepo := postgres.NewPostgresUserSessionRepository(db.Pool)
	subscriptionRepo := postgres.NewPostgresSubscriptionRepository(db.Pool)
	planRepo := postgres.NewPostgresPlanRepository(db.Pool)
	entitlementRepo := postgres.NewPostgresEntitlementRepository(db.Pool)
	driverRepo := postgres.NewPostgresDriverRepository(db.Pool)
	driverDocumentRepo := postgres.NewPostgresDriverDocumentRepository(db.Pool)

	// Platform local storage service client for tests
	storageService := storage.NewLocalFileStorage("test_uploads", "http://localhost/test_uploads")

	// UseCases
	auditUseCase := usecase.NewAuditUseCase(auditLogRepo, log)
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, 15*time.Minute, 24*time.Hour, 4)
	entitlementUseCase := usecase.NewEntitlementUseCase(subscriptionRepo, planRepo, entitlementRepo, auditUseCase)
	driverUseCase := usecase.NewDriverUseCase(driverRepo, entitlementUseCase, auditUseCase, auditLogRepo)
	driverDocumentUseCase := usecase.NewDriverDocumentUseCase(driverDocumentRepo, driverRepo, storageService, auditUseCase, log)

	// Handlers
	authHandler := handler.NewAuthHandler(authUseCase)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	driverDocumentHandler := handler.NewDriverDocumentHandler(driverDocumentUseCase)

	middleware.SetAuditUseCase(auditUseCase)
	authMiddleware := middleware.AuthMiddleware(authUseCase)
	rateLimiter := middleware.NewRateLimiter(100.0, 100.0)

	// Router setup
	router := delivery.NewRouter(cfg, driverHandler, nil, nil, authHandler, nil, nil, nil, driverDocumentHandler, authMiddleware, rateLimiter.Limit, entitlementUseCase, log)

	// Register test admin user
	regPayload := `{"name":"Ops Manager","email":"ops@btech.com","password":"SecurePassword123!","role":"admin"}`
	reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	router.ServeHTTP(wReg, reqReg)
	if wReg.Code != http.StatusCreated {
		t.Fatalf("failed to register user: %d. Body: %s", wReg.Code, wReg.Body.String())
	}

	// Login test admin user
	loginPayload := `{"email":"ops@btech.com","password":"SecurePassword123!"}`
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)
	if wLogin.Code != http.StatusOK {
		t.Fatalf("failed to login: %s", wLogin.Body.String())
	}

	var envelope authResponseEnvelope
	_ = json.Unmarshal(wLogin.Body.Bytes(), &envelope)
	var authResp dto.AuthResponse
	_ = json.Unmarshal(envelope.Data, &authResp)

	token := authResp.Token
	orgID := authResp.User.OrganizationID

	// Create test driver
	testDriver := domain.Driver{
		ID:             "DRV-OPS-1",
		OrganizationID: orgID,
		Name:           "Pedro Santos",
		Status:         "active",
	}
	_, err = driverRepo.Create(context.Background(), orgID, testDriver)
	if err != nil {
		t.Fatalf("failed to seed test driver: %v", err)
	}

	var documentID string

	// 1. Create Driver Document
	t.Run("Create_Driver_Document", func(t *testing.T) {
		payload := `{"type":"cnh","documentNumber":"54321A","issuingAuthority":"DETRAN SP","issueDate":"2020-01-01T00:00:00Z","expirationDate":"2030-01-01T00:00:00Z","notes":"CNH Categoria E"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers/DRV-OPS-1/documents", bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env documentResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var docResp dto.DriverDocumentResponse
		_ = json.Unmarshal(env.Data, &docResp)
		documentID = docResp.ID

		if docResp.DocumentNumber != "54321A" || docResp.Status != "valid" {
			t.Errorf("expected number 54321A and valid status, got %s and %s", docResp.DocumentNumber, docResp.Status)
		}
	})

	// 2. Fetch Details & Verify Sensitive View Audit Event
	t.Run("Get_Driver_Document_By_ID_And_Audit_View", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drivers/DRV-OPS-1/documents/"+documentID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Wait briefly for background audit logger
		time.Sleep(100 * time.Millisecond)

		// Assert view audit log exists in DB
		var count int
		err := db.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM audit_logs WHERE action = $1", domain.EventDriverDocumentView).Scan(&count)
		if err != nil {
			t.Fatalf("failed to query audit logs table: %v", err)
		}
		if count == 0 {
			t.Error("expected driver_document.view audit entry to be recorded on document detail fetch")
		}
	})

	// 3. Upload File Attachment (1:N verification)
	t.Run("Upload_Attachment", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "cnh_copy.pdf")
		part.Write([]byte("mock file pdf contents"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers/DRV-OPS-1/documents/"+documentID+"/upload", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created file upload, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env documentResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var docResp dto.DriverDocumentResponse
		_ = json.Unmarshal(env.Data, &docResp)

		if len(docResp.Files) != 1 {
			t.Errorf("expected 1 file attached to document DTO, got %d", len(docResp.Files))
		} else {
			if docResp.Files[0].FileName != "cnh_copy.pdf" {
				t.Errorf("expected filename 'cnh_copy.pdf', got '%s'", docResp.Files[0].FileName)
			}
		}
	})

	// Cleanup uploads
	os.RemoveAll("test_uploads")
}

// Struct layout mapping for login response parsing in integration test
type authResponseEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}
