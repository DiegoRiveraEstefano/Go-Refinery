// Package job defines the asynchronous job state machine, row-level validation
// error records, and real-time telemetry events broadcast over SSE.
package job

import (
	"errors"
	"time"

	"github.com/diego/go-refinery/internal/domain/dataset"
)

// Domain sentinel errors for job operations.
var (
	ErrJobNotFound      = errors.New("job not found")
	ErrJobAlreadyActive = errors.New("a refinement job is already running for this dataset")
	ErrMaxErrorsReached = errors.New("job aborted: maximum row error threshold exceeded")
)

// ID uniquely identifies an asynchronous refinement Job.
type ID string

// Status represents the execution state of a Job.
type Status string

const (
	StatusQueued    Status = "QUEUED"
	StatusRunning   Status = "RUNNING"
	StatusCompleted Status = "COMPLETED"
	StatusFailed    Status = "FAILED"
	StatusCancelled Status = "CANCELLED"
)

// Job tracks the asynchronous execution of a dataset refinement pipeline.
type Job struct {
	ID            ID
	DatasetID     dataset.ID
	Status        Status
	TotalRows     int64
	ProcessedRows int64
	ValidRows     int64
	ErrorRows     int64
	MaxErrors     int64
	ErrorMessage  string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
}

// Percentage computes the completion percentage (0..100) of the Job.
func (j *Job) Percentage() float64 {
	if j.TotalRows <= 0 {
		if j.Status == StatusCompleted {
			return 100.0
		}
		return 0.0
	}
	pct := (float64(j.ProcessedRows) / float64(j.TotalRows)) * 100.0
	if pct > 100.0 {
		return 100.0
	}
	return pct
}

// IsTerminal reports whether the Job has reached a final state.
func (j *Job) IsTerminal() bool {
	switch j.Status {
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// RowError captures a validation or transformation failure on a specific row and column.
type RowError struct {
	ID         int64
	JobID      ID
	RowNumber  int64
	ColumnName string
	RawValue   string
	ErrorCode  string
	Message    string
	CreatedAt  time.Time
}

// ProgressEvent represents a real-time telemetry snapshot pushed to SSE subscribers.
type ProgressEvent struct {
	JobID         ID         `json:"job_id"`
	DatasetID     dataset.ID `json:"dataset_id"`
	Status        Status     `json:"status"`
	ProcessedRows int64      `json:"processed_rows"`
	TotalRows     int64      `json:"total_rows"`
	ValidRows     int64      `json:"valid_rows"`
	ErrorRows     int64      `json:"error_rows"`
	Percentage    float64    `json:"percentage"`
	RowsPerSecond float64    `json:"rows_per_second"`
	Message       string     `json:"message,omitempty"`
	Timestamp     time.Time  `json:"timestamp"`
}
