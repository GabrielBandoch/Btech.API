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

func TestAuthUseCase_RegisterUser(t *testing.T) {
	repo := newMockUserRepository()
	secret := "mysecretjwtsecretmysecretjwtsecret"
	uc := NewAuthUseCase(repo, secret, 1*time.Hour, 4) // cost = 4 for fast tests

	ctx := context.Background()

	// 1. Success case
	user, err := uc.RegisterUser(ctx, "John Doe", "JOHN@Example.com", "secure123", "manager")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Name != "John Doe" {
		t.Errorf("expected name John Doe, got %s", user.Name)
	}

	if user.Email != "john@example.com" {
		t.Errorf("expected normalized email john@example.com, got %s", user.Email)
	}

	if user.Role != "manager" {
		t.Errorf("expected role manager, got %s", user.Role)
	}

	if user.PasswordHash == "secure123" {
		t.Error("expected password to be hashed, but got plain text")
	}

	// 2. Duplicate email case
	_, err = uc.RegisterUser(ctx, "Other Name", "john@example.com", "pass123", "operator")
	if !errors.Is(err, ErrEmailExists) {
		t.Errorf("expected ErrEmailExists, got %v", err)
	}

	// 3. Invalid role case
	_, err = uc.RegisterUser(ctx, "Invalid Role", "valid@example.com", "pass123", "invalid_role")
	if !errors.Is(err, ErrInvalidRole) {
		t.Errorf("expected ErrInvalidRole, got %v", err)
	}
}

func TestAuthUseCase_LoginUser(t *testing.T) {
	repo := newMockUserRepository()
	secret := "mysecretjwtsecretmysecretjwtsecret"
	uc := NewAuthUseCase(repo, secret, 1*time.Hour, 4)

	ctx := context.Background()

	// Register a user first
	_, err := uc.RegisterUser(ctx, "Alice Smith", "alice@example.com", "password123", "operator")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// 1. Success login
	user, token, err := uc.LoginUser(ctx, "ALICE@example.com", "password123")
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
	_, _, err = uc.LoginUser(ctx, "unknown@example.com", "password123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthUseCase_ValidateToken(t *testing.T) {
	repo := newMockUserRepository()
	secret := "mysecretjwtsecretmysecretjwtsecret"
	uc := NewAuthUseCase(repo, secret, 1*time.Second, 4)

	ctx := context.Background()

	user, err := uc.RegisterUser(ctx, "Bob Jones", "bob@example.com", "password123", "admin")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	_, token, err := uc.LoginUser(ctx, "bob@example.com", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// 1. Valid token
	validatedUser, err := uc.ValidateToken(ctx, token)
	if err != nil {
		t.Fatalf("expected token validation to succeed, got %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Errorf("expected validated user ID %s, got %s", user.ID, validatedUser.ID)
	}

	// 2. Expired token
	time.Sleep(1500 * time.Millisecond)
	_, err = uc.ValidateToken(ctx, token)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}
