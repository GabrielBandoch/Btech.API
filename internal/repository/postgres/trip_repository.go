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

type PostgresTripRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTripRepository instantiates a PostgreSQL trip repository.
func NewPostgresTripRepository(pool *pgxpool.Pool) domain.TripRepository {
	return &PostgresTripRepository{
		pool: pool,
	}
}

func (r *PostgresTripRepository) GetAll(ctx context.Context, orgID string) ([]domain.Trip, error) {
	query := `SELECT id, organization_id, origin, destination, status, driver_name, driver_avatar, 
	                 vehicle_placa, vehicle_model, cargo_type, cargo_value, cargo_weight, 
	                 temperature_required, estimated_time, speed, fuel_level, last_signal_time, 
	                 current_location, created_at, updated_at, deleted_at 
	          FROM trips 
	          WHERE organization_id = $1 AND deleted_at IS NULL
	          ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var trips []domain.Trip
	var tripIDs []string
	tripMap := make(map[string]*domain.Trip)

	for rows.Next() {
		var t domain.Trip
		err := rows.Scan(
			&t.ID,
			&t.OrganizationID,
			&t.Origin,
			&t.Destination,
			&t.Status,
			&t.DriverName,
			&t.DriverAvatar,
			&t.VehiclePlaca,
			&t.VehicleModel,
			&t.CargoType,
			&t.CargoValue,
			&t.CargoWeight,
			&t.TemperatureRequired,
			&t.EstimatedTime,
			&t.Speed,
			&t.FuelLevel,
			&t.LastSignalTime,
			&t.CurrentLocation,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trip: %w", err)
		}
		t.Checkpoints = []domain.Checkpoint{}
		trips = append(trips, t)
		tripIDs = append(tripIDs, t.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	if len(trips) == 0 {
		return trips, nil
	}

	// Map pointers to tripMap for fast lookup
	for i := range trips {
		tripMap[trips[i].ID] = &trips[i]
	}

	// Query checkpoints for these trips
	checkpointQuery := `SELECT id, trip_id, sequence, name, latitude, longitude, planned_at, arrived_at, created_at 
	                    FROM trip_checkpoints 
	                    WHERE trip_id = ANY($1) 
	                    ORDER BY trip_id, sequence ASC`

	cpRows, err := r.pool.Query(ctx, checkpointQuery, tripIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query checkpoints: %w", err)
	}
	defer cpRows.Close()

	for cpRows.Next() {
		var cp domain.Checkpoint
		err := cpRows.Scan(
			&cp.ID,
			&cp.TripID,
			&cp.Sequence,
			&cp.Name,
			&cp.Latitude,
			&cp.Longitude,
			&cp.PlannedAt,
			&cp.ArrivedAt,
			&cp.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint: %w", err)
		}
		if trip, ok := tripMap[cp.TripID]; ok {
			trip.Checkpoints = append(trip.Checkpoints, cp)
		}
	}

	if err := cpRows.Err(); err != nil {
		return nil, fmt.Errorf("checkpoint rows error: %w", err)
	}

	return trips, nil
}

func (r *PostgresTripRepository) GetByID(ctx context.Context, orgID string, id string) (domain.Trip, error) {
	query := `SELECT id, organization_id, origin, destination, status, driver_name, driver_avatar, 
	                 vehicle_placa, vehicle_model, cargo_type, cargo_value, cargo_weight, 
	                 temperature_required, estimated_time, speed, fuel_level, last_signal_time, 
	                 current_location, created_at, updated_at, deleted_at 
	          FROM trips 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var t domain.Trip
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&t.ID,
		&t.OrganizationID,
		&t.Origin,
		&t.Destination,
		&t.Status,
		&t.DriverName,
		&t.DriverAvatar,
		&t.VehiclePlaca,
		&t.VehicleModel,
		&t.CargoType,
		&t.CargoValue,
		&t.CargoWeight,
		&t.TemperatureRequired,
		&t.EstimatedTime,
		&t.Speed,
		&t.FuelLevel,
		&t.LastSignalTime,
		&t.CurrentLocation,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Trip{}, fmt.Errorf("trip not found")
		}
		return domain.Trip{}, fmt.Errorf("database query error: %w", err)
	}

	// Query checkpoints
	checkpointQuery := `SELECT id, trip_id, sequence, name, latitude, longitude, planned_at, arrived_at, created_at 
	                    FROM trip_checkpoints 
	                    WHERE trip_id = $1 
	                    ORDER BY sequence ASC`

	rows, err := r.pool.Query(ctx, checkpointQuery, t.ID)
	if err != nil {
		return domain.Trip{}, fmt.Errorf("failed to query checkpoints: %w", err)
	}
	defer rows.Close()

	t.Checkpoints = []domain.Checkpoint{}
	for rows.Next() {
		var cp domain.Checkpoint
		err := rows.Scan(
			&cp.ID,
			&cp.TripID,
			&cp.Sequence,
			&cp.Name,
			&cp.Latitude,
			&cp.Longitude,
			&cp.PlannedAt,
			&cp.ArrivedAt,
			&cp.CreatedAt,
		)
		if err != nil {
			return domain.Trip{}, fmt.Errorf("failed to scan checkpoint: %w", err)
		}
		t.Checkpoints = append(t.Checkpoints, cp)
	}

	return t, nil
}

