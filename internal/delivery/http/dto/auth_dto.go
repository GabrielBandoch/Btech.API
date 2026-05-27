package dto

import (
	"errors"
	"strings"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
func UserToResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
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
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters long")
	}

	role := strings.ToLower(strings.TrimSpace(r.Role))
	if role != "" && role != "operator" && role != "admin" && role != "manager" {
		return errors.New("role must be either operator, admin, or manager")
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
