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

func TestFuelIntegration(t *testing.T) {
	// Setup test environment to connect to docker container database
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/fleetcontrol?sslmode=disable")
	os.Setenv("JWT_SECRET", "supersecretchangeinproduction")
	os.Setenv("JWT_EXPIRES_IN", "15m")
	os.Setenv("BCRYPT_COST", "4")

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping fuel integration test: DATABASE_URL not set")
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

	// Truncate tables for a clean slate
	_, err = db.Pool.Exec(context.Background(), `
		TRUNCATE TABLE users, organizations, user_sessions, vehicles, drivers, fuel_records CASCADE`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	// Setup repositories
	userRepo := postgres.NewPostgresUserRepository(db.Pool)
	orgRepo := postgres.NewPostgresOrganizationRepository(db.Pool)
	permissionRepo := postgres.NewPostgresPermissionRepository(db.Pool)
	auditLogRepo := postgres.NewPostgresAuditLogRepository(db.Pool)
	sessionRepo := postgres.NewPostgresUserSessionRepository(db.Pool)
	vehicleRepo := postgres.NewPostgresVehicleRepository(db.Pool)
	driverRepo := postgres.NewPostgresDriverRepository(db.Pool)
	subscriptionRepo := postgres.NewPostgresSubscriptionRepository(db.Pool)
	planRepo := postgres.NewPostgresPlanRepository(db.Pool)
	entitlementRepo := postgres.NewPostgresEntitlementRepository(db.Pool)
	fuelRepo := postgres.NewFuelRepository(db.Pool, log)

	// Setup UseCases, Router, Middleware
	auditUseCase := usecase.NewAuditUseCase(auditLogRepo, log)
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, 15*time.Minute, 24*time.Hour, 4)
	entitlementUseCase := usecase.NewEntitlementUseCase(subscriptionRepo, planRepo, entitlementRepo, auditUseCase)
	vehicleUseCase := usecase.NewVehicleUseCase(vehicleRepo, auditUseCase)
	fuelUseCase := usecase.NewFuelUseCase(fuelRepo, vehicleRepo, driverRepo, auditUseCase, log)

	authHandler := handler.NewAuthHandler(authUseCase)
	vehicleHandler := handler.NewVehicleHandler(vehicleUseCase)
	fuelHandler := handler.NewFuelHandler(fuelUseCase)

	middleware.SetAuditUseCase(auditUseCase)
	authMiddleware := middleware.AuthMiddleware(authUseCase)
	rateLimiter := middleware.NewRateLimiter(100.0, 100.0)

	router := delivery.NewRouter(cfg, nil, nil, nil, authHandler, vehicleHandler, nil, fuelHandler, authMiddleware, rateLimiter.Limit, entitlementUseCase, log)

	// 1. Register a test admin/owner user
	regPayload := `{"name":"Fuel Manager","email":"fuel@btech.com","password":"SecurePassword123!","role":"admin"}`
	reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	router.ServeHTTP(wReg, reqReg)
	if wReg.Code != http.StatusCreated {
		t.Fatalf("failed to register user: %d. Body: %s", wReg.Code, wReg.Body.String())
	}

	// Login to get access token
	loginPayload := `{"email":"fuel@btech.com","password":"SecurePassword123!"}`
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

	// Create a test vehicle
	testVehicle := domain.Vehicle{
		ID:             "VH-FUEL-TEST",
		OrganizationID: orgID,
		Placa:          "ABC-9999",
		Brand:          "Scania",
		Model:          "R440",
		Year:           2020,
		Type:           "Truck",
		Mileage:        100000,
		Status:         "disponivel",
	}
	_, err = vehicleRepo.Create(context.Background(), orgID, testVehicle)
	if err != nil {
		t.Fatalf("failed to create test vehicle: %v", err)
	}

	var fuelRecordID string

	// 2. Test Create Fuel Record
	t.Run("Create_Fuel_Record", func(t *testing.T) {
		payload := `{"vehicleId":"VH-FUEL-TEST","date":"2026-05-29T21:30:00Z","liters":100.0,"pricePerLiter":5.50,"odometerReading":100500,"fuelType":"diesel","stationName":"Posto Ipiranga","notes":"Full tank fill-up"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/fuel/records", bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var fuelResp dto.FuelRecordResponse
		_ = json.Unmarshal(env.Data, &fuelResp)
		fuelRecordID = fuelResp.ID

		if fuelResp.Liters != 100.0 || fuelResp.TotalCost != 550.0 {
			t.Errorf("expected 100 liters and 550.0 total cost, got %f and %f", fuelResp.Liters, fuelResp.TotalCost)
		}
		if fuelResp.OdometerReading != 100500 {
			t.Errorf("expected 100500 odometer reading, got %d", fuelResp.OdometerReading)
		}
	})

	// 3. Test Get Fuel Record by ID
	t.Run("Get_Fuel_Record_By_ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/fuel/records/"+fuelRecordID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var fuelResp dto.FuelRecordResponse
		_ = json.Unmarshal(env.Data, &fuelResp)

		if fuelResp.ID != fuelRecordID {
			t.Errorf("expected fuel record ID %s, got %s", fuelRecordID, fuelResp.ID)
		}
	})

	// 4. Test List Fuel Records with filtering
	t.Run("List_Fuel_Records", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/fuel/records?vehicleId=VH-FUEL-TEST&fuelType=diesel", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var records []dto.FuelRecordResponse
		_ = json.Unmarshal(env.Data, &records)

		if len(records) != 1 {
			t.Errorf("expected 1 fuel record in list, got %d", len(records))
		}
	})

	// 5. Test Update Fuel Record (as Admin, which is always allowed)
	t.Run("Update_Fuel_Record_Admin", func(t *testing.T) {
		payload := `{"vehicleId":"VH-FUEL-TEST","date":"2026-05-29T21:35:00Z","liters":120.0,"pricePerLiter":5.50,"odometerReading":100600,"fuelType":"diesel","stationName":"Posto Petrobras"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/fuel/records/"+fuelRecordID, bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var fuelResp dto.FuelRecordResponse
		_ = json.Unmarshal(env.Data, &fuelResp)

		if fuelResp.Liters != 120.0 || fuelResp.TotalCost != 660.0 {
			t.Errorf("expected updated liters to be 120.0 and cost 660.0, got %f and %f", fuelResp.Liters, fuelResp.TotalCost)
		}
	})

	// 6. Test Dashboard Stats
	t.Run("Get_Fuel_Dashboard", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/fuel/dashboard", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	// 7. Test Delete Fuel Record
	t.Run("Delete_Fuel_Record", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/fuel/records/"+fuelRecordID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Ensure it is soft-deleted and cannot be found
		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/fuel/records/"+fuelRecordID, nil)
		reqGet.Header.Set("Authorization", "Bearer "+token)
		wGet := httptest.NewRecorder()
		router.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for deleted record, got %d", wGet.Code)
		}
	})
}
