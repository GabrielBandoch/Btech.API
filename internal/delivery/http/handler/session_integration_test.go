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

func TestSessionIntegration(t *testing.T) {
	// Setup test environment
	os.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/fleetcontrol?sslmode=disable")
	os.Setenv("JWT_SECRET", "supersecretchangeinproduction")
	os.Setenv("JWT_EXPIRES_IN", "15m")
	os.Setenv("BCRYPT_COST", "4")

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("Skipping session integration test: DATABASE_URL not set")
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
	_, err = db.Pool.Exec(context.Background(), "TRUNCATE TABLE users, organizations, user_sessions CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	// Setup repositories
	userRepo := postgres.NewPostgresUserRepository(db.Pool)
	orgRepo := postgres.NewPostgresOrganizationRepository(db.Pool)
	permissionRepo := postgres.NewPostgresPermissionRepository(db.Pool)
	auditLogRepo := postgres.NewPostgresAuditLogRepository(db.Pool)
	sessionRepo := postgres.NewPostgresUserSessionRepository(db.Pool)
	driverRepo := postgres.NewPostgresDriverRepository(db.Pool)
	vehicleRepo := postgres.NewPostgresVehicleRepository(db.Pool)
	tripRepo := postgres.NewPostgresTripRepository(db.Pool)
	incidentRepo := postgres.NewPostgresIncidentRepository(db.Pool)

	// Setup UseCases, Router, Middleware
	auditUseCase := usecase.NewAuditUseCase(auditLogRepo, log)
	authUseCase := usecase.NewAuthUseCase(userRepo, orgRepo, permissionRepo, sessionRepo, auditUseCase, cfg.JWTSecret, 15*time.Minute, 24*time.Hour, 4)
	
	mockEntitlementUC := &mockEntitlementUseCase{}
	driverUseCase := usecase.NewDriverUseCase(driverRepo, mockEntitlementUC, auditUseCase)
	tripUseCase := usecase.NewTripUseCase(tripRepo, auditUseCase)
	incidentUseCase := usecase.NewIncidentUseCase(incidentRepo, auditUseCase)
	vehicleUseCase := usecase.NewVehicleUseCase(vehicleRepo, auditUseCase)

	authHandler := handler.NewAuthHandler(authUseCase)
	driverHandler := handler.NewDriverHandler(driverUseCase)
	tripHandler := handler.NewTripHandler(tripUseCase)
	incidentHandler := handler.NewIncidentHandler(incidentUseCase)
	vehicleHandler := handler.NewVehicleHandler(vehicleUseCase)

	middleware.SetAuditUseCase(auditUseCase)
	authMiddleware := middleware.AuthMiddleware(authUseCase)
	rateLimiter := middleware.NewRateLimiter(100.0, 100.0)

	router := delivery.NewRouter(cfg, driverHandler, tripHandler, incidentHandler, authHandler, vehicleHandler, nil, authMiddleware, rateLimiter.Limit, mockEntitlementUC, log)

	// 1. Register a test user
	regPayload := `{"name":"Session User","email":"session@btech.com","password":"SecurePassword123!","role":"admin"}`
	reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	router.ServeHTTP(wReg, reqReg)
	if wReg.Code != http.StatusCreated {
		t.Fatalf("failed to register user: %d. Body: %s", wReg.Code, wReg.Body.String())
	}

	// Helper to extract refresh token cookie from response
	getRefreshTokenCookie := func(cookies []*http.Cookie) *http.Cookie {
		for _, cookie := range cookies {
			if cookie.Name == "refresh_token" {
				return cookie
			}
		}
		return nil
	}

	var accessToken string
	var firstRefreshToken string

	// 2. Login User - Verify login sets the cookie
	t.Run("Login_Sets_Refresh_Cookie", func(t *testing.T) {
		loginPayload := `{"email":"session@btech.com","password":"SecurePassword123!"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Real-IP", "1.1.1.1")
		req.Header.Set("User-Agent", "Mozilla/5.0")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify cookies
		cookies := w.Result().Cookies()
		cookie := getRefreshTokenCookie(cookies)
		if cookie == nil {
			t.Fatal("expected refresh_token cookie to be set, but got none")
		}

		if !cookie.HttpOnly {
			t.Error("expected refresh_token cookie to be HttpOnly")
		}

		firstRefreshToken = cookie.Value

		// Decode payload
		var envelope apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &envelope)
		var authResp dto.AuthResponse
		_ = json.Unmarshal(envelope.Data, &authResp)

		if authResp.Token == "" {
			t.Error("expected access token to be returned in body")
		}
		accessToken = authResp.Token
	})

	var secondRefreshToken string

	// 3. Refresh Token - Verify rotation works
	t.Run("RefreshToken_Rotation", func(t *testing.T) {
		time.Sleep(1100 * time.Millisecond)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: firstRefreshToken})
		req.Header.Set("X-Real-IP", "1.1.1.2")
		req.Header.Set("User-Agent", "Mozilla/5.1")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify new cookie is set
		cookies := w.Result().Cookies()
		cookie := getRefreshTokenCookie(cookies)
		if cookie == nil {
			t.Fatal("expected rotated refresh_token cookie, but got none")
		}

		secondRefreshToken = cookie.Value
		if secondRefreshToken == firstRefreshToken {
			t.Error("expected refresh token value to change on rotation (RTR)")
		}

		var envelope apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &envelope)
		var authResp dto.AuthResponse
		_ = json.Unmarshal(envelope.Data, &authResp)

		if authResp.Token == "" || authResp.Token == accessToken {
			t.Error("expected new rotated access token")
		}
	})

	// 4. Replay Attack Detection - Verify using firstRefreshToken again revokes the session
	t.Run("Replay_Attack_Detection", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: firstRefreshToken}) // Already used!
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for replay attack, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Now verify that even the second (valid) refresh token is also revoked/invalidated because the session was compromised
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req2.AddCookie(&http.Cookie{Name: "refresh_token", Value: secondRefreshToken})
		w2 := httptest.NewRecorder()

		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for valid token after family invalidation, got %d", w2.Code)
		}

		// Verify that a session compromised audit event was written
		var count int
		for i := 0; i < 20; i++ {
			err = db.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM audit_logs WHERE action = $1", domain.EventSessionCompromised).Scan(&count)
			if err == nil && count > 0 {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if err != nil {
			t.Fatalf("failed to query audit logs: %v", err)
		}
		if count == 0 {
			t.Error("expected session compromised audit log to be persisted")
		}
	})

	// Let's create a new session by logging in again for subsequent tests
	var freshAccessToken string
	var freshRefreshToken string
	loginPayload := `{"email":"session@btech.com","password":"SecurePassword123!"}`
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	reqLogin.Header.Set("X-Real-IP", "1.1.1.1")
	reqLogin.Header.Set("User-Agent", "Mozilla/5.0")
	wLogin := httptest.NewRecorder()
	router.ServeHTTP(wLogin, reqLogin)
	if wLogin.Code != http.StatusOK {
		t.Fatalf("failed to login again: %s", wLogin.Body.String())
	}
	freshRefreshToken = getRefreshTokenCookie(wLogin.Result().Cookies()).Value

	var envelope apiResponseEnvelope
	_ = json.Unmarshal(wLogin.Body.Bytes(), &envelope)
	var authResp dto.AuthResponse
	_ = json.Unmarshal(envelope.Data, &authResp)
	freshAccessToken = authResp.Token

	// 5. List Sessions - Verify list sessions returns active sessions with tenant isolation
	t.Run("List_Sessions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
		req.Header.Set("Authorization", "Bearer "+freshAccessToken)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: freshRefreshToken})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var envelope apiResponseEnvelope
		_ = json.Unmarshal(w.Body.Bytes(), &envelope)
		var sessions []dto.SessionResponse
		_ = json.Unmarshal(envelope.Data, &sessions)

		// We should have at least the current active session (excluding revoked ones)
		if len(sessions) == 0 {
			t.Fatal("expected at least 1 active session in list")
		}

		foundCurrent := false
		for _, s := range sessions {
			if s.IsCurrent {
				foundCurrent = true
			}
			if s.IPAddress == "" || s.UserAgent == "" {
				t.Error("session metadata IP Address and User Agent should be visible")
			}
		}

		if !foundCurrent {
			t.Error("expected to find the current session highlighted as isCurrent = true")
		}
	})

	// 6. Revoke Session - Verify revocation terminates session
	t.Run("Revoke_Session", func(t *testing.T) {
		// List sessions to find the current session ID
		reqList := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
		reqList.Header.Set("Authorization", "Bearer "+freshAccessToken)
		reqList.AddCookie(&http.Cookie{Name: "refresh_token", Value: freshRefreshToken})
		wList := httptest.NewRecorder()
		router.ServeHTTP(wList, reqList)

		var envelope apiResponseEnvelope
		_ = json.Unmarshal(wList.Body.Bytes(), &envelope)
		var sessions []dto.SessionResponse
		_ = json.Unmarshal(envelope.Data, &sessions)

		sessionIDToRevoke := sessions[0].ID

		// Call DELETE /auth/sessions/{id}
		reqRevoke := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+sessionIDToRevoke, nil)
		reqRevoke.Header.Set("Authorization", "Bearer "+freshAccessToken)
		wRevoke := httptest.NewRecorder()
		router.ServeHTTP(wRevoke, reqRevoke)

		if wRevoke.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on revoke, got %d. Body: %s", wRevoke.Code, wRevoke.Body.String())
		}

		// Verify that using the revoked session's refresh token now returns 401
		reqRefresh := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		reqRefresh.AddCookie(&http.Cookie{Name: "refresh_token", Value: freshRefreshToken})
		wRefresh := httptest.NewRecorder()
		router.ServeHTTP(wRefresh, reqRefresh)

		if wRefresh.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for revoked session, got %d", wRefresh.Code)
		}
	})

	// Create another session to test logout
	loginPayload2 := `{"email":"session@btech.com","password":"SecurePassword123!"}`
	reqLogin2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginPayload2))
	reqLogin2.Header.Set("Content-Type", "application/json")
	reqLogin2.Header.Set("X-Real-IP", "1.1.1.1")
	reqLogin2.Header.Set("User-Agent", "Mozilla/5.0")
	wLogin2 := httptest.NewRecorder()
	router.ServeHTTP(wLogin2, reqLogin2)
	logoutRefreshToken := getRefreshTokenCookie(wLogin2.Result().Cookies()).Value

	// 7. Logout - Verify logout revokes the session and clears the cookie
	t.Run("Logout_Resilient_Flow", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: logoutRefreshToken})
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on logout, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Verify cookie is cleared (MaxAge = -1 or empty value)
		cookies := w.Result().Cookies()
		cookie := getRefreshTokenCookie(cookies)
		if cookie == nil {
			t.Fatal("expected refresh_token cookie to be returned cleared, but got none")
		}

		if cookie.Value != "" && cookie.MaxAge >= 0 {
			t.Errorf("expected cookie to be cleared, got Value: '%s', MaxAge: %d", cookie.Value, cookie.MaxAge)
		}

		// Verify that using the logout token now returns 401
		reqRefresh := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		reqRefresh.AddCookie(&http.Cookie{Name: "refresh_token", Value: logoutRefreshToken})
		wRefresh := httptest.NewRecorder()
		router.ServeHTTP(wRefresh, reqRefresh)

		if wRefresh.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized after logout, got %d", wRefresh.Code)
		}

		// Resilient logout test: logging out again with no cookie or invalid token must still return 200 OK
		reqResilient := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		wResilient := httptest.NewRecorder()
		router.ServeHTTP(wResilient, reqResilient)

		if wResilient.Code != http.StatusOK {
			t.Errorf("expected 200 OK on resilient logout with empty cookies, got %d", wResilient.Code)
		}
	})
}
