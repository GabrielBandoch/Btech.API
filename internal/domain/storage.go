package domain

import (
	"context"
	"io"
)

// StorageService defines the contract for file attachment storage.
type StorageService interface {
	Upload(ctx context.Context, bucket string, key string, file io.Reader, contentType string) (string, error)
	GetURL(ctx context.Context, bucket string, key string) (string, error)
	Delete(ctx context.Context, bucket string, key string) error
}
