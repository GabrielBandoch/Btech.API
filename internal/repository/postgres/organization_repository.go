package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresOrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOrganizationRepository(pool *pgxpool.Pool) domain.OrganizationRepository {
	return &PostgresOrganizationRepository{
		pool: pool,
	}
}

func (r *PostgresOrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	query := `INSERT INTO organizations (id, name, slug, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5)`

	_, err := r.pool.Exec(ctx, query,
		org.ID,
		org.Name,
		org.Slug,
		org.CreatedAt,
		org.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create organization in database: %w", err)
	}

	return nil
}

func (r *PostgresOrganizationRepository) GetByID(ctx context.Context, id string) (*domain.Organization, error) {
	query := `SELECT id, name, slug, created_at, updated_at FROM organizations WHERE id = $1`

	var org domain.Organization
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to query organization by id: %w", err)
	}

	return &org, nil
}

func (r *PostgresOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	query := `SELECT id, name, slug, created_at, updated_at FROM organizations WHERE slug = $1`

	var org domain.Organization
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to query organization by slug: %w", err)
	}

	return &org, nil
}

func (r *PostgresOrganizationRepository) CreateOrganizationUser(ctx context.Context, orgUser *domain.OrganizationUser) error {
	query := `INSERT INTO organization_users (id, organization_id, user_id, role, created_at) 
	          VALUES ($1, $2, $3, $4, $5)`

	_, err := r.pool.Exec(ctx, query,
		orgUser.ID,
		orgUser.OrganizationID,
		orgUser.UserID,
		orgUser.Role,
		orgUser.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create organization user mapping in database: %w", err)
	}

	return nil
}

func (r *PostgresOrganizationRepository) GetOrganizationUser(ctx context.Context, orgID, userID string) (*domain.OrganizationUser, error) {
	query := `SELECT id, organization_id, user_id, role, created_at 
	          FROM organization_users 
	          WHERE organization_id = $1 AND user_id = $2`

	var orgUser domain.OrganizationUser
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(
		&orgUser.ID,
		&orgUser.OrganizationID,
		&orgUser.UserID,
		&orgUser.Role,
		&orgUser.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrgUserNotFound
		}
		return nil, fmt.Errorf("failed to query organization user mapping: %w", err)
	}

	return &orgUser, nil
}
