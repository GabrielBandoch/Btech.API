package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type UserResponse struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	Role             string    `json:"role"`
	Permissions      []string  `json:"permissions"`
	OrganizationID   string    `json:"organizationId"`
	OrganizationName string    `json:"organizationName"`
	OrganizationSlug string    `json:"organizationSlug"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

// UserToResponse maps a domain.User to a UserResponse DTO.
func UserToResponse(u *domain.User, orgName, orgSlug string) UserResponse {
	return UserResponse{
		ID:               u.ID,
		Name:             u.Name,
		Email:            u.Email,
		Role:             u.Role,
		Permissions:      u.Permissions,
		OrganizationID:   u.OrganizationID,
		OrganizationName: orgName,
		OrganizationSlug: orgSlug,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

// Validate RegisterRequest.
func (r *RegisterRequest) Validate() error {
	name := strings.TrimSpace(r.Name)
	email := strings.TrimSpace(r.Email)
	password := r.Password

	if name == "" {
		return errors.New("name is required")
	}
	if email == "" {
		return errors.New("email is required")
	}
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return errors.New("invalid email address format")
	}
	if err := validatePassword(password); err != nil {
		return err
	}

	role := strings.ToLower(strings.TrimSpace(r.Role))
	if role != "" && role != "owner" && role != "admin" && role != "operator" && role != "viewer" {
		return errors.New("role must be either owner, admin, operator, or viewer")
	}

	return nil
}

// validatePassword checks if a password complies with the strong password policy:
// - minimum 8 characters
// - at least 1 uppercase letter
// - at least 1 lowercase letter
// - at least 1 number
// - at least 1 special character
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	specialChars := "!@#$%^&*()-_=+[]{}|;:',.<>/?~`\""

	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case strings.ContainsRune(specialChars, r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

// Validate LoginRequest.
func (r *LoginRequest) Validate() error {
	email := strings.TrimSpace(r.Email)
	password := r.Password

	if email == "" {
		return errors.New("email is required")
	}
	if password == "" {
		return errors.New("password is required")
	}

	return nil
}
