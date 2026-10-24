// Package ports defines the hexagonal boundary interfaces (Inbound and Outbound Ports)
// that decouple the application core from transport, storage, and parsing adapters.
package ports

import (
	"context"
	"io"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
)

// UploadInput encapsulates the metadata and raw stream of an uploaded file.
type UploadInput struct {
	OriginalName string
	Format       dataset.FileFormat
	SizeBytes    int64
	Reader       io.Reader
}

// PageFilter defines pagination parameters for read queries.
type PageFilter struct {
	Limit  int
	Offset int
}

// IngestUseCase defines the primary port for uploading raw datasets and managing metadata.
type IngestUseCase interface {
	// UploadAndInfer streams an uploaded file to storage, samples initial rows,
	// infers the column schema, and persists the Dataset in DRAFT state.
	UploadAndInfer(ctx context.Context, input UploadInput) (*dataset.Dataset, error)

	// GetDataset retrieves a Dataset along with its inferred/configured Schema.
	GetDataset(ctx context.Context, id dataset.ID) (*dataset.Dataset, error)

	// ListDatasets returns a paginated list of uploaded Datasets.
	ListDatasets(ctx context.Context, filter PageFilter) ([]dataset.Dataset, error)
}

// SchemaUseCase defines the primary port for inspecting and updating column schemas and rules.
type SchemaUseCase interface {
	// UpdateSchema validates and saves user-configured target types and cleaning rules.
	UpdateSchema(ctx context.Context, datasetID dataset.ID, schema dataset.Schema) error
}

// RefineryUseCase defines the primary port for starting, monitoring, and querying refinement jobs.
type RefineryUseCase interface {
	// StartRefinementJob creates a QUEUED job and schedules it on the background worker pool.
	StartRefinementJob(ctx context.Context, datasetID dataset.ID) (*job.Job, error)

	// GetJob returns the current state and metrics of a refinement Job.
	GetJob(ctx context.Context, jobID job.ID) (*job.Job, error)

	// ListJobErrors returns paginated row-level validation errors for a Job.
	ListJobErrors(ctx context.Context, jobID job.ID, filter PageFilter) ([]job.RowError, error)

	// ListRecords returns paginated cleaned records for a refined Dataset.
	ListRecords(ctx context.Context, datasetID dataset.ID, filter PageFilter) ([]dataset.Record, error)
}
