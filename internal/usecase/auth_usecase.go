package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/platform/security"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidRole        = errors.New("invalid user role")
	ErrInternal           = errors.New("an internal error occurred")
)

type AuthUseCase interface {
	RegisterUser(ctx context.Context, name, email, password, role string) (*domain.User, error)
	LoginUser(ctx context.Context, email, password string) (*domain.User, string, string, error)
	ValidateToken(ctx context.Context, token string) (*domain.User, *security.JWTClaims, error)
	GetOrganizationByID(ctx context.Context, id string) (*domain.Organization, error)
	RefreshTokenRotation(ctx context.Context, refreshTokenStr, ip, ua string) (*domain.User, string, string, error)
	LogoutUser(ctx context.Context, refreshTokenStr string) error
	ListSessions(ctx context.Context, userID, orgID string) ([]*domain.UserSession, error)
	RevokeSession(ctx context.Context, sessionID, userID, orgID string) error
	ParseRefreshToken(ctx context.Context, tokenStr string) (*security.RefreshClaims, error)
}

type authUseCase struct {
	userRepo              domain.UserRepository
	orgRepo               domain.OrganizationRepository
	permissionRepo        domain.PermissionRepository
	sessionRepo           domain.UserSessionRepository
	auditUseCase          AuditUseCase
	jwtSecret             string
	jwtExpiresIn          time.Duration
	refreshTokenExpiresIn time.Duration
	bcryptCost            int
}

// NewAuthUseCase instantiates a new AuthUseCase.
func NewAuthUseCase(
	userRepo domain.UserRepository,
	orgRepo domain.OrganizationRepository,
	permissionRepo domain.PermissionRepository,
	sessionRepo domain.UserSessionRepository,
	auditUseCase AuditUseCase,
	jwtSecret string,
	jwtExpiresIn time.Duration,
	refreshTokenExpiresIn time.Duration,
	bcryptCost int,
) AuthUseCase {
	return &authUseCase{
		userRepo:              userRepo,
		orgRepo:               orgRepo,
		permissionRepo:        permissionRepo,
		sessionRepo:           sessionRepo,
		auditUseCase:          auditUseCase,
		jwtSecret:             jwtSecret,
		jwtExpiresIn:          jwtExpiresIn,
		refreshTokenExpiresIn: refreshTokenExpiresIn,
		bcryptCost:            bcryptCost,
	}
}

func (uc *authUseCase) GetOrganizationByID(ctx context.Context, id string) (*domain.Organization, error) {
	return uc.orgRepo.GetByID(ctx, id)
}

func (uc *authUseCase) RegisterUser(ctx context.Context, name, email, password, role string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.ToLower(strings.TrimSpace(role))

	if name == "" || email == "" || password == "" {
		return nil, errors.New("all fields are required")
	}

	if role == "" {
		role = "owner"
	}

	if role != "owner" && role != "admin" && role != "operator" && role != "viewer" {
		return nil, ErrInvalidRole
	}

	// Check if user already exists
	_, err := uc.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailExists
	} else if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, ErrInternal
	}

	// Hash password
	passwordHash, err := security.HashPassword(password, uc.bcryptCost)
	if err != nil {
		return nil, ErrInternal
	}

	now := time.Now()

	// 1. Create default organization
	orgID := newUUID()
	orgName := fmt.Sprintf("%s's Org", name)
	orgSlug := generateSlug(orgName)

	// In case of slug collisions, we check if it exists and make it unique
	existingOrg, err := uc.orgRepo.GetBySlug(ctx, orgSlug)
	if err == nil && existingOrg != nil {
		orgSlug = fmt.Sprintf("%s-%s", orgSlug, newUUID()[:8])
	}

	org := &domain.Organization{
		ID:        orgID,
		Name:      orgName,
		Slug:      orgSlug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.orgRepo.Create(ctx, org); err != nil {
		return nil, ErrInternal
	}

	// 2. Create user with organization association
	user := &domain.User{
		ID:             newUUID(),
		Name:           name,
		Email:          email,
		PasswordHash:   passwordHash,
		Role:           role,
		OrganizationID: orgID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, ErrEmailExists
		}
		return nil, ErrInternal
	}

	// 3. Create mapping in organization_users
	orgUser := &domain.OrganizationUser{
		ID:             newUUID(),
		OrganizationID: orgID,
		UserID:         user.ID,
		Role:           role,
		CreatedAt:      now,
	}

	if err := uc.orgRepo.CreateOrganizationUser(ctx, orgUser); err != nil {
		return nil, ErrInternal
	}

	perms, err := uc.permissionRepo.GetPermissionsByRole(ctx, role)
	if err != nil {
		return nil, ErrInternal
	}
	user.Permissions = perms

	// Log user registration event
	uc.auditUseCase.Log(ctx, domain.EventUserRegister, "user", &user.ID, map[string]interface{}{
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})

	return user, nil
}

