package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	"github.com/btech/fleetcontrol-api/internal/repository/postgres"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

type apiResponseEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type mockUserRepoForLimit struct{}

func (m *mockUserRepoForLimit) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForLimit) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForLimit) Create(ctx context.Context, user *domain.User) error {
	return nil
}

func (m *mockUserRepoForLimit) CountByOrganization(ctx context.Context, orgID string) (int, error) {
	return 0, nil
}

type mockEntitlementUseCase struct{}

func (m *mockEntitlementUseCase) EvaluateFeature(ctx context.Context, orgID string, featureKey string) (bool, error) {
	return true, nil
}

func (m *mockEntitlementUseCase) EvaluateQuota(ctx context.Context, orgID string, quotaKey string, currentUsage int) (bool, error) {
	return true, nil
}

func (m *mockEntitlementUseCase) GetEntitlementValue(ctx context.Context, orgID string, key string) (string, error) {
	return "", nil
}

func TestAuthIntegration(t *testing.T) {
	// Programmatically set test environment variables to connect to port 5433
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/fleetcontrol?sslmode=disable")
	os.Setenv("JWT_SECRET", "supersecretchangeinproduction")
	os.Setenv("JWT_EXPIRES_IN", "2h")
	os.Setenv("BCRYPT_COST", "4")

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL not set")
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

	// Clear tables for clean test environment
	_, err = db.Pool.Exec(context.Background(), "TRUNCATE TABLE users, organizations, user_sessions CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate users table: %v", err)
	}

	// Setup UseCases, Router and Middleware
	userRepo := postgres.NewPostgresUserRepository(db.Pool)
	orgRepo := postgres.NewPostgresOrganizationRepository(db.Pool)
	permissionRepo := postgres.NewPostgresPermissionRepository(db.Pool)
	auditLogRepo := postgres.NewPostgresAuditLogRepository(db.Pool)
	sessionRepo := postgres.NewPostgresUserSessionRepository(db.Pool)
	driverRepo := postgres.NewPostgresDriverRepository(db.Pool)
	vehicleRepo := postgres.NewPostgresVehicleRepository(db.Pool)
	tripRepo := postgres.NewPostgresTripRepository(db.Pool)
	incidentRepo := postgres.NewPostgresIncidentRepository(db.Pool)

	jwtExpires := 2 * time.Second
	auditUseCase := usecase.NewAuditUseCase(auditLogRepo, log)
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, jwtExpires, 7*24*time.Hour, 4) // cost = 4 for fast testing
	
	mockEntitlementUC := &mockEntitlementUseCase{}
	driverUseCase := usecase.NewDriverUseCase(driverRepo, mockEntitlementUC, auditUseCase)
	tripUseCase := usecase.NewTripUseCase(tripRepo, auditUseCase)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo, auditUseCase)
	vehicleUseCase := usecase.NewVehicleUseCase(vehicleRepo, auditUseCase)

	authHandler := handler.NewAuthHandler(authUseCase)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)
	vehicleHandler := handler.NewVehicleHandler(vehicleUseCase)

	middleware.SetAuditUseCase(auditUseCase)
	authMiddleware := middleware.AuthMiddleware(authUseCase)
	
	// Large capacity rate limiter to prevent rate limit blocks during main test suite
	rateLimiter := middleware.NewRateLimiter(100.0, 100.0)

	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, vehicleHandler, nil, nil, authMiddleware, rateLimiter.Limit, mockEntitlementUC, log)

	// Test 1: POST /auth/register - Success (Satisfies new password policy)
	t.Run("Register_Success", func(t *testing.T) {
		regPayload := `{"name":"Ricardo Silva","email":"RICARDO@btech.com","password":"SecurePassword123!","role":"admin"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !res.Success {
			t.Errorf("expected Success to be true, got false. Error: %s", res.Error)
		}

		// Ensure password hash is not in response
		var user dto.UserResponse
		if err := json.Unmarshal(res.Data, &user); err != nil {
			t.Fatalf("failed to parse user response data: %v", err)
		}

		if user.Name != "Ricardo Silva" {
			t.Errorf("expected user name 'Ricardo Silva', got '%s'", user.Name)
		}

		// Check lowercase normalization
		if user.Email != "ricardo@btech.com" {
			t.Errorf("expected email to be lowercased to 'ricardo@btech.com', got '%s'", user.Email)
		}

		// Verify no password_hash or similar fields are serialized
		if strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), "PasswordHash") {
			t.Error("leak: response payload contains password hash references")
		}
	})

	// Test 2: POST /auth/register - Password Policy Rejections
	t.Run("Register_PasswordPolicyRejections", func(t *testing.T) {
		testCases := []struct {
			name        string
			password    string
			expectedErr string
		}{
			{"Too short", "Sh1!", "password must be at least 8 characters long"},
			{"No uppercase", "secure123!", "password must contain at least one uppercase letter"},
			{"No lowercase", "SECURE123!", "password must contain at least one lowercase letter"},
			{"No digit", "SecurePass!", "password must contain at least one digit"},
			{"No special", "SecurePass123", "password must contain at least one special character"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				payload := map[string]string{
					"name":     "Bob",
					"email":    "bob@example.com",
					"password": tc.password,
					"role":     "operator",
				}
				body, _ := json.Marshal(payload)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status 400 Bad Request, got %d", w.Code)
				}

				var res apiResponseEnvelope
				_ = json.Unmarshal(w.Body.Bytes(), &res)
				if res.Success {
					t.Error("expected Success to be false for weak password, got true")
				}
				if res.Error != tc.expectedErr {
					t.Errorf("expected error '%s', got '%s'", tc.expectedErr, res.Error)
				}
			})
		}
	})

	// Test 3: POST /auth/register - Duplicate Email
	t.Run("Register_DuplicateEmail", func(t *testing.T) {
		regPayload := `{"name":"Another Ricardo","email":"ricardo@btech.com","password":"SecurePassword123!","role":"operator"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if res.Success {
			t.Error("expected Success to be false for duplicate email, got true")
		}

		if res.Error != "email already registered" {
			t.Errorf("expected error message 'email already registered', got '%s'", res.Error)
		}
	})

	// Test 4: POST /auth/login - Success
	var jwtToken string
	t.Run("Login_Success", func(t *testing.T) {
		loginPayload := `{"email":"RICARDO@BTECH.COM","password":"SecurePassword123!"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 OK, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !res.Success {
			t.Errorf("expected Success to be true, got false. Error: %s", res.Error)
		}

		var loginData dto.AuthResponse
		if err := json.Unmarshal(res.Data, &loginData); err != nil {
			t.Fatalf("failed to parse login response data: %v", err)
		}

		if loginData.Token == "" {
			t.Error("expected JWT token to be generated, got empty string")
		}

		jwtToken = loginData.Token
	})

	// Test 5: POST /auth/login - Invalid Password
	t.Run("Login_InvalidPassword", func(t *testing.T) {
		loginPayload := `{"email":"ricardo@btech.com","password":"wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	// Test 6: GET /auth/me - Success with valid token
	t.Run("GetMe_Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 OK, got %d", w.Code)
		}
	})

	// Test 7: GET /auth/me - Unauthorized with expired token
	t.Run("GetMe_ExpiredToken", func(t *testing.T) {
		time.Sleep(2500 * time.Millisecond)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	// Test 8: Granular permissions verification
	t.Run("GranularPermissionsChecks", func(t *testing.T) {
		// 1. Create a viewer user
		regPayloadViewer := `{"name":"Viewer User","email":"viewer@btech.com","password":"SecurePassword123!","role":"viewer"}`
		reqRegViewer := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayloadViewer))
		reqRegViewer.Header.Set("Content-Type", "application/json")
		wRegViewer := httptest.NewRecorder()
		router.ServeHTTP(wRegViewer, reqRegViewer)
		if wRegViewer.Code != http.StatusCreated {
			t.Fatalf("failed to register viewer user: %d", wRegViewer.Code)
		}

		loginPayloadViewer := `{"email":"viewer@btech.com","password":"SecurePassword123!"}`
		reqLoginViewer := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayloadViewer))
		reqLoginViewer.Header.Set("Content-Type", "application/json")
		wLoginViewer := httptest.NewRecorder()
		router.ServeHTTP(wLoginViewer, reqLoginViewer)
		
		var resViewer apiResponseEnvelope
		_ = json.Unmarshal(wLoginViewer.Body.Bytes(), &resViewer)
		var loginDataViewer dto.AuthResponse
		_ = json.Unmarshal(resViewer.Data, &loginDataViewer)
		viewerToken := loginDataViewer.Token

		// 2. Create an operator user
		regPayloadOperator := `{"name":"Operator User","email":"operator@btech.com","password":"SecurePassword123!","role":"operator"}`
		reqRegOperator := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayloadOperator))
		reqRegOperator.Header.Set("Content-Type", "application/json")
		wRegOperator := httptest.NewRecorder()
		router.ServeHTTP(wRegOperator, reqRegOperator)
		if wRegOperator.Code != http.StatusCreated {
			t.Fatalf("failed to register operator user: %d", wRegOperator.Code)
		}

		loginPayloadOperator := `{"email":"operator@btech.com","password":"SecurePassword123!"}`
		reqLoginOperator := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayloadOperator))
		reqLoginOperator.Header.Set("Content-Type", "application/json")
		wLoginOperator := httptest.NewRecorder()
		router.ServeHTTP(wLoginOperator, reqLoginOperator)
		
		var resOperator apiResponseEnvelope
		_ = json.Unmarshal(wLoginOperator.Body.Bytes(), &resOperator)
		var loginDataOperator dto.AuthResponse
		_ = json.Unmarshal(resOperator.Data, &loginDataOperator)
		operatorToken := loginDataOperator.Token

		// Assertions:
		// A. Viewer tries to POST to /drivers (requires drivers:create) -> must return 403 Forbidden
		reqPostDriverViewer := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(`{}`))
		reqPostDriverViewer.Header.Set("Authorization", "Bearer "+viewerToken)
		reqPostDriverViewer.Header.Set("Content-Type", "application/json")
		wPostDriverViewer := httptest.NewRecorder()
		router.ServeHTTP(wPostDriverViewer, reqPostDriverViewer)
		if wPostDriverViewer.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for viewer creating driver, got %d", wPostDriverViewer.Code)
		}

		// B. Operator tries to POST to /drivers (requires drivers:create) -> must return 403 Forbidden
		reqPostDriverOperator := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(`{}`))
		reqPostDriverOperator.Header.Set("Authorization", "Bearer "+operatorToken)
		reqPostDriverOperator.Header.Set("Content-Type", "application/json")
		wPostDriverOperator := httptest.NewRecorder()
		router.ServeHTTP(wPostDriverOperator, reqPostDriverOperator)
		if wPostDriverOperator.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for operator creating driver, got %d", wPostDriverOperator.Code)
		}

		// C. Admin (Ricardo) tries to POST to /drivers (requires drivers:create) -> should NOT return 403
		reqPostDriverAdmin := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(`{"name": "New Driver", "role": "operator"}`))
		reqPostDriverAdmin.Header.Set("Authorization", "Bearer "+jwtToken)
		reqPostDriverAdmin.Header.Set("Content-Type", "application/json")
		wPostDriverAdmin := httptest.NewRecorder()
		router.ServeHTTP(wPostDriverAdmin, reqPostDriverAdmin)
		if wPostDriverAdmin.Code == http.StatusForbidden {
			t.Errorf("expected admin to bypass drivers:create check, got 403 Forbidden")
		}

		// D. Operator tries to PUT to /trips/trip-123 (requires trips:update) -> should NOT return 403 (could be 400 or 404, but NOT 403)
		reqPutTripOperator := httptest.NewRequest(http.MethodPut, "/api/v1/trips/trip-123", bytes.NewBufferString(`{}`))
		reqPutTripOperator.Header.Set("Authorization", "Bearer "+operatorToken)
		reqPutTripOperator.Header.Set("Content-Type", "application/json")
		wPutTripOperator := httptest.NewRecorder()
		router.ServeHTTP(wPutTripOperator, reqPutTripOperator)
		if wPutTripOperator.Code == http.StatusForbidden {
			t.Errorf("expected operator to bypass trips:update check, got 403 Forbidden")
		}

		// E. Viewer tries to PUT to /trips/trip-123 (requires trips:update) -> must return 403 Forbidden
		reqPutTripViewer := httptest.NewRequest(http.MethodPut, "/api/v1/trips/trip-123", bytes.NewBufferString(`{}`))
		reqPutTripViewer.Header.Set("Authorization", "Bearer "+viewerToken)
		reqPutTripViewer.Header.Set("Content-Type", "application/json")
		wPutTripViewer := httptest.NewRecorder()
		router.ServeHTTP(wPutTripViewer, reqPutTripViewer)
		if wPutTripViewer.Code != http.StatusForbidden {
			t.Errorf("expected viewer to get 403 Forbidden for trips:update, got %d", wPutTripViewer.Code)
		}
	})

	// Test 9: Audit log persistence and multi-tenant isolation
	t.Run("AuditLogsPersistence", func(t *testing.T) {
		// Clear audit logs first to have clean stats
		_, err := db.Pool.Exec(context.Background(), "TRUNCATE TABLE audit_logs CASCADE")
		if err != nil {
			t.Fatalf("failed to truncate audit_logs table: %v", err)
		}

		// Register a new user (triggers EventUserRegister)
		regPayload := `{"name":"Audited User","email":"audited@btech.com","password":"SecurePassword123!","role":"operator"}`
		reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
		reqReg.Header.Set("Content-Type", "application/json")
		reqReg.Header.Set("User-Agent", "Go-Integration-Test-UA")
		reqReg.Header.Set("X-Real-IP", "1.2.3.4")
		wReg := httptest.NewRecorder()
		router.ServeHTTP(wReg, reqReg)
		if wReg.Code != http.StatusCreated {
			t.Fatalf("failed to register user for audit: %d", wReg.Code)
		}

		// Login the user (triggers EventUserLogin)
		loginPayload := `{"email":"audited@btech.com","password":"SecurePassword123!"}`
		reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
		reqLogin.Header.Set("Content-Type", "application/json")
		reqLogin.Header.Set("User-Agent", "Go-Integration-Test-UA")
		reqLogin.Header.Set("X-Real-IP", "1.2.3.4")
		wLogin := httptest.NewRecorder()
		router.ServeHTTP(wLogin, reqLogin)
		if wLogin.Code != http.StatusOK {
			t.Fatalf("failed to login user for audit: %d", wLogin.Code)
		}

		var res apiResponseEnvelope
		_ = json.Unmarshal(wLogin.Body.Bytes(), &res)
		var loginData dto.AuthResponse
		_ = json.Unmarshal(res.Data, &loginData)
		token := loginData.Token
		orgID := loginData.User.OrganizationID

		// Trigger permission denied (triggers EventPermissionDenied)
		reqPostDriver := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(`{}`))
		reqPostDriver.Header.Set("Authorization", "Bearer "+token)
		reqPostDriver.Header.Set("Content-Type", "application/json")
		reqPostDriver.Header.Set("User-Agent", "Go-Integration-Test-UA")
		reqPostDriver.Header.Set("X-Real-IP", "1.2.3.4")
		wPostDriver := httptest.NewRecorder()
		router.ServeHTTP(wPostDriver, reqPostDriver)
		if wPostDriver.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", wPostDriver.Code)
		}

		// Wait briefly for background worker queue to write to database
		time.Sleep(100 * time.Millisecond)

		// Check database directly
		var count int
		err = db.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM audit_logs").Scan(&count)
		if err != nil {
			t.Fatalf("failed to query audit log count: %v", err)
		}

		if count < 3 {
			t.Errorf("expected at least 3 audit logs in database, got %d", count)
		}

		// Verify fields of user.login and user.register
		var action string
		var ipAddress string
		var userAgent *string
		err = db.Pool.QueryRow(context.Background(), 
			"SELECT action, ip_address, user_agent FROM audit_logs WHERE action = $1 LIMIT 1", 
			domain.EventUserLogin).Scan(&action, &ipAddress, &userAgent)
		if err != nil {
			t.Fatalf("failed to fetch user.login audit log: %v", err)
		}

		if ipAddress != "1.2.3.4" {
			t.Errorf("expected IP Address '1.2.3.4', got '%s'", ipAddress)
		}
		if userAgent == nil || *userAgent != "Go-Integration-Test-UA" {
			t.Errorf("expected User Agent 'Go-Integration-Test-UA', got '%v'", userAgent)
		}

		// Verify get logs endpoint or organization query isolation
		logs, err := auditUseCase.GetLogsByOrganization(context.Background(), orgID, 10, 0)
		if err != nil {
			t.Fatalf("failed to get logs by organization: %v", err)
		}

		if len(logs) == 0 {
			t.Error("expected to find at least one audit log for organization")
		}

		for _, l := range logs {
			if l.OrganizationID == nil || *l.OrganizationID != orgID {
				t.Errorf("expected audit log organization ID to be %s, got %v", orgID, l.OrganizationID)
			}
		}
	})
}

