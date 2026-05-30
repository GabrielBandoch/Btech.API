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

type PostgresDriverDocumentRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDriverDocumentRepository instantiates a PostgreSQL driver document repository.
func NewPostgresDriverDocumentRepository(pool *pgxpool.Pool) domain.DriverDocumentRepository {
	return &PostgresDriverDocumentRepository{
		pool: pool,
	}
}

func (r *PostgresDriverDocumentRepository) GetAll(ctx context.Context, orgID string, filter domain.DriverDocumentFilter) ([]domain.DriverDocument, error) {
	query := `SELECT id, organization_id, driver_id, type, document_number, issuing_authority, 
	                 issue_date, expiration_date, notes, status, created_at, updated_at, deleted_at 
	          FROM driver_documents 
	          WHERE organization_id = $1 AND deleted_at IS NULL`

	args := []interface{}{orgID}
	placeholderIndex := 2

	if filter.DriverID != "" {
		query += fmt.Sprintf(" AND driver_id = $%d", placeholderIndex)
		args = append(args, filter.DriverID)
		placeholderIndex++
	}
	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", placeholderIndex)
		args = append(args, filter.Type)
		placeholderIndex++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", placeholderIndex)
		args = append(args, filter.Status)
		placeholderIndex++
	}
	if filter.ExpirationStart != nil {
		query += fmt.Sprintf(" AND expiration_date >= $%d", placeholderIndex)
		args = append(args, *filter.ExpirationStart)
		placeholderIndex++
	}
	if filter.ExpirationEnd != nil {
		query += fmt.Sprintf(" AND expiration_date <= $%d", placeholderIndex)
		args = append(args, *filter.ExpirationEnd)
		placeholderIndex++
	}

	query += " ORDER BY expiration_date ASC NULLS LAST"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var docs []domain.DriverDocument
	var docIDs []string
	for rows.Next() {
		var d domain.DriverDocument
		err := rows.Scan(
			&d.ID,
			&d.OrganizationID,
			&d.DriverID,
			&d.Type,
			&d.DocumentNumber,
			&d.IssuingAuthority,
			&d.IssueDate,
			&d.ExpirationDate,
			&d.Notes,
			&d.Status,
			&d.CreatedAt,
			&d.UpdatedAt,
			&d.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver document: %w", err)
		}
		d.Files = []domain.DriverDocumentFile{} // initialize slice
		docs = append(docs, d)
		docIDs = append(docIDs, d.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// Fetch files for all fetched documents in a single batch query
	if len(docIDs) > 0 {
		filesQuery := `SELECT id, organization_id, document_id, file_path, file_name, file_size, content_type, created_at, updated_at 
		               FROM driver_document_files 
		               WHERE organization_id = $1 AND document_id = ANY($2) AND deleted_at IS NULL`
		
		filesRows, err := r.pool.Query(ctx, filesQuery, orgID, docIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve document files: %w", err)
		}
		defer filesRows.Close()

		filesMap := make(map[string][]domain.DriverDocumentFile)
		for filesRows.Next() {
			var f domain.DriverDocumentFile
			err := filesRows.Scan(
				&f.ID,
				&f.OrganizationID,
				&f.DocumentID,
				&f.FilePath,
				&f.FileName,
				&f.FileSize,
				&f.ContentType,
				&f.CreatedAt,
				&f.UpdatedAt,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan file: %w", err)
			}
			filesMap[f.DocumentID] = append(filesMap[f.DocumentID], f)
		}

		// Map files back to corresponding documents
		for i, doc := range docs {
			if files, ok := filesMap[doc.ID]; ok {
				docs[i].Files = files
			}
		}
	}

	return docs, nil
}

func (r *PostgresDriverDocumentRepository) GetByID(ctx context.Context, orgID string, id string) (domain.DriverDocument, error) {
	query := `SELECT id, organization_id, driver_id, type, document_number, issuing_authority, 
	                 issue_date, expiration_date, notes, status, created_at, updated_at, deleted_at 
	          FROM driver_documents 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var d domain.DriverDocument
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&d.ID,
		&d.OrganizationID,
		&d.DriverID,
		&d.Type,
		&d.DocumentNumber,
		&d.IssuingAuthority,
		&d.IssueDate,
		&d.ExpirationDate,
		&d.Notes,
		&d.Status,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DriverDocument{}, domain.ErrDriverDocumentNotFound
		}
		return domain.DriverDocument{}, fmt.Errorf("database query error: %w", err)
	}

	// Fetch files
	filesQuery := `SELECT id, organization_id, document_id, file_path, file_name, file_size, content_type, created_at, updated_at 
	               FROM driver_document_files 
	               WHERE organization_id = $1 AND document_id = $2 AND deleted_at IS NULL`

	rows, err := r.pool.Query(ctx, filesQuery, orgID, id)
	if err != nil {
		return domain.DriverDocument{}, fmt.Errorf("failed to query document files: %w", err)
	}
	defer rows.Close()

	d.Files = []domain.DriverDocumentFile{}
	for rows.Next() {
		var f domain.DriverDocumentFile
		err := rows.Scan(
			&f.ID,
			&f.OrganizationID,
			&f.DocumentID,
			&f.FilePath,
			&f.FileName,
			&f.FileSize,
			&f.ContentType,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return domain.DriverDocument{}, fmt.Errorf("failed to scan file: %w", err)
		}
		d.Files = append(d.Files, f)
	}

	return d, nil
}

func (r *PostgresDriverDocumentRepository) Create(ctx context.Context, orgID string, doc domain.DriverDocument) (domain.DriverDocument, error) {
	if doc.ID == "" {
		doc.ID = fmt.Sprintf("DOC-%04d", rand.Intn(9000)+1000)
	}
	doc.OrganizationID = orgID
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	doc.UpdatedAt = doc.CreatedAt

	query := `INSERT INTO driver_documents (
	            id, organization_id, driver_id, type, document_number, issuing_authority, 
	            issue_date, expiration_date, notes, status, created_at, updated_at
	          ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.pool.Exec(ctx, query,
		doc.ID,
		doc.OrganizationID,
		doc.DriverID,
		doc.Type,
		doc.DocumentNumber,
		doc.IssuingAuthority,
		doc.IssueDate,
		doc.ExpirationDate,
		doc.Notes,
		doc.Status,
		doc.CreatedAt,
		doc.UpdatedAt,
	)

	if err != nil {
		return domain.DriverDocument{}, fmt.Errorf("failed to insert driver document: %w", err)
	}

	doc.Files = []domain.DriverDocumentFile{}
	return doc, nil
}

func (r *PostgresDriverDocumentRepository) Update(ctx context.Context, orgID string, id string, doc domain.DriverDocument) (domain.DriverDocument, error) {
	existing, err := r.GetByID(ctx, orgID, id)
	if err != nil {
		return domain.DriverDocument{}, err
	}

	if doc.DocumentNumber != "" {
		existing.DocumentNumber = doc.DocumentNumber
	}
	if doc.IssuingAuthority != "" {
		existing.IssuingAuthority = doc.IssuingAuthority
	}
	if doc.IssueDate != nil {
		existing.IssueDate = doc.IssueDate
	}
	if doc.ExpirationDate != nil {
		existing.ExpirationDate = doc.ExpirationDate
	}
	if doc.Notes != "" {
		existing.Notes = doc.Notes
	}
	if doc.Status != "" {
		existing.Status = doc.Status
	}
	existing.UpdatedAt = time.Now()

	query := `UPDATE driver_documents 
	          SET document_number = $1, issuing_authority = $2, issue_date = $3, expiration_date = $4, notes = $5, status = $6, updated_at = $7 
	          WHERE organization_id = $8 AND id = $9 AND deleted_at IS NULL`

	_, err = r.pool.Exec(ctx, query,
		existing.DocumentNumber,
		existing.IssuingAuthority,
		existing.IssueDate,
		existing.ExpirationDate,
		existing.Notes,
		existing.Status,
		existing.UpdatedAt,
		orgID,
		id,
	)

	if err != nil {
		return domain.DriverDocument{}, fmt.Errorf("failed to update driver document: %w", err)
	}

	return existing, nil
}

func (r *PostgresDriverDocumentRepository) Delete(ctx context.Context, orgID string, id string) error {
	now := time.Now()
	
	// Soft delete document
	queryDoc := `UPDATE driver_documents 
	             SET deleted_at = $1, updated_at = $1 
	             WHERE organization_id = $2 AND id = $3 AND deleted_at IS NULL`
	
	cmd, err := r.pool.Exec(ctx, queryDoc, now, orgID, id)
	if err != nil {
		return fmt.Errorf("failed to delete driver document: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrDriverDocumentNotFound
	}

	// Soft delete associated files
	queryFiles := `UPDATE driver_document_files 
	               SET deleted_at = $1, updated_at = $1 
	               WHERE organization_id = $2 AND document_id = $3 AND deleted_at IS NULL`
	_, _ = r.pool.Exec(ctx, queryFiles, now, orgID, id)

	return nil
}

func (r *PostgresDriverDocumentRepository) GetDashboardStats(ctx context.Context, orgID string, mandatoryTypes []string) (domain.DocumentDashboard, error) {
	var dash domain.DocumentDashboard

	// 1. Total document records count
	countQuery := `SELECT COUNT(*) FROM driver_documents WHERE organization_id = $1 AND deleted_at IS NULL`
	err := r.pool.QueryRow(ctx, countQuery, orgID).Scan(&dash.TotalDocuments)
	if err != nil {
		return dash, fmt.Errorf("failed to query total documents: %w", err)
	}

	// 2. Status breakups
	statusQuery := `
		SELECT 
			COALESCE(SUM(CASE WHEN status = 'valid' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'expiring_soon' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'expired' THEN 1 ELSE 0 END), 0)
		FROM driver_documents 
		WHERE organization_id = $1 AND deleted_at IS NULL`

	err = r.pool.QueryRow(ctx, statusQuery, orgID).Scan(&dash.ValidDocuments, &dash.ExpiringDocuments, &dash.ExpiredDocuments)
	if err != nil {
		return dash, fmt.Errorf("failed to query document statuses: %w", err)
	}

	// 3. Driver compliance calculations (Configurable Rules)
	var totalDrivers int
	totalDriversQuery := `SELECT COUNT(*) FROM drivers WHERE organization_id = $1 AND deleted_at IS NULL`
	err = r.pool.QueryRow(ctx, totalDriversQuery, orgID).Scan(&totalDrivers)
	if err != nil {
		return dash, fmt.Errorf("failed to query total drivers: %w", err)
	}

	if totalDrivers == 0 {
		dash.CompliancePercentage = 100.0
		return dash, nil
	}

	// Drivers with missing or expired mandatory documents
	var nonCompliantCount int
	nonCompliantQuery := `
		SELECT COUNT(DISTINCT d.id)
		FROM drivers d
		CROSS JOIN (
			SELECT unnest($2::text[]) as req_type
		) r
		LEFT JOIN (
			SELECT DISTINCT ON (driver_id, type) id, driver_id, type, status, expiration_date
			FROM driver_documents
			WHERE organization_id = $1 AND deleted_at IS NULL
			ORDER BY driver_id, type, expiration_date DESC NULLS LAST
		) doc ON doc.driver_id = d.id AND doc.type = r.req_type
		WHERE d.organization_id = $1 AND d.deleted_at IS NULL
		  AND (doc.id IS NULL OR doc.status = 'expired')`

	err = r.pool.QueryRow(ctx, nonCompliantQuery, orgID, mandatoryTypes).Scan(&nonCompliantCount)
	if err != nil {
		return dash, fmt.Errorf("failed to calculate non-compliant drivers: %w", err)
	}

	compliantCount := totalDrivers - nonCompliantCount
	dash.CompliancePercentage = (float64(compliantCount) / float64(totalDrivers)) * 100.0

	return dash, nil
}

func (r *PostgresDriverDocumentRepository) CreateAlert(ctx context.Context, orgID string, alert domain.DriverDocumentAlert) (domain.DriverDocumentAlert, error) {
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("ALT-%04d", rand.Intn(9000)+1000)
	}
	alert.OrganizationID = orgID
	alert.CreatedAt = time.Now()
	alert.Status = domain.AlertStatusActive

	query := `INSERT INTO driver_document_alerts (id, organization_id, document_id, days_remaining, status, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, query,
		alert.ID,
		alert.OrganizationID,
		alert.DocumentID,
		alert.DaysRemaining,
		alert.Status,
		alert.CreatedAt,
	)

	if err != nil {
		return domain.DriverDocumentAlert{}, fmt.Errorf("failed to insert document alert: %w", err)
	}

	return alert, nil
}

func (r *PostgresDriverDocumentRepository) GetActiveAlerts(ctx context.Context, orgID string) ([]domain.DriverDocumentAlert, error) {
	query := `SELECT id, organization_id, document_id, days_remaining, status, created_at, resolved_at 
	          FROM driver_document_alerts 
	          WHERE organization_id = $1 AND status = $2`

	rows, err := r.pool.Query(ctx, query, orgID, domain.AlertStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve active alerts: %w", err)
	}
	defer rows.Close()

	var alerts []domain.DriverDocumentAlert
	for rows.Next() {
		var a domain.DriverDocumentAlert
		err := rows.Scan(
			&a.ID,
			&a.OrganizationID,
			&a.DocumentID,
			&a.DaysRemaining,
			&a.Status,
			&a.CreatedAt,
			&a.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}

	return alerts, nil
}

func (r *PostgresDriverDocumentRepository) ResolveAlertsForDocument(ctx context.Context, orgID string, docID string) error {
	query := `UPDATE driver_document_alerts 
	          SET status = $1, resolved_at = $2 
	          WHERE organization_id = $3 AND document_id = $4 AND status = $5`

	_, err := r.pool.Exec(ctx, query, domain.AlertStatusResolved, time.Now(), orgID, docID, domain.AlertStatusActive)
	if err != nil {
		return fmt.Errorf("failed to resolve document alerts: %w", err)
	}
	return nil
}

func (r *PostgresDriverDocumentRepository) CreateFile(ctx context.Context, orgID string, file domain.DriverDocumentFile) (domain.DriverDocumentFile, error) {
	if file.ID == "" {
		file.ID = fmt.Sprintf("FIL-%04d", rand.Intn(9000)+1000)
	}
	file.OrganizationID = orgID
	file.CreatedAt = time.Now()
	file.UpdatedAt = file.CreatedAt

	query := `INSERT INTO driver_document_files (id, organization_id, document_id, file_path, file_name, file_size, content_type, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		file.ID,
		file.OrganizationID,
		file.DocumentID,
		file.FilePath,
		file.FileName,
		file.FileSize,
		file.ContentType,
		file.CreatedAt,
		file.UpdatedAt,
	)

	if err != nil {
		return domain.DriverDocumentFile{}, fmt.Errorf("failed to insert document file record: %w", err)
	}

	return file, nil
}

func (r *PostgresDriverDocumentRepository) GetFileByID(ctx context.Context, orgID string, id string) (domain.DriverDocumentFile, error) {
	query := `SELECT id, organization_id, document_id, file_path, file_name, file_size, content_type, created_at, updated_at 
	          FROM driver_document_files 
	          WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL`

	var f domain.DriverDocumentFile
	err := r.pool.QueryRow(ctx, query, orgID, id).Scan(
		&f.ID,
		&f.OrganizationID,
		&f.DocumentID,
		&f.FilePath,
		&f.FileName,
		&f.FileSize,
		&f.ContentType,
		&f.CreatedAt,
		&f.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DriverDocumentFile{}, domain.ErrFileNotFound
		}
		return domain.DriverDocumentFile{}, fmt.Errorf("failed to retrieve document file: %w", err)
	}

	return f, nil
}

func (r *PostgresDriverDocumentRepository) DeleteFile(ctx context.Context, orgID string, id string) error {
	query := `UPDATE driver_document_files 
	          SET deleted_at = $1, updated_at = $1 
	          WHERE organization_id = $2 AND id = $3 AND deleted_at IS NULL`

	cmd, err := r.pool.Exec(ctx, query, time.Now(), orgID, id)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrFileNotFound
	}
	return nil
}
