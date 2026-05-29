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

type PostgresVehicleRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresVehicleRepository instantiates a PostgreSQL vehicle repository.
func NewPostgresVehicleRepository(pool *pgxpool.Pool) domain.VehicleRepository {
	return &PostgresVehicleRepository{
		pool: pool,
	}
}

func (r *PostgresVehicleRepository) GetAll(ctx context.Context, orgID string) ([]domain.Vehicle, error) {
	query := `SELECT id, organization_id, placa, brand, model, year, type, mileage, status, 
	                 created_at, updated_at, deleted_at 
	          FROM vehicles 
	          WHERE organization_id = $1 AND deleted_at IS NULL
	          ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var vehicles []domain.Vehicle
	for rows.Next() {
		var v domain.Vehicle
		err := rows.Scan(
			&v.ID,
			&v.OrganizationID,
			&v.Placa,
			&v.Brand,
			&v.Model,
			&v.Year,
			&v.Type,
			&v.Mileage,
			&v.Status,
			&v.CreatedAt,
			&v.UpdatedAt,
			&v.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vehicle: %w", err)
		}
		vehicles = append(vehicles, v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return vehicles, nil
}

func (r *PostgresVehicleRepository) GetByID(ctx context.Context, orgID string, id string) (domain.Vehicle, error) {
	query := `SELECT id, organization_id, placa, brand, model, year, type, mileage, status, 
	                 created_at, updated_at, deleted_at 
	          FROM vehicles 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var v domain.Vehicle
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&v.ID,
		&v.OrganizationID,
		&v.Placa,
		&v.Brand,
		&v.Model,
		&v.Year,
		&v.Type,
		&v.Mileage,
		&v.Status,
		&v.CreatedAt,
		&v.UpdatedAt,
		&v.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Vehicle{}, fmt.Errorf("vehicle not found")
		}
		return domain.Vehicle{}, fmt.Errorf("database query error: %w", err)
	}

	return v, nil
}

func (r *PostgresVehicleRepository) Create(ctx context.Context, orgID string, vehicle domain.Vehicle) (domain.Vehicle, error) {
	if vehicle.ID == "" {
		vehicle.ID = fmt.Sprintf("VH-%03d", rand.Intn(1000)+10)
	}
	vehicle.OrganizationID = orgID
	if vehicle.CreatedAt.IsZero() {
		vehicle.CreatedAt = time.Now()
	}
	vehicle.UpdatedAt = vehicle.CreatedAt

	query := `INSERT INTO vehicles (
	            id, organization_id, placa, brand, model, year, type, mileage, status, 
	            created_at, updated_at, deleted_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		vehicle.ID,
		vehicle.OrganizationID,
		vehicle.Placa,
		vehicle.Brand,
		vehicle.Model,
		vehicle.Year,
		vehicle.Type,
		vehicle.Mileage,
		vehicle.Status,
		vehicle.CreatedAt,
		vehicle.UpdatedAt,
		vehicle.DeletedAt,
	)

	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("failed to create vehicle in db: %w", err)
	}

	return vehicle, nil
}

func (r *PostgresVehicleRepository) Update(ctx context.Context, orgID string, id string, vehicle domain.Vehicle) (domain.Vehicle, error) {
	var existing domain.Vehicle
	queryCheck := `SELECT id, organization_id, placa, brand, model, year, type, mileage, status, 
	                      created_at, updated_at, deleted_at 
	               FROM vehicles 
	               WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	err := r.pool.QueryRow(ctx, queryCheck, orgID, id).Scan(
		&existing.ID,
		&existing.OrganizationID,
		&existing.Placa,
		&existing.Brand,
		&existing.Model,
		&existing.Year,
		&existing.Type,
		&existing.Mileage,
		&existing.Status,
		&existing.CreatedAt,
		&existing.UpdatedAt,
		&existing.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Vehicle{}, fmt.Errorf("vehicle not found")
		}
		return domain.Vehicle{}, fmt.Errorf("database query error: %w", err)
	}

	// Apply updates
	if vehicle.Placa != "" {
		existing.Placa = vehicle.Placa
	}
	if vehicle.Brand != "" {
		existing.Brand = vehicle.Brand
	}
	if vehicle.Model != "" {
		existing.Model = vehicle.Model
	}
	if vehicle.Year != 0 {
		existing.Year = vehicle.Year
	}
	if vehicle.Type != "" {
		existing.Type = vehicle.Type
	}
	if vehicle.Mileage != 0 {
		existing.Mileage = vehicle.Mileage
	}
	if vehicle.Status != "" {
		existing.Status = vehicle.Status
	}
	existing.UpdatedAt = time.Now()

	updateQuery := `UPDATE vehicles 
	                SET placa = $1, brand = $2, model = $3, year = $4, type = $5, mileage = $6, status = $7, updated_at = $8 
	                WHERE organization_id = $9 AND id = $10`

	_, err = r.pool.Exec(ctx, updateQuery,
		existing.Placa,
		existing.Brand,
		existing.Model,
		existing.Year,
		existing.Type,
		existing.Mileage,
		existing.Status,
		existing.UpdatedAt,
		orgID,
		id,
	)

	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("failed to update vehicle in db: %w", err)
	}

	return existing, nil
}

