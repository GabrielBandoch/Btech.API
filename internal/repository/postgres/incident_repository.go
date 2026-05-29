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

type PostgresIncidentRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresIncidentRepository instantiates a PostgreSQL incident repository.
func NewPostgresIncidentRepository(pool *pgxpool.Pool) domain.IncidentRepository {
	return &PostgresIncidentRepository{
		pool: pool,
	}
}

func (r *PostgresIncidentRepository) GetAll(ctx context.Context, orgID string) ([]domain.Incident, error) {
	query := `SELECT id, organization_id, trip_id, vehicle_placa, driver_name, type, severity, 
	                 description, timestamp, location, status, created_at, updated_at, deleted_at 
	          FROM incidents 
	          WHERE organization_id = $1 AND deleted_at IS NULL
	          ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var incidents []domain.Incident
	for rows.Next() {
		var i domain.Incident
		var tripID *string
		err := rows.Scan(
			&i.ID,
			&i.OrganizationID,
			&tripID,
			&i.VehiclePlaca,
			&i.DriverName,
			&i.Type,
			&i.Severity,
			&i.Description,
			&i.Timestamp,
			&i.Location,
			&i.Status,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		if tripID != nil {
			i.TripID = *tripID
		}
		incidents = append(incidents, i)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return incidents, nil
}

func (r *PostgresIncidentRepository) Create(ctx context.Context, orgID string, incident domain.Incident) (domain.Incident, error) {
	if incident.ID == "" {
		incident.ID = fmt.Sprintf("INC-%03d", rand.Intn(1000)+10)
	}
	incident.OrganizationID = orgID
	if incident.CreatedAt.IsZero() {
		incident.CreatedAt = time.Now()
	}
	incident.UpdatedAt = incident.CreatedAt

	if incident.Severity == "" {
		incident.Severity = domain.IncidentSeverityMedium
	}

	query := `INSERT INTO incidents (
	            id, organization_id, trip_id, vehicle_placa, driver_name, type, severity, 
	            description, timestamp, location, status, created_at, updated_at, deleted_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	var tripID interface{}
	if incident.TripID != "" {
		tripID = incident.TripID
	}

	_, err := r.pool.Exec(ctx, query,
		incident.ID,
		incident.OrganizationID,
		tripID,
		incident.VehiclePlaca,
		incident.DriverName,
		incident.Type,
		incident.Severity,
		incident.Description,
		incident.Timestamp,
		incident.Location,
		incident.Status,
		incident.CreatedAt,
		incident.UpdatedAt,
		incident.DeletedAt,
	)

	if err != nil {
		return domain.Incident{}, fmt.Errorf("failed to create incident: %w", err)
	}

	return incident, nil
}

func (r *PostgresIncidentRepository) Update(ctx context.Context, orgID string, id string, incident domain.Incident) (domain.Incident, error) {
	// Fetch existing incident to update fields
	var existing domain.Incident
	var tripID *string
	queryCheck := `SELECT id, organization_id, trip_id, vehicle_placa, driver_name, type, severity, 
	                      description, timestamp, location, status, created_at, updated_at, deleted_at 
	               FROM incidents 
	               WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	err := r.pool.QueryRow(ctx, queryCheck, orgID, id).Scan(
		&existing.ID,
		&existing.OrganizationID,
		&tripID,
		&existing.VehiclePlaca,
		&existing.DriverName,
		&existing.Type,
		&existing.Severity,
		&existing.Description,
		&existing.Timestamp,
		&existing.Location,
		&existing.Status,
		&existing.CreatedAt,
		&existing.UpdatedAt,
		&existing.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Incident{}, fmt.Errorf("incident not found")
		}
		return domain.Incident{}, fmt.Errorf("database query error: %w", err)
	}

	if tripID != nil {
		existing.TripID = *tripID
	}

	// Apply modifications
	if incident.Status != "" {
		existing.Status = incident.Status
	}
	if incident.Severity != "" {
		existing.Severity = incident.Severity
	}
	if incident.Description != "" {
		existing.Description = incident.Description
	}
	existing.UpdatedAt = time.Now()

	updateQuery := `UPDATE incidents 
	                SET status = $1, severity = $2, description = $3, updated_at = $4 
	                WHERE organization_id = $5 AND id = $6`

	_, err = r.pool.Exec(ctx, updateQuery,
		existing.Status,
		existing.Severity,
		existing.Description,
		existing.UpdatedAt,
		orgID,
		id,
	)

	if err != nil {
		return domain.Incident{}, fmt.Errorf("failed to update incident: %w", err)
	}

	return existing, nil
}
