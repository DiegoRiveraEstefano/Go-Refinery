package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/domain/pipeline"
)

const sseThrottleInterval = 250 * time.Millisecond

func (s *RefineryService) workerLoop(ctx context.Context, _ int) {
	for {
		select {
		case <-ctx.Done():
			return
		case jobID, ok := <-s.jobQueue:
			if !ok {
				return
			}
			s.processJob(ctx, jobID)
		}
	}
}

// processJob executes a single refinement job end-to-end: compiling column pipelines,
// streaming rows in O(1) memory, writing 500-row batches to SQLite, and publishing
// throttled SSE telemetry events.
func (s *RefineryService) processJob(ctx context.Context, jobID job.ID) {
	j, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return
	}

	ds, err := s.datasets.GetByID(ctx, j.DatasetID)
	if err != nil {
		s.failJob(ctx, j, time.Now().UTC(), fmt.Sprintf("loading dataset: %v", err))
		return
	}

	colPipelines, err := buildColumnPipelines(ds.Schema)
	if err != nil {
		s.failJob(ctx, j, time.Now().UTC(), fmt.Sprintf("compiling pipeline rules: %v", err))
		return
	}

	rc, err := s.storage.OpenStream(ctx, ds.StoragePath)
	if err != nil {
		s.failJob(ctx, j, time.Now().UTC(), fmt.Sprintf("opening dataset stream: %v", err))
		return
	}
	defer rc.Close()

	iter, err := s.parsers.NewIterator(ctx, ds.Format, rc)
	if err != nil {
		s.failJob(ctx, j, time.Now().UTC(), fmt.Sprintf("initializing stream parser: %v", err))
		return
	}
	defer iter.Close()

	startTime := time.Now().UTC()
	j.Status = job.StatusRunning
	j.StartedAt = &startTime
	_ = s.jobs.UpdateProgress(ctx, j)
	_ = s.datasets.UpdateStatus(ctx, ds.ID, dataset.StatusProcessing, ds.TotalRows)
	s.publishEvent(j, startTime, "Refinement started")

	batchSize := s.batchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}

	recordsBatch := make([]dataset.Record, 0, batchSize)
	errorsBatch := make([]job.RowError, 0, 64)
	lastBroadcast := startTime
	var rowNumber int64

	flushBatch := func() error {
		if len(recordsBatch) == 0 && len(errorsBatch) == 0 {
			return nil
		}
		if err := s.records.WriteBatch(ctx, j.ID, recordsBatch, errorsBatch); err != nil {
			return fmt.Errorf("writing record batch: %w", err)
		}
		recordsBatch = recordsBatch[:0]
		errorsBatch = errorsBatch[:0]

		if err := s.jobs.UpdateProgress(ctx, j); err != nil {
			return fmt.Errorf("updating job progress: %w", err)
		}

		now := time.Now().UTC()
		if now.Sub(lastBroadcast) >= sseThrottleInterval {
			s.publishEvent(j, startTime, "Processing batch")
			lastBroadcast = now
		}
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			_ = flushBatch()
			s.cancelJob(j, startTime, "Server shutting down or context cancelled")
			return
		default:
		}

		rawRow, err := iter.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			rowNumber++
			j.ProcessedRows++
			j.ErrorRows++
			errorsBatch = append(errorsBatch, job.RowError{
				JobID:      j.ID,
				RowNumber:  rowNumber,
				ColumnName: "_row",
				RawValue:   "",
				ErrorCode:  "ERR_PARSE_ROW",
				Message:    err.Error(),
				CreatedAt:  time.Now().UTC(),
			})
		} else {
			rowNumber++
			j.ProcessedRows++

			rec, rowErrs, dropped := evaluateRow(ds.ID, j.ID, rowNumber, rawRow, colPipelines)
			if len(rowErrs) > 0 {
				j.ErrorRows++
				errorsBatch = append(errorsBatch, rowErrs...)
			} else if !dropped {
				j.ValidRows++
				recordsBatch = append(recordsBatch, rec)
			}
		}

		if j.MaxErrors > 0 && j.ErrorRows >= j.MaxErrors {
			_ = flushBatch()
			s.failJob(ctx, j, startTime, job.ErrMaxErrorsReached.Error())
			return
		}

		if len(recordsBatch)+len(errorsBatch) >= batchSize {
			if err := flushBatch(); err != nil {
				s.failJob(ctx, j, startTime, err.Error())
				return
			}
		}
	}

	if err := flushBatch(); err != nil {
		s.failJob(ctx, j, startTime, err.Error())
		return
	}

	finishTime := time.Now().UTC()
	j.TotalRows = j.ProcessedRows
	j.Status = job.StatusCompleted
	j.FinishedAt = &finishTime
	_ = s.jobs.UpdateProgress(ctx, j)
	_ = s.datasets.UpdateStatus(ctx, ds.ID, dataset.StatusCompleted, j.TotalRows)
	s.publishEvent(j, startTime, "Refinement completed successfully")
}

