package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/btech/fleetcontrol-api/internal/domain"
)

type LocalFileStorage struct {
	baseDir string
	baseURL string
}

// NewLocalFileStorage creates a local file storage instance.
func NewLocalFileStorage(baseDir string, baseURL string) domain.StorageService {
	_ = os.MkdirAll(baseDir, os.ModePerm)
	return &LocalFileStorage{
		baseDir: baseDir,
		baseURL: baseURL,
	}
}

func (s *LocalFileStorage) Upload(ctx context.Context, bucket string, key string, file io.Reader, contentType string) (string, error) {
	destPath := filepath.Join(s.baseDir, bucket, key)
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create storage directories: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	return fmt.Sprintf("%s/%s/%s", s.baseURL, bucket, key), nil
}

func (s *LocalFileStorage) GetURL(ctx context.Context, bucket string, key string) (string, error) {
	return fmt.Sprintf("%s/%s/%s", s.baseURL, bucket, key), nil
}

func (s *LocalFileStorage) Delete(ctx context.Context, bucket string, key string) error {
	destPath := filepath.Join(s.baseDir, bucket, key)
	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file from storage: %w", err)
	}
	return nil
}
