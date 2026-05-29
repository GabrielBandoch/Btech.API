package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/btech/fleetcontrol-api/internal/repository/postgres"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

func TestBillingIntegration(t *testing.T) {
	// Setup test environment
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/fleetcontrol?sslmode=disable")
	os.Setenv("JWT_SECRET", "supersecretchangeinproduction")
	os.Setenv("JWT_EXPIRES_IN", "15m")
	os.Setenv("BCRYPT_COST", "4")

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping billing integration test: DATABASE_URL not set")
	}

	log := logger.New("development")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := database.NewPostgresDB(ctx, cfg.DatabaseURL, log)
	cancel()
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Run migrations to ensure schema is fully up to date
	if err := database.RunMigrations(cfg.DatabaseURL, "../../../../migrations", log); err != nil {
		t.Fatalf("failed to run database migrations: %v", err)
	}

	// Truncate tables for a clean slate
	_, err = db.Pool.Exec(context.Background(), `
		TRUNCATE TABLE users, organizations, user_sessions, subscriptions, 
		organization_entitlement_overrides, subscription_event_history, usage_counters CASCADE`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	// Setup repositories
	userRepo := postgres.NewPostgresUserRepository(db.Pool)
	orgRepo := postgres.NewPostgresOrganizationRepository(db.Pool)
	permissionRepo := postgres.NewPostgresPermissionRepository(db.Pool)
	auditLogRepo := postgres.NewPostgresAuditLogRepository(db.Pool)
	sessionRepo := postgres.NewPostgresUserSessionRepository(db.Pool)
	planRepo := postgres.NewPostgresPlanRepository(db.Pool)
	subscriptionRepo := postgres.NewPostgresSubscriptionRepository(db.Pool)
	entitlementRepo := postgres.NewPostgresEntitlementRepository(db.Pool)
	usageRepo := postgres.NewPostgresUsageCounterRepository(db.Pool)
	driverRepo := postgres.NewPostgresDriverRepository(db.Pool)
	vehicleRepo := postgres.NewPostgresVehicleRepository(db.Pool)
	tripRepo := postgres.NewPostgresTripRepository(db.Pool)
	incidentRepo := postgres.NewPostgresIncidentRepository(db.Pool)

	// Setup UseCases, Router, Middleware
	auditUseCase := usecase.NewAuditUseCase(auditLogRepo, log)
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, 15*time.Minute, 24*time.Hour, 4)
	billingUseCase := usecase.NewBillingUseCase(subscriptionRepo, planRepo, auditUseCase, log)
	entitlementUseCase := usecase.NewEntitlementUseCase(subscriptionRepo, planRepo, entitlementRepo, auditUseCase)
	usageTrackingUseCase := usecase.NewUsageTrackingUseCase(usageRepo, entitlementUseCase, auditUseCase)
	_ = usageTrackingUseCase

	driverUseCase := usecase.NewDriverUseCase(driverRepo, entitlementUseCase, auditUseCase)
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
	rateLimiter := middleware.NewRateLimiter(100.0, 100.0)

	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, vehicleHandler, authMiddleware, rateLimiter.Limit, entitlementUseCase, log)

	// 1. Register a test user (creates default organization)
	regPayload := `{"name":"Billing Operator","email":"billing@btech.com","password":"SecurePassword123!","role":"admin"}`
	reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	router.ServeHTTP(wReg, reqReg)
	if wReg.Code != http.StatusCreated {
		t.Fatalf("failed to register user: %d. Body: %s", wReg.Code, wReg.Body.String())
	}

	// Login to get access token and organization details
	loginPayload := `{"email":"billing@btech.com","password":"SecurePassword123!"}`
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)
	if wLogin.Code != http.StatusOK {
		t.Fatalf("failed to login: %s", wLogin.Body.String())
	}

	var envelope apiResponseEnvelope
	_ = json.Unmarshal(wLogin.Body.Bytes(), &envelope)
	var authResp dto.AuthResponse
	_ = json.Unmarshal(envelope.Data, &authResp)

	token := authResp.Token
	orgID := authResp.User.OrganizationID

	// Resolve active subscription initially to trigger auto-provisioning of Free plan subscription
	_, _, err = billingUseCase.ResolveActiveSubscription(context.Background(), orgID)
	if err != nil {
		t.Fatalf("failed to resolve active subscription: %v", err)
	}

	// 2. Access /reports/advanced under the default Free plan (no entitlement)
	t.Run("Gated_Endpoint_Forbidden_Under_Free_Plan", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/advanced", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 3. Apply custom override for advanced reports entitlement
	t.Run("Override_Allows_Access", func(t *testing.T) {
		override := &domain.OrganizationEntitlementOverride{
			ID:             "override-test-id",
			OrganizationID: orgID,
			Key:            domain.EntitlementFeatureAdvancedReports,
			Value:          "true",
			CreatedAt:      time.Now(),
		}
		if err := entitlementRepo.SetOverride(context.Background(), override); err != nil {
			t.Fatalf("failed to set override: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/advanced", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 4. Test driver quota limits under Free plan (limit = 5 drivers)
	t.Run("Quota_Enforcement_Drivers_Max", func(t *testing.T) {
		// Seed 3 drivers to simulate the original memory driver repository seeding behavior
		for i := 1; i <= 3; i++ {
			_, err := driverRepo.Create(context.Background(), orgID, domain.Driver{
				Name:          "Seeded Driver",
				LicenseExpiry: "2030-12-31",
				Status:        "active",
			})
			if err != nil {
				t.Fatalf("failed to seed mock driver: %v", err)
			}
		}

		// So creating 2 drivers will succeed (bringing total to 5).
		for i := 1; i <= 2; i++ {
			driverJSON := `{"name":"Driver Test","licenseExpiry":"2030-12-31"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(driverJSON))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("failed to create driver %d: %s", i, w.Body.String())
			}
		}

		// 3rd new driver (6th total) should be blocked with 402 Payment Required
		driverJSON := `{"name":"Driver Test 3","licenseExpiry":"2030-12-31"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(driverJSON))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusPaymentRequired {
			t.Errorf("expected 402 Payment Required due to quota, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 5. Upgrade Subscription to Pro plan (drivers limit = 50)
	t.Run("Subscription_Upgrade_Lifts_Quota", func(t *testing.T) {
		_, err := billingUseCase.UpdateSubscription(context.Background(), orgID, "pro")
		if err != nil {
			t.Fatalf("failed to upgrade subscription: %v", err)
		}

		// 3rd new driver (6th total) creation should now succeed
		driverJSON := `{"name":"Driver Test 3","licenseExpiry":"2030-12-31"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(driverJSON))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected 201 Created after upgrade, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}
