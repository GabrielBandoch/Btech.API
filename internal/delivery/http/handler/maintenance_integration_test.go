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


func TestMaintenanceIntegration(t *testing.T) {
	// Setup test environment
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/fleetcontrol?sslmode=disable")
	os.Setenv("JWT_SECRET", "supersecretchangeinproduction")
	os.Setenv("JWT_EXPIRES_IN", "15m")
	os.Setenv("BCRYPT_COST", "4")

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping maintenance integration test: DATABASE_URL not set")
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
		TRUNCATE TABLE users, organizations, user_sessions, vehicles, 
		maintenance_suppliers, maintenance_plans, maintenances, maintenance_alerts CASCADE`)
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
	subscriptionRepo := postgres.NewPostgresSubscriptionRepository(db.Pool)
	planRepo := postgres.NewPostgresPlanRepository(db.Pool)
	entitlementRepo := postgres.NewPostgresEntitlementRepository(db.Pool)

	maintenanceSupplierRepo := postgres.NewPostgresMaintenanceSupplierRepository(db.Pool)
	maintenancePlanRepo := postgres.NewPostgresMaintenancePlanRepository(db.Pool)
	maintenanceRepo := postgres.NewPostgresMaintenanceRepository(db.Pool)
	maintenanceAlertRepo := postgres.NewPostgresMaintenanceAlertRepository(db.Pool)

	// Setup UseCases, Router, Middleware
	auditUseCase := usecase.NewAuditUseCase(auditLogRepo, log)
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, 15*time.Minute, 24*time.Hour, 4)
	entitlementUseCase := usecase.NewEntitlementUseCase(subscriptionRepo, planRepo, entitlementRepo, auditUseCase)

	vehicleUseCase := usecase.NewVehicleUseCase(vehicleRepo, auditUseCase)
	maintenanceUseCase := usecase.NewMaintenanceUseCase(
		maintenanceSupplierRepo,
		maintenancePlanRepo,
		maintenanceRepo,
		maintenanceAlertRepo,
		vehicleRepo,
		auditUseCase,
		log,
	)

	authHandler := handler.NewAuthHandler(authUseCase)
	vehicleHandler := handler.NewVehicleHandler(vehicleUseCase)
	maintenanceHandler := handler.NewMaintenanceHandler(maintenanceUseCase)

	middleware.SetAuditUseCase(auditUseCase)
	authMiddleware := middleware.AuthMiddleware(authUseCase)
	rateLimiter := middleware.NewRateLimiter(100.0, 100.0)

	router := delivery.NewRouter(cfg, nil, nil, nil, authHandler, vehicleHandler, maintenanceHandler, authMiddleware, rateLimiter.Limit, entitlementUseCase, log)

	// 1. Register a test user
	regPayload := `{"name":"Maint Operator","email":"maint@btech.com","password":"SecurePassword123!","role":"admin"}`
	reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	router.ServeHTTP(wReg, reqReg)
	if wReg.Code != http.StatusCreated {
		t.Fatalf("failed to register user: %d. Body: %s", wReg.Code, wReg.Body.String())
	}

	// Login to get access token
	loginPayload := `{"email":"maint@btech.com","password":"SecurePassword123!"}`
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
		ID:             "VH-TEST",
		OrganizationID: orgID,
		Placa:          "ABC-1234",
		Brand:          "Volvo",
		Model:          "FH 540",
		Year:           2023,
		Type:           "Truck",
		Mileage:        50000,
		Status:         "disponivel",
	}
	_, err = vehicleRepo.Create(context.Background(), orgID, testVehicle)
	if err != nil {
		t.Fatalf("failed to create test vehicle: %v", err)
	}

	var supplierID string
	var planID string
	var maintenanceID string

	// 2. Test Supplier CRUD
	t.Run("Supplier_CRUD", func(t *testing.T) {
		// Create
		payload := `{"name":"Mecânica Bosch","phone":"11999998888","email":"bosch@test.com","address":"Av Paulista, 100"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/suppliers", bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var supResp dto.SupplierResponse
		_ = json.Unmarshal(env.Data, &supResp)
		supplierID = supResp.ID

		if supResp.Name != "Mecânica Bosch" {
			t.Errorf("expected name Mecânica Bosch, got %s", supResp.Name)
		}

		// List
		reqList := httptest.NewRequest(http.MethodGet, "/api/v1/maintenance/suppliers", nil)
		reqList.Header.Set("Authorization", "Bearer "+token)
		wList := httptest.NewRecorder()
		router.ServeHTTP(wList, reqList)
		if wList.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", wList.Code)
		}

		// Update
		upPayload := `{"name":"Mecânica Bosch Express"}`
		reqUp := httptest.NewRequest(http.MethodPut, "/api/v1/maintenance/suppliers/"+supplierID, bytes.NewBufferString(upPayload))
		reqUp.Header.Set("Authorization", "Bearer "+token)
		reqUp.Header.Set("Content-Type", "application/json")
		wUp := httptest.NewRecorder()
		router.ServeHTTP(wUp, reqUp)
		if wUp.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", wUp.Code)
		}
	})

	// 3. Test Plan CRUD
	t.Run("Plan_CRUD", func(t *testing.T) {
		// Create plan: interval 10,000km, 6 months
		payload := `{"vehicleId":"VH-TEST","name":"Troca de Óleo","intervalKm":10000,"intervalMonths":6,"lastMaintenanceKm":50000}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/plans", bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var planResp dto.PlanResponse
		_ = json.Unmarshal(env.Data, &planResp)
		planID = planResp.ID

		if planResp.NextDueKM == nil || *planResp.NextDueKM != 60000 {
			t.Errorf("expected NextDueKM to be 60000, got %v", planResp.NextDueKM)
		}

		// Get by ID
		reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/maintenance/plans/"+planID, nil)
		reqGet.Header.Set("Authorization", "Bearer "+token)
		wGet := httptest.NewRecorder()
		router.ServeHTTP(wGet, reqGet)
		if wGet.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", wGet.Code)
		}
	})

	// 4. Test Alert Generation by Mileage
	t.Run("Alert_Generation_Mileage", func(t *testing.T) {
		// Vehicle is at 50,000km, NextDueKM is 60,000km.
		// Update vehicle mileage to 59,200km (within 1,000km of next due)
		payload := `{"mileage":59200}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/vehicles/VH-TEST", bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Re-trigger alert calculation via usecase (normally done during updates)
		err = maintenanceUseCase.CheckAlertsForVehicle(context.Background(), orgID, "VH-TEST")
		if err != nil {
			t.Fatalf("failed to check alerts: %v", err)
		}

		// Fetch active alerts
		reqAlerts := httptest.NewRequest(http.MethodGet, "/api/v1/maintenance/alerts?status=active", nil)
		reqAlerts.Header.Set("Authorization", "Bearer "+token)
		wAlerts := httptest.NewRecorder()
		router.ServeHTTP(wAlerts, reqAlerts)
		if wAlerts.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", wAlerts.Code)
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(wAlerts.Body.Bytes(), &env)
		var alerts []dto.AlertResponse
		_ = json.Unmarshal(env.Data, &alerts)

		if len(alerts) == 0 {
			t.Errorf("expected active alerts to be generated, got 0")
		} else {
			if alerts[0].Type != domain.MaintenanceAlertTypeMileage {
				t.Errorf("expected alert type mileage_due, got %s", alerts[0].Type)
			}
		}
	})

	// 5. Test Maintenance CRUD and side effects (status transitions & alert resolutions)
	t.Run("Maintenance_CRUD_And_Side_Effects", func(t *testing.T) {
		// Create a maintenance record in progress
		payload := `{"vehicleId":"VH-TEST","maintenancePlanId":"` + planID + `","supplierId":"` + supplierID + `","type":"preventive","priority":"high","status":"in_progress","odometerAtService":60000,"downtimeHours":6.5,"cost":1500.00,"description":"Revisão de óleo periódica","attachments":["https://example.com/invoice.pdf"]}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/records", bytes.NewBufferString(payload))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		var maintResp dto.MaintenanceResponse
		_ = json.Unmarshal(env.Data, &maintResp)
		maintenanceID = maintResp.ID

		// Verify vehicle status transitioned to "manutencao"
		veh, err := vehicleRepo.GetByID(context.Background(), orgID, "VH-TEST")
		if err != nil || veh.Status != "manutencao" {
			t.Errorf("expected vehicle status to be manutencao, got %s (err: %v)", veh.Status, err)
		}

		// Update to completed
		upPayload := `{"status":"completed"}`
		reqUp := httptest.NewRequest(http.MethodPut, "/api/v1/maintenance/records/"+maintenanceID, bytes.NewBufferString(upPayload))
		reqUp.Header.Set("Authorization", "Bearer "+token)
		reqUp.Header.Set("Content-Type", "application/json")
		wUp := httptest.NewRecorder()
		router.ServeHTTP(wUp, reqUp)
		if wUp.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", wUp.Code, wUp.Body.String())
		}

		// Verify vehicle status transitioned back to "disponivel" and mileage updated to 60,000
		veh2, err := vehicleRepo.GetByID(context.Background(), orgID, "VH-TEST")
		if err != nil || veh2.Status != "disponivel" || veh2.Mileage != 60000 {
			t.Errorf("expected vehicle status disponivel and mileage 60000, got %s and %d", veh2.Status, veh2.Mileage)
		}

		// Verify plan NextDueKM got re-calculated to 70,000km
		plan, err := maintenancePlanRepo.GetByID(context.Background(), orgID, planID)
		if err != nil || plan.NextDueKM == nil || *plan.NextDueKM != 70000 {
			t.Errorf("expected plan next due 70000, got %v (err: %v)", plan.NextDueKM, err)
		}

		// Verify that the active alert was resolved/closed
		activeAlerts, err := maintenanceAlertRepo.GetActiveByPlanID(context.Background(), orgID, planID)
		if err != nil || len(activeAlerts) > 0 {
			t.Errorf("expected active alerts to be resolved/empty, got %d active alerts", len(activeAlerts))
		}
	})

	// 6. Test Dashboard & Reports
	t.Run("Dashboard_And_Reports", func(t *testing.T) {
		// Dashboard
		reqDash := httptest.NewRequest(http.MethodGet, "/api/v1/maintenance/dashboard", nil)
		reqDash.Header.Set("Authorization", "Bearer "+token)
		wDash := httptest.NewRecorder()
		router.ServeHTTP(wDash, reqDash)
		if wDash.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", wDash.Code)
		}

		// Cost report
		reqRep := httptest.NewRequest(http.MethodGet, "/api/v1/maintenance/reports/costs", nil)
		reqRep.Header.Set("Authorization", "Bearer "+token)
		wRep := httptest.NewRecorder()
		router.ServeHTTP(wRep, reqRep)
		if wRep.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", wRep.Code)
		}

		var env apiResponseEnvelope
		_ = json.Unmarshal(wRep.Body.Bytes(), &env)
		var report map[string]interface{}
		_ = json.Unmarshal(env.Data, &report)

		total, ok := report["totalCost"].(float64)
		if !ok || total != 1500.00 {
			t.Errorf("expected totalCost to be 1500.00, got %v", report["totalCost"])
		}
	})
}
