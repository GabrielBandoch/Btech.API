package usecase

import (
	"context"
	"crypto/rand"
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
	LoginUser(ctx context.Context, email, password string) (*domain.User, string, error)
	ValidateToken(ctx context.Context, token string) (*domain.User, error)
}

type authUseCase struct {
	userRepo     domain.UserRepository
	jwtSecret    string
	jwtExpiresIn time.Duration
	bcryptCost   int
}

// NewAuthUseCase instantiates a new AuthUseCase.
func NewAuthUseCase(userRepo domain.UserRepository, jwtSecret string, jwtExpiresIn time.Duration, bcryptCost int) AuthUseCase {
	return &authUseCase{
		userRepo:     userRepo,
		jwtSecret:    jwtSecret,
		jwtExpiresIn: jwtExpiresIn,
		bcryptCost:   bcryptCost,
	}
}

func (uc *authUseCase) RegisterUser(ctx context.Context, name, email, password, role string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.ToLower(strings.TrimSpace(role))

	if name == "" || email == "" || password == "" {
		return nil, errors.New("all fields are required")
	}

	if role == "" {
		role = "operator"
	}

	if role != "operator" && role != "admin" && role != "manager" {
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
	user := &domain.User{
		ID:           newUUID(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, ErrEmailExists
		}
		return nil, ErrInternal
	}

	return user, nil
}

func (uc *authUseCase) LoginUser(ctx context.Context, email, password string) (*domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" || password == "" {
		return nil, "", errors.New("email and password are required")
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", ErrInternal
	}

	if !security.ComparePassword(user.PasswordHash, password) {
		return nil, "", ErrInvalidCredentials
	}

	// Generate access token
	token, err := security.GenerateToken(user.ID, user.Role, uc.jwtSecret, uc.jwtExpiresIn)
	if err != nil {
		return nil, "", ErrInternal
	}

	return user, token, nil
}

func (uc *authUseCase) ValidateToken(ctx context.Context, tokenStr string) (*domain.User, error) {
	claims, err := security.ValidateToken(tokenStr, uc.jwtSecret)
	if err != nil {
		// Normalise security token validation errors to a clean domain error
		return nil, errors.New("unauthorized: invalid or expired token")
	}

	user, err := uc.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, errors.New("unauthorized: user not found")
		}
		return nil, ErrInternal
	}

	return user, nil
}

// newUUID generates a version 4 UUID.
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