func (r *PostgresTripRepository) Update(ctx context.Context, orgID string, id string, trip domain.Trip) (domain.Trip, error) {
	// We want to execute this inside a transaction because we might update the trip AND its checkpoints
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Trip{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch current trip first to make sure it exists
	var existing domain.Trip
	checkQuery := `SELECT id, organization_id, origin, destination, status, driver_name, driver_avatar, 
	                      vehicle_placa, vehicle_model, cargo_type, cargo_value, cargo_weight, 
	                      temperature_required, estimated_time, speed, fuel_level, last_signal_time, 
	                      current_location, created_at, updated_at, deleted_at 
	               FROM trips 
	               WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	err = tx.QueryRow(ctx, checkQuery, orgID, id).Scan(
		&existing.ID,
		&existing.OrganizationID,
		&existing.Origin,
		&existing.Destination,
		&existing.Status,
		&existing.DriverName,
		&existing.DriverAvatar,
		&existing.VehiclePlaca,
		&existing.VehicleModel,
		&existing.CargoType,
		&existing.CargoValue,
		&existing.CargoWeight,
		&existing.TemperatureRequired,
		&existing.EstimatedTime,
		&existing.Speed,
		&existing.FuelLevel,
		&existing.LastSignalTime,
		&existing.CurrentLocation,
		&existing.CreatedAt,
		&existing.UpdatedAt,
		&existing.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Trip{}, fmt.Errorf("trip not found")
		}
		return domain.Trip{}, fmt.Errorf("failed to query trip before update: %w", err)
	}

	// Apply modifications to fields if set
	if trip.Status != "" {
		existing.Status = trip.Status
	}
	if trip.EstimatedTime != "" {
		existing.EstimatedTime = trip.EstimatedTime
	}
	if trip.Speed != 0 {
		existing.Speed = trip.Speed
	}
	if trip.FuelLevel != 0 {
		existing.FuelLevel = trip.FuelLevel
	}
	if trip.CurrentLocation != "" {
		existing.CurrentLocation = trip.CurrentLocation
	}
	if trip.LastSignalTime != "" {
		existing.LastSignalTime = trip.LastSignalTime
	}
	existing.UpdatedAt = time.Now()

	updateQuery := `UPDATE trips 
	                SET status = $1, estimated_time = $2, speed = $3, fuel_level = $4, 
	                    current_location = $5, last_signal_time = $6, updated_at = $7 
	                WHERE organization_id = $8 AND id = $9`

	_, err = tx.Exec(ctx, updateQuery,
		existing.Status,
		existing.EstimatedTime,
		existing.Speed,
		existing.FuelLevel,
		existing.CurrentLocation,
		existing.LastSignalTime,
		existing.UpdatedAt,
		orgID,
		id,
	)

	if err != nil {
		return domain.Trip{}, fmt.Errorf("failed to update trip table: %w", err)
	}

	// If trip has checkpoints, update them
	if len(trip.Checkpoints) > 0 {
		// Delete existing checkpoints
		_, err = tx.Exec(ctx, `DELETE FROM trip_checkpoints WHERE trip_id = $1`, id)
		if err != nil {
			return domain.Trip{}, fmt.Errorf("failed to delete checkpoints: %w", err)
		}

		// Insert new checkpoints
		for _, cp := range trip.Checkpoints {
			if cp.ID == "" {
				cp.ID = fmt.Sprintf("CK-%s-%03d", id, cp.Sequence)
			}
			if cp.CreatedAt.IsZero() {
				cp.CreatedAt = time.Now()
			}
			insertCp := `INSERT INTO trip_checkpoints (id, trip_id, sequence, name, latitude, longitude, planned_at, arrived_at, created_at) 
			             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
			_, err = tx.Exec(ctx, insertCp,
				cp.ID,
				id,
				cp.Sequence,
				cp.Name,
				cp.Latitude,
				cp.Longitude,
				cp.PlannedAt,
				cp.ArrivedAt,
				cp.CreatedAt,
			)
			if err != nil {
				return domain.Trip{}, fmt.Errorf("failed to insert checkpoint sequence %d: %w", cp.Sequence, err)
			}
		}
		existing.Checkpoints = trip.Checkpoints
	} else {
		// Fetch existing checkpoints if not updated
		checkpointQuery := `SELECT id, trip_id, sequence, name, latitude, longitude, planned_at, arrived_at, created_at 
		                    FROM trip_checkpoints 
		                    WHERE trip_id = $1 
		                    ORDER BY sequence ASC`

		rows, err := tx.Query(ctx, checkpointQuery, id)
		if err == nil {
			existing.Checkpoints = []domain.Checkpoint{}
			for rows.Next() {
				var cp domain.Checkpoint
				err := rows.Scan(
					&cp.ID,
					&cp.TripID,
					&cp.Sequence,
					&cp.Name,
					&cp.Latitude,
					&cp.Longitude,
					&cp.PlannedAt,
					&cp.ArrivedAt,
					&cp.CreatedAt,
				)
				if err == nil {
					existing.Checkpoints = append(existing.Checkpoints, cp)
				}
			}
			rows.Close()
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return domain.Trip{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return existing, nil
}

// Extra helper method for internal seeding
func (r *PostgresTripRepository) Create(ctx context.Context, orgID string, trip domain.Trip) (domain.Trip, error) {
	if trip.ID == "" {
		trip.ID = fmt.Sprintf("TR-%03d", rand.Intn(1000)+10)
	}
	trip.OrganizationID = orgID
	if trip.CreatedAt.IsZero() {
		trip.CreatedAt = time.Now()
	}
	trip.UpdatedAt = trip.CreatedAt

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Trip{}, err
	}
	defer tx.Rollback(ctx)

	query := `INSERT INTO trips (
	            id, organization_id, origin, destination, status, driver_name, driver_avatar, 
	            vehicle_placa, vehicle_model, cargo_type, cargo_value, cargo_weight, 
	            temperature_required, estimated_time, speed, fuel_level, last_signal_time, 
	            current_location, created_at, updated_at, deleted_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`

	_, err = tx.Exec(ctx, query,
		trip.ID,
		trip.OrganizationID,
		trip.Origin,
		trip.Destination,
		trip.Status,
		trip.DriverName,
		trip.DriverAvatar,
		trip.VehiclePlaca,
		trip.VehicleModel,
		trip.CargoType,
		trip.CargoValue,
		trip.CargoWeight,
		trip.TemperatureRequired,
		trip.EstimatedTime,
		trip.Speed,
		trip.FuelLevel,
		trip.LastSignalTime,
		trip.CurrentLocation,
		trip.CreatedAt,
		trip.UpdatedAt,
		trip.DeletedAt,
	)

	if err != nil {
		return domain.Trip{}, fmt.Errorf("failed to create trip: %w", err)
	}

	for _, cp := range trip.Checkpoints {
		if cp.ID == "" {
			cp.ID = fmt.Sprintf("CK-%s-%03d", trip.ID, cp.Sequence)
		}
		if cp.CreatedAt.IsZero() {
			cp.CreatedAt = time.Now()
		}
		insertCp := `INSERT INTO trip_checkpoints (id, trip_id, sequence, name, latitude, longitude, planned_at, arrived_at, created_at) 
		             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		_, err = tx.Exec(ctx, insertCp,
			cp.ID,
			trip.ID,
			cp.Sequence,
			cp.Name,
			cp.Latitude,
			cp.Longitude,
			cp.PlannedAt,
			cp.ArrivedAt,
			cp.CreatedAt,
		)
		if err != nil {
			return domain.Trip{}, fmt.Errorf("failed to create checkpoint: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return domain.Trip{}, err
	}

	return trip, nil
}
