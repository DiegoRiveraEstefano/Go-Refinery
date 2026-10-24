package application

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/logger"
	"github.com/diego/go-refinery/internal/ports"
)

const (
	defaultBatchSize = 500
	defaultMaxErrors = 1000
)

// Compile-time check that RefineryService satisfies ports.RefineryUseCase.
var _ ports.RefineryUseCase = (*RefineryService)(nil)

// RefineryService schedules, executes, and monitors asynchronous dataset refinement jobs.
type RefineryService struct {
	datasets  ports.DatasetRepository
	jobs      ports.JobRepository
	records   ports.RecordBatchWriter
	storage   ports.FileStorage
	parsers   ports.StreamParserFactory
	broker    ports.ProgressBroker
	jobQueue  chan job.ID
	batchSize int
	wg        sync.WaitGroup
	log       *logger.Logger
}

// NewRefineryService creates a RefineryService with a bounded internal job queue.
func NewRefineryService(
	datasets ports.DatasetRepository,
	jobs ports.JobRepository,
	records ports.RecordBatchWriter,
	storage ports.FileStorage,
	parsers ports.StreamParserFactory,
	broker ports.ProgressBroker,
	queueSize int,
) *RefineryService {
	if queueSize <= 0 {
		queueSize = 64
	}
	return &RefineryService{
		datasets:  datasets,
		jobs:      jobs,
		records:   records,
		storage:   storage,
		parsers:   parsers,
		broker:    broker,
		jobQueue:  make(chan job.ID, queueSize),
		batchSize: defaultBatchSize,
		log:       logger.Default().With(logger.F("component", "refinery_service")),
	}
}

// SetLogger overrides the default structured logger on RefineryService.
func (s *RefineryService) SetLogger(l *logger.Logger) {
	if l != nil {
		s.log = l.With(logger.F("component", "refinery_service"))
	}
}

// SetBatchSize overrides the default number of rows flushed per SQLite transaction.
func (s *RefineryService) SetBatchSize(size int) {
	if size > 0 {
		s.batchSize = size
	}
}

// StartWorkers spawns n background worker goroutines that consume queued job IDs
// until ctx is cancelled. If n <= 0, runtime.NumCPU() workers are started.
func (s *RefineryService) StartWorkers(ctx context.Context, n int) {
	if n <= 0 {
		n = runtime.NumCPU()
	}
	if n < 1 {
		n = 1
	}

	s.log.Info("starting background refinement worker pool",
		logger.F("workers", n),
		logger.F("batch_size", s.batchSize),
	)

	for i := 0; i < n; i++ {
		s.wg.Add(1)
		go func(workerID int) {
			defer s.wg.Done()
			s.workerLoop(ctx, workerID)
		}(i + 1)
	}
}

// Wait blocks until all background workers spawned by StartWorkers have exited.
func (s *RefineryService) Wait() {
	s.wg.Wait()
}

// StartRefinementJob creates a new QUEUED job for the given dataset and enqueues it.
func (s *RefineryService) StartRefinementJob(ctx context.Context, datasetID dataset.ID) (*job.Job, error) {
	ds, err := s.datasets.GetByID(ctx, datasetID)
	if err != nil {
		return nil, err
	}

	if latest, err := s.jobs.GetLatestByDataset(ctx, datasetID); err == nil && latest != nil {
		if !latest.IsTerminal() {
			s.log.WarnContext(ctx, "rejected duplicate job start while active job exists",
				logger.F("dataset_id", datasetID),
				logger.F("active_job_id", latest.ID),
			)
			return nil, job.ErrJobAlreadyActive
		}
	}

	id, err := newRandomID()
	if err != nil {
		return nil, fmt.Errorf("generating job id: %w", err)
	}

	now := time.Now().UTC()
	j := &job.Job{
		ID:        job.ID(id),
		DatasetID: ds.ID,
		Status:    job.StatusQueued,
		TotalRows: ds.TotalRows,
		MaxErrors: defaultMaxErrors,
		CreatedAt: now,
	}

	if err := s.jobs.Create(ctx, j); err != nil {
		s.log.ErrorContext(ctx, "failed to create job record in sqlite",
			logger.F("dataset_id", datasetID),
			logger.Err(err),
		)
		return nil, fmt.Errorf("creating job record: %w", err)
	}

	s.log.InfoContext(ctx, "enqueued asynchronous refinement job",
		logger.F("job_id", j.ID),
		logger.F("dataset_id", ds.ID),
	)

	s.publishEvent(j, time.Now().UTC(), "Job queued for refinement")

	select {
	case s.jobQueue <- j.ID:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return j, nil
}

// GetJob retrieves a Job by its ID.
func (s *RefineryService) GetJob(ctx context.Context, jobID job.ID) (*job.Job, error) {
	return s.jobs.GetByID(ctx, jobID)
}

// ListJobErrors returns paginated row-level validation errors for a Job.
func (s *RefineryService) ListJobErrors(ctx context.Context, jobID job.ID, filter ports.PageFilter) ([]job.RowError, error) {
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	return s.jobs.ListErrors(ctx, jobID, filter)
}

// ListRecords returns paginated cleaned records for a Dataset.
func (s *RefineryService) ListRecords(ctx context.Context, datasetID dataset.ID, filter ports.PageFilter) ([]dataset.Record, error) {
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	return s.records.ListRecords(ctx, datasetID, filter)
}

// SearchRecords scans refined records for datasetID in 500-row batches and filters them
// using the compiled SearchQuery (supporting text, regex, operators, and Mongo-style JSON/DSL).
func (s *RefineryService) SearchRecords(
	ctx context.Context,
	datasetID dataset.ID,
	query dataset.SearchQuery,
	filter ports.PageFilter,
) ([]dataset.Record, int, error) {
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	if query.IsEmpty() {
		recs, err := s.ListRecords(ctx, datasetID, filter)
		return recs, len(recs), err
	}

	compiled, err := dataset.CompileSearchQuery(query)
	if err != nil {
		s.log.WarnContext(ctx, "invalid dataset search query",
			logger.F("dataset_id", datasetID),
			logger.F("query", query.Query),
			logger.Err(err),
		)
		return nil, 0, err
	}

	start := time.Now()
	var (
		scanOffset   int
		totalMatched int
		pageResults  = make([]dataset.Record, 0, filter.Limit)
	)

	for {
		batch, err := s.records.ListRecords(ctx, datasetID, ports.PageFilter{
			Limit:  exportBatchPageSize,
			Offset: scanOffset,
		})
		if err != nil {
			return nil, 0, err
		}
		if len(batch) == 0 {
			break
		}

		for _, rec := range batch {
			if compiled.Match(rec) {
				if totalMatched >= filter.Offset && len(pageResults) < filter.Limit {
					pageResults = append(pageResults, rec)
				}
				totalMatched++
			}
		}

		if len(batch) < exportBatchPageSize {
			break
		}
		scanOffset += len(batch)
	}

	s.log.InfoContext(ctx, "executed dataset search query",
		logger.F("dataset_id", datasetID),
		logger.F("query", query.Query),
		logger.F("field", query.Field),
		logger.F("op", query.Operator),
		logger.F("value", query.Value),
		logger.F("matched_rows", totalMatched),
		logger.F("returned_rows", len(pageResults)),
		logger.F("duration_ms", time.Since(start).Milliseconds()),
	)

	return pageResults, totalMatched, nil
}

// JobQueue exposes the read-only channel consumed by background workers.
func (s *RefineryService) JobQueue() <-chan job.ID {
	return s.jobQueue
}
