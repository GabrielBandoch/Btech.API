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
	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/platform/database"
	"github.com/btech/fleetcontrol-api/internal/platform/logger"
	"github.com/btech/fleetcontrol-api/internal/repository/memory"
	"github.com/btech/fleetcontrol-api/internal/repository/postgres"
	"github.com/btech/fleetcontrol-api/internal/usecase"
)

type apiResponseEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type mockUserRepoForLimit struct{}

func (m *mockUserRepoForLimit) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForLimit) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepoForLimit) Create(ctx context.Context, user *domain.User) error {
	return nil
}

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

	// Run migrations
	if err := database.RunMigrations(cfg.DatabaseURL, "../../../../migrations", log); err != nil {
		t.Fatalf("failed to run database migrations: %v", err)
	}

	// Clear tables for clean test environment
	_, err = db.Pool.Exec(context.Background(), "TRUNCATE TABLE users, organizations CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate users table: %v", err)
	}

	// Setup UseCases, Router and Middleware
	userRepo := postgres.NewPostgresUserRepository(db.Pool)
	orgRepo := postgres.NewPostgresOrganizationRepository(db.Pool)
	driverRepo := memory.NewMemoryDriverRepository()
	tripRepo := memory.NewMemoryTripRepository()
	incidentRepo := memory.NewMemoryIncidentRepository()

	// Short-lived JWT expiration (2 seconds) for expiration verification
	jwtExpires := 2 * time.Second
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, cfg.JWTSecret, jwtExpires, 4) // cost = 4 for fast testing
	driverUseCase := usecase.NewDriverUseCase(driverRepo)
	tripUseCase := usecase.NewTripUseCase(tripRepo)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo)

	authHandler := handler.NewAuthHandler(authUseCase)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)

	authMiddleware := middleware.AuthMiddleware(authUseCase)
	
	// Large capacity rate limiter to prevent rate limit blocks during main test suite
	rateLimiter := middleware.NewRateLimiter(100.0, 100.0)

	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, authMiddleware, rateLimiter.Limit, log)

	// Test 1: POST /auth/register - Success (Satisfies new password policy)
	t.Run("Register_Success", func(t *testing.T) {
		regPayload := `{"name":"Ricardo Silva","email":"RICARDO@btech.com","password":"SecurePassword123!","role":"admin"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
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

	// Test 2: POST /auth/register - Password Policy Rejections
	t.Run("Register_PasswordPolicyRejections", func(t *testing.T) {
		testCases := []struct {
			name        string
			password    string
			expectedErr string
		}{
			{"Too short", "Sh1!", "password must be at least 8 characters long"},
			{"No uppercase", "secure123!", "password must contain at least one uppercase letter"},
			{"No lowercase", "SECURE123!", "password must contain at least one lowercase letter"},
			{"No digit", "SecurePass!", "password must contain at least one digit"},
			{"No special", "SecurePass123", "password must contain at least one special character"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				payload := map[string]string{
					"name":     "Bob",
					"email":    "bob@example.com",
					"password": tc.password,
					"role":     "operator",
				}
				body, _ := json.Marshal(payload)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router.ServeHTTP(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status 400 Bad Request, got %d", w.Code)
				}

				var res apiResponseEnvelope
				_ = json.Unmarshal(w.Body.Bytes(), &res)
				if res.Success {
					t.Error("expected Success to be false for weak password, got true")
				}
				if res.Error != tc.expectedErr {
					t.Errorf("expected error '%s', got '%s'", tc.expectedErr, res.Error)
				}
			})
		}
	})

	// Test 3: POST /auth/register - Duplicate Email
	t.Run("Register_DuplicateEmail", func(t *testing.T) {
		regPayload := `{"name":"Another Ricardo","email":"ricardo@btech.com","password":"SecurePassword123!","role":"operator"}`
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

	// Test 4: POST /auth/login - Success
	var jwtToken string
	t.Run("Login_Success", func(t *testing.T) {
		loginPayload := `{"email":"RICARDO@BTECH.COM","password":"SecurePassword123!"}`
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
	})

	// Test 5: POST /auth/login - Invalid Password
	t.Run("Login_InvalidPassword", func(t *testing.T) {
		loginPayload := `{"email":"ricardo@btech.com","password":"wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	// Test 6: GET /auth/me - Success with valid token
	t.Run("GetMe_Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 OK, got %d", w.Code)
		}
	})

	// Test 7: GET /auth/me - Unauthorized with expired token
	t.Run("GetMe_ExpiredToken", func(t *testing.T) {
		time.Sleep(2500 * time.Millisecond)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401 Unauthorized, got %d", w.Code)
		}
	})
}

type mockOrgRepoForLimit struct{}

func (m *mockOrgRepoForLimit) Create(ctx context.Context, org *domain.Organization) error { return nil }
func (m *mockOrgRepoForLimit) GetByID(ctx context.Context, id string) (*domain.Organization, error) {
	return &domain.Organization{ID: id, Name: "Mock Org", Slug: "mock-org"}, nil
}
func (m *mockOrgRepoForLimit) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	return &domain.Organization{ID: "mock-id", Name: "Mock Org", Slug: slug}, nil
}
func (m *mockOrgRepoForLimit) CreateOrganizationUser(ctx context.Context, orgUser *domain.OrganizationUser) error {
	return nil
}
func (m *mockOrgRepoForLimit) GetOrganizationUser(ctx context.Context, orgID, userID string) (*domain.OrganizationUser, error) {
	return &domain.OrganizationUser{ID: "mock-mapping-id", OrganizationID: orgID, UserID: userID, Role: "operator"}, nil
}

func TestAuthRateLimiting(t *testing.T) {
	log := logger.New("development")
	cfg := config.Load()
	
	// Create mock dependencies to prevent DB connection panics
	userRepo := &mockUserRepoForLimit{}
	orgRepo := &mockOrgRepoForLimit{}
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, cfg.JWTSecret, 1*time.Hour, 4)
	authHandler := handler.NewAuthHandler(authUseCase)
	
	driverRepo := memory.NewMemoryDriverRepository()
	tripRepo := memory.NewMemoryTripRepository()
	incidentRepo := memory.NewMemoryIncidentRepository()
	driverUseCase := usecase.NewDriverUseCase(driverRepo)
	tripUseCase := usecase.NewTripUseCase(tripRepo)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)
	
	authMiddleware := middleware.AuthMiddleware(authUseCase)

	// Rate limiter with capacity = 2, rate = 0 (no token refills during test)
	rateLimiter := middleware.NewRateLimiter(0.0, 2.0)
	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, authMiddleware, rateLimiter.Limit, log)

	payload := `{"email":"test@example.com","password":"Password123!"}`
	
	// 1st request - should PASS rate limiter (returns 401 Unauthorized because user doesn't exist, but NOT 429)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(payload))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code == http.StatusTooManyRequests {
		t.Error("1st request should not be rate limited")
	}

	// 2nd request - should PASS rate limiter
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(payload))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code == http.StatusTooManyRequests {
		t.Error("2nd request should not be rate limited")
	}

	// 3rd request - should FAIL rate limiter and return 429 Too Many Requests
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(payload))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("expected 3rd request to be rate limited with status 429, got %d", w3.Code)
	}

	var res apiResponseEnvelope
	if err := json.Unmarshal(w3.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse rate limit response envelope: %v", err)
	}

	if res.Success {
		t.Error("expected envelope success to be false for 429, got true")
	}

	if res.Error != "too many requests - please try again later" {
		t.Errorf("expected error message 'too many requests - please try again later', got '%s'", res.Error)
	}
}
