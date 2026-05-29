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