type mockOrgRepoForLimit struct{}

func (m *mockOrgRepoForLimit) Create(ctx context.Context, org *domain.Organization) error { return nil }
func (m *mockOrgRepoForLimit) GetByID(ctx context.Context, id string) (*domain.Organization, error) {
	return &domain.Organization{ID: id, Name: "Mock Org", Slug: "mock-org"}, nil
}
func (m *mockOrgRepoForLimit) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	return &domain.Organization{ID: "mock-id", Name: "Mock Org", Slug: slug}, nil
}
func (m *mockOrgRepoForLimit) CreateOrganizationUser(ctx context.Context, orgUser *domain.OrganizationUser) error {
	return nil
}
func (m *mockOrgRepoForLimit) GetOrganizationUser(ctx context.Context, orgID, userID string) (*domain.OrganizationUser, error) {
	return &domain.OrganizationUser{ID: "mock-mapping-id", OrganizationID: orgID, UserID: userID, Role: "operator"}, nil
}

type mockAuditUseCaseForLimit struct{}
func (m *mockAuditUseCaseForLimit) Log(ctx context.Context, action string, entityType string, entityID *string, metadata map[string]interface{}) {}
func (m *mockAuditUseCaseForLimit) GetLogsByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*domain.AuditLog, error) {
	return []*domain.AuditLog{}, nil
}

