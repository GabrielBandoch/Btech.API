package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/btech/fleetcontrol-api/internal/config"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/handler"
	customMiddleware "github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/domain"
)

// NewRouter configures routes, sets up CORS, and attaches middlewares.
func NewRouter(
	cfg *config.Config,
	driverHandler *handler.DriverHandler,
	tripHandler *handler.TripHandler,
	incidentHandler *handler.IncidentHandler,
	authHandler *handler.AuthHandler,
	authMiddleware func(http.Handler) http.Handler,
	rateLimiterMiddleware func(http.Handler) http.Handler,
	logger *slog.Logger,
) *chi.Mux {
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

		// Public Auth Routes (guarded by client IP rate limiting)
		r.Route("/auth", func(r chi.Router) {
			r.Use(rateLimiterMiddleware)
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
		})

		// Protected Route Group
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)

			// Authenticated User
			r.Get("/auth/me", authHandler.Me)

			// Drivers
			r.Get("/drivers", driverHandler.GetDrivers)
			r.Get("/drivers/{id}", driverHandler.GetDriverByID)
			r.With(customMiddleware.RequirePermission(domain.PermissionDriversCreate)).Post("/drivers", driverHandler.CreateDriver)

			// Trips
			r.Get("/trips", tripHandler.GetTrips)
			r.Get("/trips/{id}", tripHandler.GetTripByID)
			r.With(customMiddleware.RequirePermission(domain.PermissionTripsUpdate)).Put("/trips/{id}", tripHandler.UpdateTrip)

			// Incidents
			r.Get("/incidents", incidentHandler.GetIncidents)
			r.With(customMiddleware.RequirePermission(domain.PermissionIncidentsCreate)).Post("/incidents", incidentHandler.CreateIncident)
			r.With(customMiddleware.RequirePermission(domain.PermissionIncidentsCreate)).Put("/incidents/{id}", incidentHandler.UpdateIncident)
		})
	})

	return r
}
