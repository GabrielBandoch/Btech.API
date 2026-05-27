package http

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/btech/fleetcontrol-api/internal/config"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/handler"
	customMiddleware "github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
)

// NewRouter configures routes, sets up CORS, and attaches middlewares.
func NewRouter(cfg *config.Config, driverHandler *handler.DriverHandler, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.StructuredLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Duration(cfg.TimeoutSeconds) * time.Second))
	r.Use(customMiddleware.EnforceJSON)

	// CORS Config
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API Routing Group with version prefix
	r.Route("/api/"+cfg.APIVersion, func(r chi.Router) {
		r.Get("/health", handler.HealthHandler)
		r.Get("/drivers", driverHandler.GetDrivers)
	})

	return r
}
