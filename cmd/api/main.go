package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/btech/fleetcontrol-api/internal/config"
	"github.com/btech/fleetcontrol-api/internal/handler"
	"github.com/btech/fleetcontrol-api/internal/repository"
)

func main() {
	log.Println("Starting FleetControl API...")

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize repository and handlers
	repo := repository.NewRepository()
	h := handler.NewHandler(repo)

	// Create a new ServeMux using Go 1.22+ routing features
	mux := http.NewServeMux()

	// Register routes with CORS middleware
	mux.HandleFunc("GET /api/trips", h.EnableCORS(h.GetTrips))
	mux.HandleFunc("GET /api/trips/{id}", h.EnableCORS(h.GetTripByID))
	mux.HandleFunc("PUT /api/trips/{id}", h.EnableCORS(h.UpdateTrip))
	mux.HandleFunc("GET /api/drivers", h.EnableCORS(h.GetDrivers))
	mux.HandleFunc("POST /api/drivers", h.EnableCORS(h.CreateDriver))
	mux.HandleFunc("GET /api/incidents", h.EnableCORS(h.GetIncidents))
	mux.HandleFunc("PUT /api/incidents/{id}", h.EnableCORS(h.UpdateIncident))

	// Options preflight fallback for non-matched methods or route-level options
	mux.HandleFunc("OPTIONS /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusOK)
	})

	// Setup Server
	serverAddr := ":" + cfg.Port
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to listen for errors during startup
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		log.Printf("Server listening on %s in %s mode\n", serverAddr, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Channel to listen for interrupt signals to perform graceful shutdown
	shutdownChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownChannel, os.Interrupt, syscall.SIGTERM)

	// Block until a signal or server startup error occurs
	select {
	case err := <-serverErrors:
		log.Fatalf("Fatal error starting server: %v", err)

	case sig := <-shutdownChannel:
		log.Printf("Shutdown signal received: %v. Initiating graceful shutdown...\n", sig)

		// Create a context with timeout for shutdown phase
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown with error: %v", err)
			if err := srv.Close(); err != nil {
				log.Fatalf("Failed to close server: %v", err)
			}
		}
		log.Println("Server gracefully stopped.")
	}
}
