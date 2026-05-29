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

type PostgresMaintenanceSupplierRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMaintenanceSupplierRepository(pool *pgxpool.Pool) domain.MaintenanceSupplierRepository {
	return &PostgresMaintenanceSupplierRepository{
		pool: pool,
	}
}

func (r *PostgresMaintenanceSupplierRepository) GetAll(ctx context.Context, orgID string) ([]domain.MaintenanceSupplier, error) {
	query := `SELECT id, organization_id, name, phone, email, address, created_at, updated_at, deleted_at 
	          FROM maintenance_suppliers 
	          WHERE organization_id = $1 AND deleted_at IS NULL
	          ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var suppliers []domain.MaintenanceSupplier
	for rows.Next() {
		var s domain.MaintenanceSupplier
		err := rows.Scan(
			&s.ID,
			&s.OrganizationID,
			&s.Name,
			&s.Phone,
			&s.Email,
			&s.Address,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan maintenance supplier: %w", err)
		}
		suppliers = append(suppliers, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return suppliers, nil
}

func (r *PostgresMaintenanceSupplierRepository) GetByID(ctx context.Context, orgID string, id string) (domain.MaintenanceSupplier, error) {
	query := `SELECT id, organization_id, name, phone, email, address, created_at, updated_at, deleted_at 
	          FROM maintenance_suppliers 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var s domain.MaintenanceSupplier
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&s.ID,
		&s.OrganizationID,
		&s.Name,
		&s.Phone,
		&s.Email,
		&s.Address,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.MaintenanceSupplier{}, fmt.Errorf("supplier not found")
		}
		return domain.MaintenanceSupplier{}, fmt.Errorf("database query error: %w", err)
	}

	return s, nil
}

func (r *PostgresMaintenanceSupplierRepository) Create(ctx context.Context, orgID string, supplier domain.MaintenanceSupplier) (domain.MaintenanceSupplier, error) {
	if supplier.ID == "" {
		supplier.ID = fmt.Sprintf("SUP-%03d", rand.Intn(1000)+10)
	}
	supplier.OrganizationID = orgID
	if supplier.CreatedAt.IsZero() {
		supplier.CreatedAt = time.Now()
	}
	supplier.UpdatedAt = supplier.CreatedAt

	query := `INSERT INTO maintenance_suppliers (
	            id, organization_id, name, phone, email, address, created_at, updated_at, deleted_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		supplier.ID,
		supplier.OrganizationID,
		supplier.Name,
		supplier.Phone,
		supplier.Email,
		supplier.Address,
		supplier.CreatedAt,
		supplier.UpdatedAt,
		supplier.DeletedAt,
	)

	if err != nil {
		return domain.MaintenanceSupplier{}, fmt.Errorf("failed to create supplier in db: %w", err)
	}

	return supplier, nil
}

func (r *PostgresMaintenanceSupplierRepository) Update(ctx context.Context, orgID string, id string, supplier domain.MaintenanceSupplier) (domain.MaintenanceSupplier, error) {
	var existing domain.MaintenanceSupplier
	queryCheck := `SELECT id, organization_id, name, phone, email, address, created_at, updated_at, deleted_at 
	               FROM maintenance_suppliers 
	               WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	err := r.pool.QueryRow(ctx, queryCheck, orgID, id).Scan(
		&existing.ID,
		&existing.OrganizationID,
		&existing.Name,
		&existing.Phone,
		&existing.Email,
		&existing.Address,
		&existing.CreatedAt,
		&existing.UpdatedAt,
		&existing.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.MaintenanceSupplier{}, fmt.Errorf("supplier not found")
		}
		return domain.MaintenanceSupplier{}, fmt.Errorf("database query error: %w", err)
	}

	// Apply updates
	if supplier.Name != "" {
		existing.Name = supplier.Name
	}
	if supplier.Phone != "" {
		existing.Phone = supplier.Phone
	}
	if supplier.Email != "" {
		existing.Email = supplier.Email
	}
	if supplier.Address != "" {
		existing.Address = supplier.Address
	}
	existing.UpdatedAt = time.Now()

	updateQuery := `UPDATE maintenance_suppliers 
	                SET name = $1, phone = $2, email = $3, address = $4, updated_at = $5 
	                WHERE organization_id = $6 AND id = $7`

	_, err = r.pool.Exec(ctx, updateQuery,
		existing.Name,
		existing.Phone,
		existing.Email,
		existing.Address,
		existing.UpdatedAt,
		orgID,
		id,
	)

	if err != nil {
		return domain.MaintenanceSupplier{}, fmt.Errorf("failed to update supplier: %w", err)
	}

	return existing, nil
}

func (r *PostgresMaintenanceSupplierRepository) Delete(ctx context.Context, orgID string, id string) error {
	query := `UPDATE maintenance_suppliers SET deleted_at = $1 WHERE organization_id = $2 AND id = $3 AND deleted_at IS NULL`
	_, err := r.pool.Exec(ctx, query, time.Now(), orgID, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete supplier: %w", err)
	}
	return nil
}
