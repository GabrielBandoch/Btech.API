package postgres

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDriverRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDriverRepository instantiates a PostgreSQL driver repository.
func NewPostgresDriverRepository(pool *pgxpool.Pool) domain.DriverRepository {
	return &PostgresDriverRepository{
		pool: pool,
	}
}

func (r *PostgresDriverRepository) GetAll(ctx context.Context, orgID string) ([]domain.Driver, error) {
	query := `SELECT id, organization_id, name, avatar, status, score, trips_count, incidents_count, 
	                 next_scale, role, license_expiry, toxicology_expiry, training_expiry, 
	                 created_at, updated_at, deleted_at 
	          FROM drivers 
	          WHERE organization_id = $1 AND deleted_at IS NULL
	          ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var drivers []domain.Driver
	for rows.Next() {
		var d domain.Driver
		err := rows.Scan(
			&d.ID,
			&d.OrganizationID,
			&d.Name,
			&d.Avatar,
			&d.Status,
			&d.Score,
			&d.TripsCount,
			&d.IncidentsCount,
			&d.NextScale,
			&d.Role,
			&d.LicenseExpiry,
			&d.ToxicologyExpiry,
			&d.TrainingExpiry,
			&d.CreatedAt,
			&d.UpdatedAt,
			&d.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver: %w", err)
		}
		drivers = append(drivers, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return drivers, nil
}

func (r *PostgresDriverRepository) GetByID(ctx context.Context, orgID string, id string) (domain.Driver, error) {
	query := `SELECT id, organization_id, name, avatar, status, score, trips_count, incidents_count, 
	                 next_scale, role, license_expiry, toxicology_expiry, training_expiry, 
	                 created_at, updated_at, deleted_at 
	          FROM drivers 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var d domain.Driver
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&d.ID,
		&d.OrganizationID,
		&d.Name,
		&d.Avatar,
		&d.Status,
		&d.Score,
		&d.TripsCount,
		&d.IncidentsCount,
		&d.NextScale,
		&d.Role,
		&d.LicenseExpiry,
		&d.ToxicologyExpiry,
		&d.TrainingExpiry,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Driver{}, fmt.Errorf("driver not found")
		}
		return domain.Driver{}, fmt.Errorf("database query error: %w", err)
	}

	return d, nil
}

func (r *PostgresDriverRepository) Create(ctx context.Context, orgID string, driver domain.Driver) (domain.Driver, error) {
	if driver.ID == "" {
		driver.ID = fmt.Sprintf("DRV-%03d", rand.Intn(1000)+10)
	}
	driver.OrganizationID = orgID
	if driver.CreatedAt.IsZero() {
		driver.CreatedAt = time.Now()
	}
	driver.UpdatedAt = driver.CreatedAt

	if driver.Status == "" {
		driver.Status = domain.DriverStatusActive
	}

	query := `INSERT INTO drivers (
	            id, organization_id, name, avatar, status, score, trips_count, incidents_count, 
	            next_scale, role, license_expiry, toxicology_expiry, training_expiry, 
	            created_at, updated_at, deleted_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err := r.pool.Exec(ctx, query,
		driver.ID,
		driver.OrganizationID,
		driver.Name,
		driver.Avatar,
		driver.Status,
		driver.Score,
		driver.TripsCount,
		driver.IncidentsCount,
		driver.NextScale,
		driver.Role,
		driver.LicenseExpiry,
		driver.ToxicologyExpiry,
		driver.TrainingExpiry,
		driver.CreatedAt,
		driver.UpdatedAt,
		driver.DeletedAt,
	)

	if err != nil {
		return domain.Driver{}, fmt.Errorf("failed to create driver in db: %w", err)
	}

	return driver, nil
}

func (r *PostgresDriverRepository) Count(ctx context.Context, orgID string) (int, error) {
	query := `SELECT COUNT(*) FROM drivers WHERE organization_id = $1 AND deleted_at IS NULL`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count drivers: %w", err)
	}
	return count, nil
}