func buildColumnPipelines(schema dataset.Schema) ([]pipeline.ColumnPipeline, error) {
	result := make([]pipeline.ColumnPipeline, len(schema.Columns))
	for i, col := range schema.Columns {
		stages := make([]pipeline.Stage, 0, len(col.RuleSpecs))
		for _, spec := range col.RuleSpecs {
			st, err := pipeline.BuildStage(spec)
			if err != nil {
				return nil, fmt.Errorf("column %q: %w", col.NormalizedName, err)
			}
			stages = append(stages, st)
		}
		result[i] = pipeline.ColumnPipeline{
			Column: col,
			Stages: stages,
		}
	}
	return result, nil
}

func evaluateRow(
	datasetID dataset.ID,
	jobID job.ID,
	rowNum int64,
	rawRow []string,
	colPipelines []pipeline.ColumnPipeline,
) (dataset.Record, []job.RowError, bool) {
	values := make(map[string]any, len(colPipelines))
	var rowErrors []job.RowError
	now := time.Now().UTC()

	for i := range colPipelines {
		cp := &colPipelines[i]
		var rawCell string
		if cp.Column.Index < len(rawRow) {
			rawCell = rawRow[cp.Column.Index]
		}

		val, err := cp.Execute(rawCell)
		if err != nil {
			if errors.Is(err, pipeline.ErrDropRow) {
				return dataset.Record{}, nil, true
			}
			rowErrors = append(rowErrors, job.RowError{
				JobID:      jobID,
				RowNumber:  rowNum,
				ColumnName: cp.Column.NormalizedName,
				RawValue:   rawCell,
				ErrorCode:  classifyPipelineError(err),
				Message:    err.Error(),
				CreatedAt:  now,
			})
			continue
		}
		values[cp.Column.NormalizedName] = val
	}

	if len(rowErrors) > 0 {
		return dataset.Record{}, rowErrors, false
	}

	return dataset.Record{
		DatasetID: datasetID,
		RowNumber: rowNum,
		Values:    values,
		CreatedAt: now,
	}, nil, false
}

func classifyPipelineError(err error) string {
	switch {
	case errors.Is(err, pipeline.ErrRequiredNull):
		return "ERR_REQUIRED_NULL"
	case errors.Is(err, pipeline.ErrTypeCastFailed):
		return "ERR_TYPE_CAST"
	case errors.Is(err, pipeline.ErrValidationCheck):
		return "ERR_VALIDATION"
	default:
		return "ERR_PIPELINE_RULE"
	}
}

func (s *RefineryService) failJob(ctx context.Context, j *job.Job, startTime time.Time, msg string) {
	finishTime := time.Now().UTC()
	j.Status = job.StatusFailed
	j.ErrorMessage = msg
	j.FinishedAt = &finishTime
	_ = s.jobs.UpdateProgress(ctx, j)
	_ = s.datasets.UpdateStatus(ctx, j.DatasetID, dataset.StatusFailed, j.ProcessedRows)
	s.publishEvent(j, startTime, msg)
}

func (s *RefineryService) cancelJob(j *job.Job, startTime time.Time, msg string) {
	saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	finishTime := time.Now().UTC()
	j.Status = job.StatusCancelled
	j.ErrorMessage = msg
	j.FinishedAt = &finishTime
	_ = s.jobs.UpdateProgress(saveCtx, j)
	_ = s.datasets.UpdateStatus(saveCtx, j.DatasetID, dataset.StatusFailed, j.ProcessedRows)
	s.publishEvent(j, startTime, msg)
}

func (s *RefineryService) publishEvent(j *job.Job, startTime time.Time, message string) {
	if s.broker == nil {
		return
	}
	now := time.Now().UTC()
	var rps float64
	if elapsed := now.Sub(startTime).Seconds(); elapsed > 0 && j.ProcessedRows > 0 {
		rps = float64(j.ProcessedRows) / elapsed
	}

	s.broker.Publish(job.ProgressEvent{
		JobID:         j.ID,
		DatasetID:     j.DatasetID,
		Status:        j.Status,
		ProcessedRows: j.ProcessedRows,
		TotalRows:     j.TotalRows,
		ValidRows:     j.ValidRows,
		ErrorRows:     j.ErrorRows,
		Percentage:    j.Percentage(),
		RowsPerSecond: rps,
		Message:       message,
		Timestamp:     now,
	})
}
