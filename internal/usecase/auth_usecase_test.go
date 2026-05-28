package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/platform/security"
)

type mockUserRepository struct {
	users map[string]*domain.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == normEmail {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	normEmail := strings.ToLower(strings.TrimSpace(user.Email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == normEmail {
			return domain.ErrEmailAlreadyExists
		}
	}
	m.users[user.ID] = user
	return nil
}

type mockOrganizationRepository struct {
	orgs     map[string]*domain.Organization
	orgUsers map[string]*domain.OrganizationUser
}

func newMockOrganizationRepository() *mockOrganizationRepository {
	return &mockOrganizationRepository{
		orgs:     make(map[string]*domain.Organization),
		orgUsers: make(map[string]*domain.OrganizationUser),
	}
}

func (m *mockOrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	m.orgs[org.ID] = org
	return nil
}

func (m *mockOrganizationRepository) GetByID(ctx context.Context, id string) (*domain.Organization, error) {
	org, ok := m.orgs[id]
	if !ok {
		return nil, domain.ErrOrganizationNotFound
	}
	return org, nil
}

func (m *mockOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	for _, org := range m.orgs {
		if org.Slug == slug {
			return org, nil
		}
	}
	return nil, domain.ErrOrganizationNotFound
}

func (m *mockOrganizationRepository) CreateOrganizationUser(ctx context.Context, orgUser *domain.OrganizationUser) error {
	key := orgUser.OrganizationID + ":" + orgUser.UserID
	m.orgUsers[key] = orgUser
	return nil
}

func (m *mockOrganizationRepository) GetOrganizationUser(ctx context.Context, orgID, userID string) (*domain.OrganizationUser, error) {
	key := orgID + ":" + userID
	ou, ok := m.orgUsers[key]
	if !ok {
		return nil, domain.ErrOrgUserNotFound
	}
	return ou, nil
}

type mockPermissionRepository struct {
	permissionsByRole map[string][]string
}

func newMockPermissionRepository() *mockPermissionRepository {
	m := &mockPermissionRepository{
		permissionsByRole: make(map[string][]string),
	}
	m.permissionsByRole["owner"] = []string{"drivers:create", "drivers:read", "drivers:delete", "trips:create", "trips:read", "trips:update", "trips:delete", "incidents:create", "incidents:read", "settings:manage"}
	m.permissionsByRole["admin"] = []string{"drivers:create", "drivers:read", "drivers:delete", "trips:create", "trips:read", "trips:update", "trips:delete", "incidents:create", "incidents:read", "settings:manage"}
	m.permissionsByRole["operator"] = []string{"drivers:read", "trips:create", "trips:read", "trips:update", "incidents:create", "incidents:read"}
	m.permissionsByRole["viewer"] = []string{"drivers:read", "trips:read", "incidents:read"}
	return m
}

func (m *mockPermissionRepository) GetPermissionsByRole(ctx context.Context, role string) ([]string, error) {
	perms, ok := m.permissionsByRole[role]
	if !ok {
		return []string{}, nil
	}
	return perms, nil
}

type mockAuditUseCase struct{}
func (m *mockAuditUseCase) Log(ctx context.Context, action string, entityType string, entityID *string, metadata map[string]interface{}) {}
func (m *mockAuditUseCase) GetLogsByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*domain.AuditLog, error) {
	return []*domain.AuditLog{}, nil
}

func TestAuthUseCase_RegisterUser(t *testing.T) {
	repo := newMockUserRepository()
	orgRepo := newMockOrganizationRepository()
	permRepo := newMockPermissionRepository()
	auditUC := &mockAuditUseCase{}
	secret := "mysecretjwtsecretmysecretjwtsecret"
	uc := NewAuthUseCase(repo, orgRepo, permRepo, auditUC, secret, 1*time.Hour, 4) // cost = 4 for fast tests

	ctx := context.Background()

	// 1. Success case
	user, err := uc.RegisterUser(ctx, "John Doe", "JOHN@Example.com", "SecurePassword123!", "admin")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Name != "John Doe" {
		t.Errorf("expected name John Doe, got %s", user.Name)
	}

	if user.Email != "john@example.com" {
		t.Errorf("expected normalized email john@example.com, got %s", user.Email)
	}

	if user.Role != "admin" {
		t.Errorf("expected role admin, got %s", user.Role)
	}

	if user.PasswordHash == "SecurePassword123!" {
		t.Error("expected password to be hashed, but got plain text")
	}

	// 2. Duplicate email case
	_, err = uc.RegisterUser(ctx, "Other Name", "john@example.com", "SecurePassword123!", "operator")
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}

	// 3. Invalid role case
	_, err = uc.RegisterUser(ctx, "Invalid Role", "valid@example.com", "SecurePassword123!", "invalid_role")
	if !errors.Is(err, ErrInvalidRole) {
		t.Errorf("expected ErrInvalidRole, got %v", err)
	}
}

func TestAuthUseCase_LoginUser(t *testing.T) {
	repo := newMockUserRepository()
	orgRepo := newMockOrganizationRepository()
	permRepo := newMockPermissionRepository()
	auditUC := &mockAuditUseCase{}
	secret := "mysecretjwtsecretmysecretjwtsecret"
	uc := NewAuthUseCase(repo, orgRepo, permRepo, auditUC, secret, 1*time.Hour, 4)

	ctx := context.Background()

	// Register a user first
	_, err := uc.RegisterUser(ctx, "Alice Smith", "alice@example.com", "SecurePassword123!", "operator")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// 1. Success login
	user, token, err := uc.LoginUser(ctx, "ALICE@example.com", "SecurePassword123!")
	if err != nil {
		t.Fatalf("expected no error on login, got %v", err)
	}

	if user.Name != "Alice Smith" {
		t.Errorf("expected user Alice Smith, got %s", user.Name)
	}

	if token == "" {
		t.Error("expected token to be generated, got empty string")
	}

	// Verify token claims
	claims, err := security.ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("token validation failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("expected token user_id %s, got %s", user.ID, claims.UserID)
	}

	if claims.Role != "operator" {
		t.Errorf("expected token role operator, got %s", claims.Role)
	}

	// 2. Invalid credentials
	_, _, err = uc.LoginUser(ctx, "alice@example.com", "wrongpassword")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}

	// 3. User not found
	_, _, err = uc.LoginUser(ctx, "unknown@example.com", "SecurePassword123!")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthUseCase_ValidateToken(t *testing.T) {
	repo := newMockUserRepository()
	orgRepo := newMockOrganizationRepository()
	permRepo := newMockPermissionRepository()
	auditUC := &mockAuditUseCase{}
	secret := "mysecretjwtsecretmysecretjwtsecret"
	uc := NewAuthUseCase(repo, orgRepo, permRepo, auditUC, secret, 1*time.Second, 4)

	ctx := context.Background()

	user, err := uc.RegisterUser(ctx, "Bob Jones", "bob@example.com", "SecurePassword123!", "admin")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, token, err := uc.LoginUser(ctx, "bob@example.com", "SecurePassword123!")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// 1. Valid token
	validatedUser, claims, err := uc.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("expected token validation to succeed, got %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Errorf("expected validated user ID %s, got %s", user.ID, validatedUser.ID)
	}

	if claims.UserID != user.ID {
		t.Errorf("expected claims UserID %s, got %s", user.ID, claims.UserID)
	}

	// 2. Expired token
	time.Sleep(1500 * time.Millisecond)
	_, _, err = uc.ValidateToken(ctx, token)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}
