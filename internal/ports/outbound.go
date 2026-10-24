package ports

import (
	"context"
	"io"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
)

// DatasetRepository defines the outbound persistence port for Datasets and their Schemas.
type DatasetRepository interface {
	Create(ctx context.Context, ds *dataset.Dataset) error
	GetByID(ctx context.Context, id dataset.ID) (*dataset.Dataset, error)
	List(ctx context.Context, filter PageFilter) ([]dataset.Dataset, error)
	UpdateStatus(ctx context.Context, id dataset.ID, status dataset.Status, totalRows int64) error
	SaveSchema(ctx context.Context, datasetID dataset.ID, schema dataset.Schema) error
}

// JobRepository defines the outbound persistence port for asynchronous Jobs and row errors.
type JobRepository interface {
	Create(ctx context.Context, j *job.Job) error
	GetByID(ctx context.Context, id job.ID) (*job.Job, error)
	GetLatestByDataset(ctx context.Context, datasetID dataset.ID) (*job.Job, error)
	UpdateProgress(ctx context.Context, j *job.Job) error
	ListErrors(ctx context.Context, jobID job.ID, filter PageFilter) ([]job.RowError, error)
}

// RecordBatchWriter persists batches of refined records and row errors atomically
// within a single SQLite write transaction.
type RecordBatchWriter interface {
	WriteBatch(ctx context.Context, jobID job.ID, records []dataset.Record, rowErrors []job.RowError) error
	ListRecords(ctx context.Context, datasetID dataset.ID, filter PageFilter) ([]dataset.Record, error)
}

// FileStorage defines the outbound port for streaming raw uploaded files to and from disk.
type FileStorage interface {
	SaveStream(ctx context.Context, datasetID dataset.ID, r io.Reader) (storagePath string, bytesWritten int64, err error)
	OpenStream(ctx context.Context, storagePath string) (io.ReadCloser, error)
	Delete(ctx context.Context, storagePath string) error
}

// RowIterator iterates sequentially over a raw dataset stream in O(1) memory.
type RowIterator interface {
	// Headers returns the detected or synthesized column headers.
	Headers() []string
	// Next advances to the next row, returning io.EOF when the stream ends.
	Next() ([]string, error)
	// Close releases any underlying file or archive resources.
	Close() error
}

// StreamParserFactory creates a format-specific RowIterator for a raw file stream.
type StreamParserFactory interface {
	NewIterator(ctx context.Context, format dataset.FileFormat, r io.ReadCloser) (RowIterator, error)
}

// ProgressBroker defines the outbound Pub/Sub port for real-time job progress events (SSE).
type ProgressBroker interface {
	Publish(event job.ProgressEvent)
	Subscribe(ctx context.Context, jobID job.ID) (<-chan job.ProgressEvent, func())
}