func (uc *authUseCase) LoginUser(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" || password == "" {
		return nil, "", "", errors.New("email and password are required")
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", ErrInternal
	}

	if !security.ComparePassword(user.PasswordHash, password) {
		return nil, "", "", ErrInvalidCredentials
	}

	// Verify organization user mapping
	orgUser, err := uc.orgRepo.GetOrganizationUser(ctx, user.OrganizationID, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrOrgUserNotFound) {
			return nil, "", "", errors.New("unauthorized: no active organization association found")
		}
		return nil, "", "", ErrInternal
	}

	// Generate access token with organization ID and role claims
	token, err := security.GenerateToken(user.ID, user.OrganizationID, orgUser.Role, uc.jwtSecret, uc.jwtExpiresIn)
	if err != nil {
		return nil, "", "", ErrInternal
	}

	// Generate user session
	sessionID := newUUID()
	refreshToken, err := security.GenerateRefreshToken(sessionID, user.ID, user.OrganizationID, uc.jwtSecret, uc.refreshTokenExpiresIn)
	if err != nil {
		return nil, "", "", ErrInternal
	}

	tokenHash := hashString(refreshToken)
	now := time.Now()

	ip := extractStringFromContext(ctx, domain.ClientIPContextKey)
	ua := extractStringFromContext(ctx, domain.UserAgentContextKey)

	session := &domain.UserSession{
		ID:             sessionID,
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		TokenHash:      tokenHash,
		TokenVersion:   1,
		UserAgent:      ua,
		IPAddress:      ip,
		IsRevoked:      false,
		ExpiresAt:      now.Add(uc.refreshTokenExpiresIn),
		LastSeenAt:     &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, "", "", ErrInternal
	}

	// Set user active role to organization role
	user.Role = orgUser.Role

	perms, err := uc.permissionRepo.GetPermissionsByRole(ctx, orgUser.Role)
	if err != nil {
		return nil, "", "", ErrInternal
	}
	user.Permissions = perms

	// Log user login event
	uc.auditUseCase.Log(ctx, domain.EventUserLogin, "user", &user.ID, map[string]interface{}{
		"email":           user.Email,
		"organization_id": user.OrganizationID,
		"session_id":      sessionID,
	})

	return user, token, refreshToken, nil
}

func (uc *authUseCase) ValidateToken(ctx context.Context, tokenStr string) (*domain.User, *security.JWTClaims, error) {
	claims, err := security.ValidateToken(tokenStr, uc.jwtSecret)
	if err != nil {
		return nil, nil, errors.New("unauthorized: invalid or expired token")
	}

	user, err := uc.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, nil, errors.New("unauthorized: user not found")
		}
		return nil, nil, ErrInternal
	}

	// Verify organization user mapping from JWT claims
	orgUser, err := uc.orgRepo.GetOrganizationUser(ctx, claims.OrganizationID, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrOrgUserNotFound) {
			return nil, nil, errors.New("unauthorized: user does not belong to token organization")
		}
		return nil, nil, ErrInternal
	}

	// Set active role and organization ID from claims
	user.Role = orgUser.Role
	user.OrganizationID = orgUser.OrganizationID

	perms, err := uc.permissionRepo.GetPermissionsByRole(ctx, orgUser.Role)
	if err != nil {
		return nil, nil, ErrInternal
	}
	user.Permissions = perms

	return user, claims, nil
}

