// Package application implements the inbound use-case ports by coordinating
// domain entities and outbound adapter ports.
package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/logger"
	"github.com/diego/go-refinery/internal/ports"
)

const defaultSampleRowLimit = 100

// IngestService orchestrates file upload streaming and initial schema inference.
type IngestService struct {
	datasets ports.DatasetRepository
	storage  ports.FileStorage
	parsers  ports.StreamParserFactory
	log      *logger.Logger
}

// NewIngestService constructs an IngestService with its required outbound ports.
func NewIngestService(
	datasets ports.DatasetRepository,
	storage ports.FileStorage,
	parsers ports.StreamParserFactory,
) *IngestService {
	return &IngestService{
		datasets: datasets,
		storage:  storage,
		parsers:  parsers,
		log:      logger.Default().With(logger.F("component", "ingest_service")),
	}
}

// SetLogger overrides the default structured logger on IngestService.
func (s *IngestService) SetLogger(l *logger.Logger) {
	if l != nil {
		s.log = l.With(logger.F("component", "ingest_service"))
	}
}

// UploadAndInfer streams the uploaded file to disk, reads a sample of rows,
// infers column types, and persists the Dataset in DRAFT status.
func (s *IngestService) UploadAndInfer(ctx context.Context, input ports.UploadInput) (*dataset.Dataset, error) {
	start := time.Now()
	id, err := newRandomID()
	if err != nil {
		return nil, fmt.Errorf("generating dataset id: %w", err)
	}
	dsID := dataset.ID(id)

	s.log.InfoContext(ctx, "starting dataset upload stream",
		logger.F("dataset_id", dsID),
		logger.F("filename", input.OriginalName),
		logger.F("format", input.Format),
	)

	storagePath, written, err := s.storage.SaveStream(ctx, dsID, input.Reader)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to save upload stream to storage",
			logger.F("dataset_id", dsID),
			logger.Err(err),
		)
		return nil, fmt.Errorf("saving upload stream: %w", err)
	}

	rc, err := s.storage.OpenStream(ctx, storagePath)
	if err != nil {
		_ = s.storage.Delete(ctx, storagePath)
		return nil, fmt.Errorf("reopening stored file for sampling: %w", err)
	}
	defer rc.Close()

	iter, err := s.parsers.NewIterator(ctx, input.Format, rc)
	if err != nil {
		_ = s.storage.Delete(ctx, storagePath)
		s.log.WarnContext(ctx, "failed to initialize stream parser for uploaded file",
			logger.F("dataset_id", dsID),
			logger.F("format", input.Format),
			logger.Err(err),
		)
		return nil, fmt.Errorf("creating stream parser: %w", err)
	}
	defer iter.Close()

	var sampleRows [][]string
	for len(sampleRows) < defaultSampleRowLimit {
		row, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_ = s.storage.Delete(ctx, storagePath)
			s.log.WarnContext(ctx, "error reading sample row from uploaded file",
				logger.F("dataset_id", dsID),
				logger.F("sample_row", len(sampleRows)+1),
				logger.Err(err),
			)
			return nil, fmt.Errorf("reading sample row: %w", err)
		}
		sampleRows = append(sampleRows, row)
	}

	schema := dataset.InferSchema(iter.Headers(), sampleRows)
	now := time.Now().UTC()

	ds := &dataset.Dataset{
		ID:           dsID,
		OriginalName: input.OriginalName,
		StoragePath:  storagePath,
		Format:       input.Format,
		Delimiter:    ',',
		HasHeader:    true,
		SizeBytes:    written,
		Status:       dataset.StatusDraft,
		Schema:       schema,
		SampleRows:   sampleRows,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := ds.Validate(); err != nil {
		_ = s.storage.Delete(ctx, storagePath)
		return nil, err
	}

	if err := s.datasets.Create(ctx, ds); err != nil {
		_ = s.storage.Delete(ctx, storagePath)
		s.log.ErrorContext(ctx, "failed to persist dataset metadata",
			logger.F("dataset_id", dsID),
			logger.Err(err),
		)
		return nil, fmt.Errorf("persisting dataset: %w", err)
	}

	s.log.InfoContext(ctx, "dataset uploaded and schema inferred",
		logger.F("dataset_id", dsID),
		logger.F("filename", ds.OriginalName),
		logger.F("size_bytes", written),
		logger.F("columns", len(schema.Columns)),
		logger.F("sample_rows", len(sampleRows)),
		logger.F("duration_ms", time.Since(start).Milliseconds()),
	)

	return ds, nil
}

// GetDataset retrieves a Dataset by ID.
func (s *IngestService) GetDataset(ctx context.Context, id dataset.ID) (*dataset.Dataset, error) {
	return s.datasets.GetByID(ctx, id)
}

// ListDatasets retrieves a paginated list of Datasets.
func (s *IngestService) ListDatasets(ctx context.Context, filter ports.PageFilter) ([]dataset.Dataset, error) {
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 50
	}
	return s.datasets.List(ctx, filter)
}

func newRandomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
