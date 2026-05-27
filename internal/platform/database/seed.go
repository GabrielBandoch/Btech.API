package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/btech/fleetcontrol-api/internal/platform/security"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedDevelopmentDatabase checks if the database has any organizations.
// If it's empty, it inserts a demo organization and creates 4 demo users (owner, admin, operator, viewer).
func SeedDevelopmentDatabase(
	ctx context.Context,
	pool *pgxpool.Pool,
	userRepo domain.UserRepository,
	orgRepo domain.OrganizationRepository,
	bcryptCost int,
	logger *slog.Logger,
) error {
	logger.Info("Checking if development database needs seeding...")

	// Check if any organization exists
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM organizations").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count organizations: %w", err)
	}

	if count > 0 {
		logger.Info("Database already seeded. Skipping seeding.")
		return nil
	}

	logger.Info("Seeding development database with demo tenant and accounts...")

	now := time.Now()
	orgID := "btech-demo-org-id"
	org := &domain.Organization{
		ID:        orgID,
		Name:      "BTech Demo Org",
		Slug:      "btech-demo-org",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := orgRepo.Create(ctx, org); err != nil {
		return fmt.Errorf("failed to seed organization: %w", err)
	}

	demoUsers := []struct {
		Email string
		Name  string
		Role  string
	}{
		{Email: "owner@btech.com", Name: "Demo Owner", Role: "owner"},
		{Email: "admin@btech.com", Name: "Demo Admin", Role: "admin"},
		{Email: "operator@btech.com", Name: "Demo Operator", Role: "operator"},
		{Email: "viewer@btech.com", Name: "Demo Viewer", Role: "viewer"},
	}

	passwordHash, err := security.HashPassword("Password123!", bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash seed password: %w", err)
	}

	for _, du := range demoUsers {
		userID := du.Role + "-demo-id"
		user := &domain.User{
			ID:             userID,
			Name:           du.Name,
			Email:          du.Email,
			PasswordHash:   passwordHash,
			Role:           du.Role,
			OrganizationID: orgID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if err := userRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("failed to seed user %s: %w", du.Email, err)
		}

		orgUser := &domain.OrganizationUser{
			ID:             du.Role + "-mapping-id",
			OrganizationID: orgID,
			UserID:         userID,
			Role:           du.Role,
			CreatedAt:      now,
		}

		if err := orgRepo.CreateOrganizationUser(ctx, orgUser); err != nil {
			return fmt.Errorf("failed to seed organization user mapping for %s: %w", du.Email, err)
		}

		logger.Info("Successfully seeded demo user account", slog.String("email", du.Email), slog.String("role", du.Role))
	}

	logger.Info("Development database seeding completed successfully.")
	return nil
}
