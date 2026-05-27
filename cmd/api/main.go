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
	"github.com/btech/fleetcontrol-api/internal/platform/logger"
	"github.com/btech/fleetcontrol-api/internal/repository/memory"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize structured logging (slog) using platform/logger
	log := logger.New(cfg.Env)
	slog.SetDefault(log)

	log.Info("Starting BTech.API...", slog.String("env", cfg.Env), slog.String("port", cfg.Port))

	// 3. Dependency Injection
	// Repositories
	driverRepo := memory.NewMemoryDriverRepository()
	tripRepo := memory.NewMemoryTripRepository()
	incidentRepo := memory.NewMemoryIncidentRepository()

	// UseCases
	driverUseCase := usecase.NewDriverUseCase(driverRepo)
	tripUseCase := usecase.NewTripUseCase(tripRepo)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo)

	// Handlers
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)

	// 4. Setup Router
	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, log)

	// 5. Setup Server
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
		os.Exit(1)

	case sig := <-shutdownChannel:
		log.Info("Shutdown signal received, initiating graceful shutdown", slog.String("signal", sig.String()))

		// Context with timeout for shutdown phase
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("Server forced to shutdown with error", slog.String("err", err.Error()))
			if err := srv.Close(); err != nil {
				log.Error("Failed to close server", slog.String("err", err.Error()))
				os.Exit(1)
			}
		}
		log.Info("Server gracefully stopped.")
	}
}
