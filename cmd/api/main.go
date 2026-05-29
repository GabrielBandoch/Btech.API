package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/btech/fleetcontrol-api/internal/config"
	delivery "github.com/btech/fleetcontrol-api/internal/delivery/http"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/handler"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/platform/database"
	"github.com/btech/fleetcontrol-api/internal/platform/logger"
	"github.com/btech/fleetcontrol-api/internal/repository/postgres"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize structured logging (slog) using platform/logger
	log := logger.New(cfg.Env)
	slog.SetDefault(log)

	log.Info("Starting BTech.API...", slog.String("env", cfg.Env), slog.String("port", cfg.Port))

	// 3. Setup PostgreSQL Connection & Run Migrations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := database.NewPostgresDB(ctx, cfg.DatabaseURL, log)
	cancel()
	if err != nil {
		log.Error("Fatal error connecting to database", slog.String("err", err.Error()))
		os.Exit(1)
	}

	// Run goose migrations on startup
	if err := database.RunMigrations(cfg.DatabaseURL, "migrations", log); err != nil {
		log.Error("Fatal error running migrations", slog.String("err", err.Error()))
		db.Close()
		os.Exit(1)
	}

	// Parse JWT Expires duration
	jwtDuration, err := time.ParseDuration(cfg.JWTExpiresIn)
	if err != nil {
		log.Warn("Failed to parse JWT_EXPIRES_IN, defaulting to 24h", slog.String("err", err.Error()))
		jwtDuration = 24 * time.Hour
	}

	// Parse Refresh Token Expires duration
	refreshTokenDuration, err := time.ParseDuration(cfg.RefreshTokenExpiresIn)
	if err != nil {
		log.Warn("Failed to parse REFRESH_TOKEN_EXPIRES_IN, defaulting to 168h", slog.String("err", err.Error()))
		refreshTokenDuration = 7 * 24 * time.Hour
	}

	// 4. Dependency Injection
	// Repositories
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
	maintenanceSupplierRepo := postgres.NewPostgresMaintenanceSupplierRepository(db.Pool)
	maintenancePlanRepo := postgres.NewPostgresMaintenancePlanRepository(db.Pool)
	maintenanceRepo := postgres.NewPostgresMaintenanceRepository(db.Pool)
	maintenanceAlertRepo := postgres.NewPostgresMaintenanceAlertRepository(db.Pool)

	// Auto-seed development database if in development mode
	if cfg.Env == "development" {
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := database.SeedDevelopmentDatabase(seedCtx, db.Pool, userRepo, orgRepo, cfg.BCryptCost, log); err != nil {
			log.Warn("Failed to seed development database", slog.String("err", err.Error()))
		}
		seedCancel()
	}

	// UseCases
	auditUseCase := usecase.NewAuditUseCase(auditLogRepo, log)
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, jwtDuration, refreshTokenDuration, cfg.BCryptCost)
	
	// Billing UseCases
	billingUseCase := usecase.NewBillingUseCase(subscriptionRepo, planRepo, auditUseCase, log)
	_ = billingUseCase // make sure it's referenced or register it as needed
	entitlementUseCase := usecase.NewEntitlementUseCase(subscriptionRepo, planRepo, entitlementRepo, auditUseCase)
	usageTrackingUseCase := usecase.NewUsageTrackingUseCase(usageRepo, entitlementUseCase, auditUseCase)
	_ = usageTrackingUseCase // make sure it's referenced or register it as needed

	driverUseCase := usecase.NewDriverUseCase(driverRepo, entitlementUseCase, auditUseCase)
	tripUseCase := usecase.NewTripUseCase(tripRepo, auditUseCase)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo, auditUseCase)
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

	// Handlers
	authHandler := handler.NewAuthHandler(authUseCase)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)
	vehicleHandler := handler.NewVehicleHandler(vehicleUseCase)
	maintenanceHandler := handler.NewMaintenanceHandler(maintenanceUseCase)

	// Middlewares
	middleware.SetAuditUseCase(auditUseCase)
	authMiddleware := middleware.AuthMiddleware(authUseCase)
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRate, cfg.RateLimitBurst)

	// 5. Setup Router
	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, vehicleHandler, maintenanceHandler, authMiddleware, rateLimiter.Limit, entitlementUseCase, log)

	// 6. Setup Server
	serverAddr := ":" + cfg.Port
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for errors during startup
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Info("Server is listening", slog.String("addr", serverAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Graceful shutdown channel
	shutdownChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownChannel, os.Interrupt, syscall.SIGTERM)

	// Block until signal or startup error
	select {
	case err := <-serverErrors:
		log.Error("Fatal error starting server", slog.String("err", err.Error()))
		db.Close()
		os.Exit(1)

	case sig := <-shutdownChannel:
		log.Info("Shutdown signal received, initiating graceful shutdown", slog.String("signal", sig.String()))

		// Context with timeout for shutdown phase
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer shutdownCancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("Server forced to shutdown with error", slog.String("err", err.Error()))
			if err := srv.Close(); err != nil {
				log.Error("Failed to close server", slog.String("err", err.Error()))
				db.Close()
				os.Exit(1)
			}
		}

		// Graceful database pool shutdown
		log.Info("Closing database connection pool...")
		db.Close()

		log.Info("Server gracefully stopped.")
	}
}
