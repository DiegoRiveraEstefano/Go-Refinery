// Package storage implements the FileStorage outbound port on the local filesystem.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/diego/go-refinery/internal/domain/dataset"
)

// LocalStorage persists uploaded dataset streams inside a restricted base directory.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage initializes the base storage directory with 0700 permissions.
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating storage base dir: %w", err)
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

// SaveStream writes an io.Reader stream directly to disk in O(1) memory.
func (s *LocalStorage) SaveStream(_ context.Context, datasetID dataset.ID, r io.Reader) (string, int64, error) {
	cleanID := filepath.Base(string(datasetID))
	fullPath := filepath.Join(s.baseDir, cleanID+".raw")

	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", 0, fmt.Errorf("opening target file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		_ = os.Remove(fullPath)
		return "", 0, fmt.Errorf("streaming to disk: %w", err)
	}

	return fullPath, n, nil
}

// OpenStream opens a stored dataset file for sequential reading.
func (s *LocalStorage) OpenStream(_ context.Context, storagePath string) (io.ReadCloser, error) {
	return os.Open(storagePath)
}

// Delete removes a stored dataset file from disk.
func (s *LocalStorage) Delete(_ context.Context, storagePath string) error {
	return os.Remove(storagePath)
}