func (uc *authUseCase) RefreshTokenRotation(ctx context.Context, refreshTokenStr, ip, ua string) (*domain.User, string, string, error) {
	claims, err := security.ValidateRefreshToken(refreshTokenStr, uc.jwtSecret)
	if err != nil {
		return nil, "", "", errors.New("unauthorized: invalid or expired refresh token")
	}

	session, err := uc.sessionRepo.GetByID(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return nil, "", "", errors.New("unauthorized: session not found")
		}
		return nil, "", "", ErrInternal
	}

	if session.IsRevoked {
		return nil, "", "", errors.New("unauthorized: session is revoked")
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, "", "", errors.New("unauthorized: session expired")
	}

	incomingHash := hashString(refreshTokenStr)

	// Replay Attack Detection
	if session.TokenHash != incomingHash {
		// Old token used -> Compromised! Revoke entire family
		session.IsRevoked = true
		session.UpdatedAt = time.Now()
		_ = uc.sessionRepo.Update(ctx, session)

		// Log security breach
		uc.auditUseCase.Log(ctx, domain.EventSessionCompromised, "session", &session.ID, map[string]interface{}{
			"session_id":      session.ID,
			"user_id":         session.UserID,
			"organization_id": session.OrganizationID,
			"ip_address":      ip,
			"user_agent":      ua,
		})

		return nil, "", "", errors.New("unauthorized: session compromised")
	}

	// Fetch user details
	user, err := uc.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, "", "", errors.New("unauthorized: user not found")
		}
		return nil, "", "", ErrInternal
	}

	// Verify organization user mapping
	orgUser, err := uc.orgRepo.GetOrganizationUser(ctx, session.OrganizationID, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrOrgUserNotFound) {
			return nil, "", "", errors.New("unauthorized: no active organization association found")
		}
		return nil, "", "", ErrInternal
	}

	// Set active role and organization
	user.Role = orgUser.Role
	user.OrganizationID = orgUser.OrganizationID

	perms, err := uc.permissionRepo.GetPermissionsByRole(ctx, orgUser.Role)
	if err != nil {
		return nil, "", "", ErrInternal
	}
	user.Permissions = perms

	// Rotate tokens
	newAccessToken, err := security.GenerateToken(user.ID, user.OrganizationID, orgUser.Role, uc.jwtSecret, uc.jwtExpiresIn)
	if err != nil {
		return nil, "", "", ErrInternal
	}

	newRefreshToken, err := security.GenerateRefreshToken(session.ID, user.ID, user.OrganizationID, uc.jwtSecret, uc.refreshTokenExpiresIn)
	if err != nil {
		return nil, "", "", ErrInternal
	}

	now := time.Now()
	newHash := hashString(newRefreshToken)

	session.TokenHash = newHash
	session.TokenVersion = session.TokenVersion + 1
	session.ExpiresAt = now.Add(uc.refreshTokenExpiresIn)
	session.IPAddress = ip
	session.UserAgent = ua
	session.LastSeenAt = &now
	session.UpdatedAt = now

	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return nil, "", "", ErrInternal
	}

	// Log refresh event
	uc.auditUseCase.Log(ctx, domain.EventSessionRefresh, "session", &session.ID, map[string]interface{}{
		"session_id": session.ID,
		"user_id":    user.ID,
	})

	return user, newAccessToken, newRefreshToken, nil
}

func (uc *authUseCase) LogoutUser(ctx context.Context, refreshTokenStr string) error {
	if refreshTokenStr == "" {
		return nil // Resilient logout: succeed even if cookie is missing
	}

	claims, err := security.ValidateRefreshToken(refreshTokenStr, uc.jwtSecret)
	if err != nil {
		return nil // Resilient logout: succeed even if token is invalid or expired
	}

	session, err := uc.sessionRepo.GetByID(ctx, claims.SessionID)
	if err != nil {
		return nil // Resilient logout: succeed if session not found
	}

	if !session.IsRevoked {
		session.IsRevoked = true
		session.UpdatedAt = time.Now()
		_ = uc.sessionRepo.Update(ctx, session)

		// Log user logout event
		uc.auditUseCase.Log(ctx, domain.EventUserLogout, "user", &session.UserID, map[string]interface{}{
			"session_id":      session.ID,
			"organization_id": session.OrganizationID,
		})
	}

	return nil
}

func (uc *authUseCase) ListSessions(ctx context.Context, userID, orgID string) ([]*domain.UserSession, error) {
	return uc.sessionRepo.ListByUserID(ctx, userID, orgID)
}

func (uc *authUseCase) RevokeSession(ctx context.Context, sessionID, userID, orgID string) error {
	session, err := uc.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.ErrSessionNotFound
		}
		return ErrInternal
	}

	// Enforce tenant isolation and ownership
	if session.UserID != userID || session.OrganizationID != orgID {
		return errors.New("forbidden: cannot revoke session belonging to another user or tenant")
	}

	if !session.IsRevoked {
		session.IsRevoked = true
		session.UpdatedAt = time.Now()
		if err := uc.sessionRepo.Update(ctx, session); err != nil {
			return ErrInternal
		}

		// Log session revoke event
		uc.auditUseCase.Log(ctx, domain.EventSessionRevoke, "session", &session.ID, map[string]interface{}{
			"session_id":      session.ID,
			"revoked_user_id": session.UserID,
		})
	}

	return nil
}

func (uc *authUseCase) ParseRefreshToken(ctx context.Context, tokenStr string) (*security.RefreshClaims, error) {
	return security.ValidateRefreshToken(tokenStr, uc.jwtSecret)
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// newUUID generates a version 4 UUID.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	var builder strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			builder.WriteRune('-')
		}
	}
	res := builder.String()
	for strings.Contains(res, "--") {
		res = strings.ReplaceAll(res, "--", "-")
	}
	return strings.Trim(res, "-")
}

