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

type operationResponseEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

func TestOperationalIntegration(t *testing.T) {
	// Setup test environment
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/fleetcontrol?sslmode=disable")
	os.Setenv("JWT_SECRET", "supersecretchangeinproduction")
	os.Setenv("JWT_EXPIRES_IN", "15m")
	os.Setenv("BCRYPT_COST", "4")

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping operational integration test: DATABASE_URL not set")
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
		organization_entitlement_overrides, subscription_event_history, usage_counters,
		drivers, vehicles, trips, trip_checkpoints, incidents CASCADE`)
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
	
	driverRepo := postgres.NewPostgresDriverRepository(db.Pool)
	vehicleRepo := postgres.NewPostgresVehicleRepository(db.Pool)
	tripRepo := postgres.NewPostgresTripRepository(db.Pool)
	incidentRepo := postgres.NewPostgresIncidentRepository(db.Pool)

	// Setup UseCases, Router, Middleware
	auditUseCase := usecase.NewAuditUseCase(auditLogRepo, log)
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, 15*time.Minute, 24*time.Hour, 4)
	billingUseCase := usecase.NewBillingUseCase(subscriptionRepo, planRepo, auditUseCase, log)
	entitlementUseCase := usecase.NewEntitlementUseCase(subscriptionRepo, planRepo, entitlementRepo, auditUseCase)

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

	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, vehicleHandler, nil, authMiddleware, rateLimiter.Limit, entitlementUseCase, log)

	// Register a test user (creates default organization)
	regPayload := `{"name":"Operational Manager","email":"ops@btech.com","password":"SecurePassword123!","role":"admin"}`
	reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	router.ServeHTTP(wReg, reqReg)
	if wReg.Code != http.StatusCreated {
		t.Fatalf("failed to register user: %d. Body: %s", wReg.Code, wReg.Body.String())
	}

	// Login to get access token and organization details
	loginPayload := `{"email":"ops@btech.com","password":"SecurePassword123!"}`
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)
	if wLogin.Code != http.StatusOK {
		t.Fatalf("failed to login: %s", wLogin.Body.String())
	}

	var envelope operationResponseEnvelope
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

	// 1. Create a Driver
	t.Run("Create_And_Read_Drivers", func(t *testing.T) {
		driverJSON := `{"name":"Carlos Santos","licenseExpiry":"2030-05-15","status":"active","role":"regional"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(driverJSON))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		// List drivers and verify
		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/drivers", nil)
		reqGet.Header.Set("Authorization", "Bearer "+token)
		wGet := httptest.NewRecorder()
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Fatalf("failed to list drivers: %s", wGet.Body.String())
		}

		var envList operationResponseEnvelope
		_ = json.Unmarshal(wGet.Body.Bytes(), &envList)
		var drivers []dto.DriverResponse
		_ = json.Unmarshal(envList.Data, &drivers)

		if len(drivers) != 1 {
			t.Errorf("expected 1 driver, got %d", len(drivers))
		} else {
			if drivers[0].Name != "Carlos Santos" {
				t.Errorf("expected name 'Carlos Santos', got '%s'", drivers[0].Name)
			}
			if drivers[0].Status != "active" {
				t.Errorf("expected status 'active', got '%s'", drivers[0].Status)
			}
		}
	})

	// 2. Test Quota Enforcement (Free plan limit is 5)
	t.Run("Driver_Quota_Enforcement_Free_Plan", func(t *testing.T) {
		// We already created 1 driver. Let's create 4 more to hit the limit of 5.
		for i := 2; i <= 5; i++ {
			driverJSON := `{"name":"Driver Test","licenseExpiry":"2030-12-31","status":"active"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(driverJSON))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("failed to create driver %d: %s", i, w.Body.String())
			}
		}

		// 6th driver should be blocked
		driverJSON := `{"name":"Blocked Driver","licenseExpiry":"2030-12-31","status":"active"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBufferString(driverJSON))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusPaymentRequired {
			t.Errorf("expected 402 Payment Required, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 3. Create a Vehicle
	t.Run("Create_And_Read_Vehicles", func(t *testing.T) {
		vehicleJSON := `{"brand":"Scania","model":"R 500","year":2022,"type":"Truck","mileage":150000,"status":"disponivel"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", bytes.NewBufferString(vehicleJSON))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		// List vehicles
		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles", nil)
		reqGet.Header.Set("Authorization", "Bearer "+token)
		wGet := httptest.NewRecorder()
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Fatalf("failed to list vehicles: %s", wGet.Body.String())
		}

		var envList operationResponseEnvelope
		_ = json.Unmarshal(wGet.Body.Bytes(), &envList)
		var vehicles []dto.VehicleResponse
		_ = json.Unmarshal(envList.Data, &vehicles)

		if len(vehicles) != 1 {
			t.Errorf("expected 1 vehicle, got %d", len(vehicles))
		} else {
			if vehicles[0].Brand != "Scania" || vehicles[0].Model != "R 500" {
				t.Errorf("expected Scania R 500, got %s %s", vehicles[0].Brand, vehicles[0].Model)
			}
			if vehicles[0].Year != 2022 {
				t.Errorf("expected 2022 year, got %d", vehicles[0].Year)
			}
		}
	})

	// 4. Test Soft Deletes filtering on repositories
	t.Run("Soft_Deletes_Filtering", func(t *testing.T) {
		// Active driver
		drv1, err := driverRepo.Create(context.Background(), orgID, domain.Driver{Name: "Active Driver", Status: "active"})
		if err != nil {
			t.Fatalf("failed to create active driver: %v", err)
		}

		// Deleted driver
		now := time.Now()
		_, err = driverRepo.Create(context.Background(), orgID, domain.Driver{Name: "Deleted Driver", Status: "active", DeletedAt: &now})
		if err != nil {
			t.Fatalf("failed to create deleted driver: %v", err)
		}

		// Retrieve all via repository
		list, err := driverRepo.GetAll(context.Background(), orgID)
		if err != nil {
			t.Fatalf("failed to get all drivers: %v", err)
		}

		foundActive := false
		foundDeleted := false
		for _, d := range list {
			if d.ID == drv1.ID {
				foundActive = true
			}
			if d.Name == "Deleted Driver" {
				foundDeleted = true
			}
		}

		if !foundActive {
			t.Error("expected to find active driver in list")
		}
		if foundDeleted {
			t.Error("expected NOT to find deleted driver in list due to soft delete filter")
		}
	})

	// 5. Test Role-Based Permissions for Vehicles
	t.Run("Vehicle_Permissions_Enforcement", func(t *testing.T) {
		// Register a viewer user
		regViewer := `{"name":"Operational Viewer","email":"viewer@btech.com","password":"SecurePassword123!","role":"viewer"}`
		reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regViewer))
		reqReg.Header.Set("Content-Type", "application/json")
		wReg := httptest.NewRecorder()
		router.ServeHTTP(wReg, reqReg)
		if wReg.Code != http.StatusCreated {
			t.Fatalf("failed to register viewer: %s", wReg.Body.String())
		}

		// Login viewer
		loginViewer := `{"email":"viewer@btech.com","password":"SecurePassword123!"}`
		reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginViewer))
		reqLogin.Header.Set("Content-Type", "application/json")
		wLogin := httptest.NewRecorder()
		router.ServeHTTP(wLogin, reqLogin)

		var envViewer operationResponseEnvelope
		_ = json.Unmarshal(wLogin.Body.Bytes(), &envViewer)
		var authViewer dto.AuthResponse
		_ = json.Unmarshal(envViewer.Data, &authViewer)
		viewerToken := authViewer.Token

		// Try to create a vehicle as viewer -> should be 403 Forbidden
		vehicleJSON := `{"brand":"Volvo","model":"FH","year":2021,"type":"Truck","mileage":80000,"status":"disponivel"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/vehicles", bytes.NewBufferString(vehicleJSON))
		req.Header.Set("Authorization", "Bearer "+viewerToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for viewer creating vehicle, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Try to read vehicles as viewer -> should be 200 OK
		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/vehicles", nil)
		reqGet.Header.Set("Authorization", "Bearer "+viewerToken)
		wGet := httptest.NewRecorder()
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Errorf("expected 200 OK for viewer reading vehicles, got %d. Body: %s", wGet.Code, wGet.Body.String())
		}
	})
}