type mockPermissionRepoForLimit struct{}
func (m *mockPermissionRepoForLimit) GetPermissionsByRole(ctx context.Context, role string) ([]string, error) {
	return []string{}, nil
}

type mockUserSessionRepoForLimit struct{}
func (m *mockUserSessionRepoForLimit) GetByID(ctx context.Context, id string) (*domain.UserSession, error) {
	return nil, domain.ErrSessionNotFound
}
func (m *mockUserSessionRepoForLimit) Create(ctx context.Context, s *domain.UserSession) error {
	return nil
}
func (m *mockUserSessionRepoForLimit) Update(ctx context.Context, s *domain.UserSession) error {
	return nil
}
func (m *mockUserSessionRepoForLimit) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *mockUserSessionRepoForLimit) ListByUserID(ctx context.Context, userID, orgID string) ([]*domain.UserSession, error) {
	return []*domain.UserSession{}, nil
}
func (m *mockUserSessionRepoForLimit) RevokeAllByUserID(ctx context.Context, userID string) error {
	return nil
}

func TestAuthRateLimiting(t *testing.T) {
	log := logger.New("development")
	cfg := config.Load()
	
	// Create mock dependencies to prevent DB connection panics
	userRepo := &mockUserRepoForLimit{}
	orgRepo := &mockOrgRepoForLimit{}
	permissionRepo := &mockPermissionRepoForLimit{}
	sessionRepo := &mockUserSessionRepoForLimit{}
	auditUseCase := &mockAuditUseCaseForLimit{}
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, 1*time.Hour, 7*24*time.Hour, 4)
	authHandler := handler.NewAuthHandler(authUseCase)
	
	driverRepo := postgres.NewPostgresDriverRepository(nil)
	vehicleRepo := postgres.NewPostgresVehicleRepository(nil)
	tripRepo := postgres.NewPostgresTripRepository(nil)
	incidentRepo := postgres.NewPostgresIncidentRepository(nil)
	mockEntitlementUC := &mockEntitlementUseCase{}
	driverUseCase := usecase.NewDriverUseCase(driverRepo, mockEntitlementUC, auditUseCase)
	tripUseCase := usecase.NewTripUseCase(tripRepo, auditUseCase)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo, auditUseCase)
	vehicleUseCase := usecase.NewVehicleUseCase(vehicleRepo, auditUseCase)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)
	vehicleHandler := handler.NewVehicleHandler(vehicleUseCase)
	
	authMiddleware := middleware.AuthMiddleware(authUseCase)

	// Rate limiter with capacity = 2, rate = 0 (no token refills during test)
	rateLimiter := middleware.NewRateLimiter(0.0, 2.0)
	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, vehicleHandler, nil, nil, authMiddleware, rateLimiter.Limit, mockEntitlementUC, log)

	payload := `{"email":"test@example.com","password":"Password123!"}`
	
	// 1st request - should PASS rate limiter (returns 401 Unauthorized because user doesn't exist, but NOT 429)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(payload))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code == http.StatusTooManyRequests {
		t.Error("1st request should not be rate limited")
	}

	// 2nd request - should PASS rate limiter
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(payload))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code == http.StatusTooManyRequests {
		t.Error("2nd request should not be rate limited")
	}

	// 3rd request - should FAIL rate limiter and return 429 Too Many Requests
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(payload))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("expected 3rd request to be rate limited with status 429, got %d", w3.Code)
	}

	var res apiResponseEnvelope
	if err := json.Unmarshal(w3.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse rate limit response envelope: %v", err)
	}

	if res.Success {
		t.Error("expected envelope success to be false for 429, got true")
	}

	if res.Error != "too many requests - please try again later" {
		t.Errorf("expected error message 'too many requests - please try again later', got '%s'", res.Error)
	}
}
