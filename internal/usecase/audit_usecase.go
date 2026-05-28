package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type AuditUseCase interface {
	Log(ctx context.Context, action string, entityType string, entityID *string, metadata map[string]interface{})
	GetLogsByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*domain.AuditLog, error)
}

type auditUseCase struct {
	repo    domain.AuditLogRepository
	logger  *slog.Logger
	logChan chan *domain.AuditLog
}

// NewAuditUseCase instantiates a new AuditUseCase and starts a background worker.
func NewAuditUseCase(repo domain.AuditLogRepository, logger *slog.Logger) AuditUseCase {
	uc := &auditUseCase{
		repo:    repo,
		logger:  logger,
		logChan: make(chan *domain.AuditLog, 1000), // Buffer size of 1000
	}

	// Start the background audit logger worker
	go uc.startWorker()

	return uc
}

func (uc *auditUseCase) Log(ctx context.Context, action string, entityType string, entityID *string, metadata map[string]interface{}) {
	// 1. Sanitize metadata to remove sensitive credentials
	sanitizedMetadata := uc.sanitizeMetadata(metadata)

	// 2. Extract values from request context
	actorUserIDVal := extractStringFromContext(ctx, domain.UserIDContextKey)
	orgIDVal := extractStringFromContext(ctx, domain.OrganizationIDContextKey)
	ipAddress := extractStringFromContext(ctx, domain.ClientIPContextKey)
	userAgent := extractStringFromContext(ctx, domain.UserAgentContextKey)

	var actorUserID *string
	if actorUserIDVal != "" {
		actorUserID = &actorUserIDVal
	}

	var orgID *string
	if orgIDVal != "" {
		orgID = &orgIDVal
	}

	if ipAddress == "" {
		ipAddress = "unknown"
	}

	var uaPtr *string
	if userAgent != "" {
		uaPtr = &userAgent
	}

	// 3. Build AuditLog entry
	auditLog := &domain.AuditLog{
		ID:             generateUUID(),
		ActorUserID:    actorUserID,
		OrganizationID: orgID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       entityID,
		Metadata:       sanitizedMetadata,
		IPAddress:      ipAddress,
		UserAgent:      uaPtr,
		CreatedAt:      time.Now(),
	}

	// 4. Send to background logging queue
	select {
	case uc.logChan <- auditLog:
	default:
		// Queue is full: fall back to a separate background goroutine to avoid blocking the caller
		uc.logger.Warn("Audit queue full, falling back to spawn-on-demand goroutine")
		go func(log *domain.AuditLog) {
			if err := uc.repo.Create(context.Background(), log); err != nil {
				uc.logger.Error("Failed to write audit log in fallback goroutine", slog.String("err", err.Error()))
			}
		}(auditLog)
	}
}

func (uc *auditUseCase) GetLogsByOrganization(ctx context.Context, orgID string, limit, offset int) ([]*domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 250 {
		limit = 250
	}
	return uc.repo.GetByOrganization(ctx, orgID, limit, offset)
}

func (uc *auditUseCase) startWorker() {
	uc.logger.Info("Audit log background worker started successfully")
	for log := range uc.logChan {
		// Use Background context to ensure write completes even if the original request context was cancelled
		if err := uc.repo.Create(context.Background(), log); err != nil {
			uc.logger.Error("Failed to write audit log in background worker", slog.String("err", err.Error()))
		}
	}
}

func (uc *auditUseCase) sanitizeMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return map[string]interface{}{}
	}

	sanitized := make(map[string]interface{})
	sensitiveKeys := []string{"password", "password_hash", "token", "secret", "jwt", "pass", "credit_card"}

	for k, v := range metadata {
		isSensitive := false
		lowerKey := strings.ToLower(k)
		for _, sk := range sensitiveKeys {
			if strings.Contains(lowerKey, sk) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}

// Helpers
func extractStringFromContext(ctx context.Context, key domain.ContextKey) string {
	if val, ok := ctx.Value(key).(string); ok {
		return val
	}
	return ""
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
