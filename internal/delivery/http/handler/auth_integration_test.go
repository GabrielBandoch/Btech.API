package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/btech/fleetcontrol-api/internal/config"
	delivery "github.com/btech/fleetcontrol-api/internal/delivery/http"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/dto"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/handler"
	"github.com/btech/fleetcontrol-api/internal/delivery/http/middleware"
	"github.com/btech/fleetcontrol-api/internal/platform/database"
	"github.com/btech/fleetcontrol-api/internal/platform/logger"
	"github.com/btech/fleetcontrol-api/internal/repository/memory"
	"github.com/btech/fleetcontrol-api/internal/repository/postgres"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

func TestAuthIntegration(t *testing.T) {
	// Programmatically set test environment variables to connect to port 5433
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/fleetcontrol?sslmode=disable")
	os.Setenv("JWT_SECRET", "supersecretchangeinproduction")
	os.Setenv("JWT_EXPIRES_IN", "2h")
	os.Setenv("BCRYPT_COST", "4")

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping integration test: DATABASE_URL not set")
	}

	log := logger.New("development")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	db, err := database.NewPostgresDB(ctx, cfg.DatabaseURL, log)
	cancel()
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Clear users table for clean test environment
	_, err = db.Pool.Exec(context.Background(), "TRUNCATE TABLE users")
	if err != nil {
		t.Fatalf("failed to truncate users table: %v", err)
	}

	// 1. Setup UseCases, Router and Middleware
	userRepo := postgres.NewPostgresUserRepository(db.Pool)
	driverRepo := memory.NewMemoryDriverRepository()
	tripRepo := memory.NewMemoryTripRepository()
	incidentRepo := memory.NewMemoryIncidentRepository()

	// Short-lived JWT expiration (2 seconds) for expiration verification
	jwtExpires := 2 * time.Second
	authUseCase := usecase.NewAuthUseCase(userRepo, cfg.JWTSecret, jwtExpires, 4) // cost = 4 for fast testing
	driverUseCase := usecase.NewDriverUseCase(driverRepo)
	tripUseCase := usecase.NewTripUseCase(tripRepo)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo)

	authHandler := handler.NewAuthHandler(authUseCase)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)

	authMiddleware := middleware.AuthMiddleware(authUseCase)

	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, authMiddleware, log)

	// Define response envelopes to match response.APIResponse
	type apiResponseEnvelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data,omitempty"`
		Error   string          `json:"error,omitempty"`
	}

	// Test 1: POST /auth/register - Success
	t.Run("Register_Success", func(t *testing.T) {
		regPayload := `{"name":"Ricardo Silva","email":"RICARDO@btech.com","password":"securepassword","role":"manager"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201 Created, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !res.Success {
			t.Errorf("expected Success to be true, got false. Error: %s", res.Error)
		}

		// Ensure password hash is not in response
		var user dto.UserResponse
		if err := json.Unmarshal(res.Data, &user); err != nil {
			t.Fatalf("failed to parse user response data: %v", err)
		}

		if user.Name != "Ricardo Silva" {
			t.Errorf("expected user name 'Ricardo Silva', got '%s'", user.Name)
		}

		// Check lowercase normalization
		if user.Email != "ricardo@btech.com" {
			t.Errorf("expected email to be lowercased to 'ricardo@btech.com', got '%s'", user.Email)
		}

		// Verify no password_hash or similar fields are serialized
		if strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), "PasswordHash") {
			t.Error("leak: response payload contains password hash references")
		}
	})

	// Test 2: POST /auth/register - Duplicate Email
	t.Run("Register_DuplicateEmail", func(t *testing.T) {
		regPayload := `{"name":"Another Ricardo","email":"ricardo@btech.com","password":"anotherpassword","role":"operator"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if res.Success {
			t.Error("expected Success to be false for duplicate email, got true")
		}

		if res.Error != "email already registered" {
			t.Errorf("expected error message 'email already registered', got '%s'", res.Error)
		}
	})

	// Test 3: POST /auth/login - Success
	var jwtToken string
	t.Run("Login_Success", func(t *testing.T) {
		loginPayload := `{"email":"RICARDO@BTECH.COM","password":"securepassword"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 OK, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !res.Success {
			t.Errorf("expected Success to be true, got false. Error: %s", res.Error)
		}

		var loginData dto.AuthResponse
		if err := json.Unmarshal(res.Data, &loginData); err != nil {
			t.Fatalf("failed to parse login response data: %v", err)
		}

		if loginData.Token == "" {
			t.Error("expected JWT token to be generated, got empty string")
		}

		jwtToken = loginData.Token

		// Ensure password hash not serialized in login response
		if strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), "PasswordHash") {
			t.Error("leak: login response payload contains password hash references")
		}
	})

	// Test 4: POST /auth/login - Invalid Password
	t.Run("Login_InvalidPassword", func(t *testing.T) {
		loginPayload := `{"email":"ricardo@btech.com","password":"wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if res.Success {
			t.Error("expected Success to be false for wrong password, got true")
		}

		if res.Error != "invalid email or password" {
			t.Errorf("expected error 'invalid email or password', got '%s'", res.Error)
		}
	})

	// Test 5: GET /auth/me - Success with valid token
	t.Run("GetMe_Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 OK, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !res.Success {
			t.Errorf("expected Success to be true, got false. Error: %s", res.Error)
		}

		var user dto.UserResponse
		if err := json.Unmarshal(res.Data, &user); err != nil {
			t.Fatalf("failed to parse user data: %v", err)
		}

		if user.Email != "ricardo@btech.com" {
			t.Errorf("expected email 'ricardo@btech.com', got '%s'", user.Email)
		}
	})

	// Test 6: GET /auth/me - Unauthorized with missing token
	t.Run("GetMe_MissingToken", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if res.Success {
			t.Error("expected Success to be false for missing token, got true")
		}
	})

	// Test 7: GET /auth/me - Unauthorized with invalid token format
	t.Run("GetMe_InvalidTokenFormat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "InvalidFormatToken")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	// Test 8: GET /auth/me - Unauthorized with expired token
	t.Run("GetMe_ExpiredToken", func(t *testing.T) {
		// Wait for token expiration (duration is 2 seconds, wait 2.5s)
		time.Sleep(2500 * time.Millisecond)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}

		var res apiResponseEnvelope
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if res.Success {
			t.Error("expected Success to be false for expired token, got true")
		}

		if res.Error != "invalid or expired authorization token" {
			t.Errorf("expected normalized expiration error message, got '%s'", res.Error)
		}
	})

	// Test 9: Protected endpoint (e.g. /drivers) - Reject unauthorized
	t.Run("Drivers_RejectUnauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/drivers", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})
}
