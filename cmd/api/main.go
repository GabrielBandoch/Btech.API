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
	"github.com/btech/fleetcontrol-api/internal/repository/memory"
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

	// 4. Dependency Injection
	// Repositories
	userRepo := postgres.NewPostgresUserRepository(db.Pool)
	orgRepo := postgres.NewPostgresOrganizationRepository(db.Pool)
	driverRepo := memory.NewMemoryDriverRepository()
	tripRepo := memory.NewMemoryTripRepository()
	incidentRepo := memory.NewMemoryIncidentRepository()

	// Auto-seed development database if in development mode
	if cfg.Env == "development" {
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := database.SeedDevelopmentDatabase(seedCtx, db.Pool, userRepo, orgRepo, cfg.BCryptCost, log); err != nil {
			log.Warn("Failed to seed development database", slog.String("err", err.Error()))
		}
		seedCancel()
	}

	// UseCases
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, cfg.JWTSecret, jwtDuration, cfg.BCryptCost)
	driverUseCase := usecase.NewDriverUseCase(driverRepo)
	tripUseCase := usecase.NewTripUseCase(tripRepo)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authUseCase)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)

	// Middlewares
	authMiddleware := middleware.AuthMiddleware(authUseCase)
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRate, cfg.RateLimitBurst)

	// 5. Setup Router
	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, authMiddleware, rateLimiter.Limit, log)

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
