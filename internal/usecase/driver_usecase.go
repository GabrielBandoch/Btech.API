package usecase

import (
	"context"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type DriverUseCase interface {
	GetDrivers(ctx context.Context, orgID string) ([]domain.Driver, error)
	GetDriverByID(ctx context.Context, orgID string, id string) (domain.Driver, error)
	CreateDriver(ctx context.Context, orgID string, driver domain.Driver) (domain.Driver, error)
	GetDriverAuditLogs(ctx context.Context, orgID string, id string) ([]*domain.AuditLog, error)
}

type driverUseCase struct {
	repo               domain.DriverRepository
	entitlementUseCase EntitlementUseCase
	auditUseCase       AuditUseCase
	auditLogRepo       domain.AuditLogRepository
}

// NewDriverUseCase creates a new instance of DriverUseCase with injected repository, entitlement usecase, audit usecase, and audit log repository.
func NewDriverUseCase(
	repo domain.DriverRepository,
	entitlementUseCase EntitlementUseCase,
	auditUseCase AuditUseCase,
	auditLogRepo domain.AuditLogRepository,
) DriverUseCase {
	return &driverUseCase{
		repo:               repo,
		entitlementUseCase: entitlementUseCase,
		auditUseCase:       auditUseCase,
		auditLogRepo:       auditLogRepo,
	}
}

// GetDrivers retrieves all drivers for the organization.
func (uc *driverUseCase) GetDrivers(ctx context.Context, orgID string) ([]domain.Driver, error) {
	return uc.repo.GetAll(ctx, orgID)
}

// GetDriverByID retrieves a single driver by their ID within the organization.
func (uc *driverUseCase) GetDriverByID(ctx context.Context, orgID string, id string) (domain.Driver, error) {
	return uc.repo.GetByID(ctx, orgID, id)
}

// CreateDriver registers a new driver for the organization.
func (uc *driverUseCase) CreateDriver(ctx context.Context, orgID string, driver domain.Driver) (domain.Driver, error) {
	// Count existing drivers to evaluate quota
	count, err := uc.repo.Count(ctx, orgID)
	if err != nil {
		return domain.Driver{}, err
	}

	allowed, err := uc.entitlementUseCase.EvaluateQuota(ctx, orgID, domain.EntitlementDriversMax, count)
	if err != nil {
		return domain.Driver{}, err
	}
	if !allowed {
		return domain.Driver{}, domain.ErrQuotaExceeded
	}

	d, err := uc.repo.Create(ctx, orgID, driver)
	if err != nil {
		return d, err
	}

	uc.auditUseCase.Log(ctx, domain.EventDriverCreate, "driver", &d.ID, map[string]interface{}{
		"driver_name": d.Name,
		"license":     d.LicenseExpiry,
	})

	return d, nil
}

// GetDriverAuditLogs retrieves audit logs for a specific driver and their related documents.
func (uc *driverUseCase) GetDriverAuditLogs(ctx context.Context, orgID string, id string) ([]*domain.AuditLog, error) {
	driver, err := uc.repo.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	return uc.auditLogRepo.GetByDriver(ctx, orgID, id, driver.Name)
}

