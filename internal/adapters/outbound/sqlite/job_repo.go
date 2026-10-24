package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/ports"
)

// Create satisfies ports.JobRepository by delegating to Store.CreateJob.
func (r *JobRepo) Create(ctx context.Context, j *job.Job) error {
	return r.Store.CreateJob(ctx, j)
}

// GetByID satisfies ports.JobRepository by delegating to Store.GetJobByID.
func (r *JobRepo) GetByID(ctx context.Context, id job.ID) (*job.Job, error) {
	return r.Store.GetJobByID(ctx, id)
}

// CreateJob inserts a new asynchronous Job record into SQLite.
func (s *Store) CreateJob(ctx context.Context, j *job.Job) error {
	_, err := s.writer.ExecContext(
		ctx,
		sqlInsertJob,
		string(j.ID),
		string(j.DatasetID),
		string(j.Status),
		j.TotalRows,
		j.ProcessedRows,
		j.ValidRows,
		j.ErrorRows,
		j.MaxErrors,
		j.ErrorMessage,
		formatNullableTime(j.StartedAt),
		formatNullableTime(j.FinishedAt),
		formatTime(j.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("inserting job %s: %w", j.ID, err)
	}
	return nil
}

// GetJobByID retrieves a Job by its primary key.
func (s *Store) GetJobByID(ctx context.Context, id job.ID) (*job.Job, error) {
	return scanJobRow(s.reader.QueryRowContext(ctx, sqlGetJobByID, string(id)))
}

// GetLatestByDataset retrieves the most recently created Job for a given Dataset.
func (s *Store) GetLatestByDataset(ctx context.Context, datasetID dataset.ID) (*job.Job, error) {
	return scanJobRow(s.reader.QueryRowContext(ctx, sqlGetLatestJob, string(datasetID)))
}

// UpdateProgress updates a Job's state, counters, error message, and start/finish timestamps.
func (s *Store) UpdateProgress(ctx context.Context, j *job.Job) error {
	res, err := s.writer.ExecContext(
		ctx,
		sqlUpdateJobProgress,
		string(j.Status),
		j.TotalRows,
		j.ProcessedRows,
		j.ValidRows,
		j.ErrorRows,
		j.ErrorMessage,
		formatNullableTime(j.StartedAt),
		formatNullableTime(j.FinishedAt),
		string(j.ID),
	)
	if err != nil {
		return fmt.Errorf("updating job %s progress: %w", j.ID, err)
	}

	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return job.ErrJobNotFound
	}
	return nil
}

// ListErrors returns paginated row-level validation errors for a Job ordered by row number.
func (s *Store) ListErrors(ctx context.Context, jobID job.ID, filter ports.PageFilter) ([]job.RowError, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := s.reader.QueryContext(ctx, sqlListJobErrors, string(jobID), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("querying job errors for job %s: %w", jobID, err)
	}
	defer rows.Close()

	var result []job.RowError
	for rows.Next() {
		var (
			re         job.RowError
			rawJobID   string
			createdStr string
		)
		if err := rows.Scan(
			&re.ID,
			&rawJobID,
			&re.RowNumber,
			&re.ColumnName,
			&re.RawValue,
			&re.ErrorCode,
			&re.Message,
			&createdStr,
		); err != nil {
			return nil, fmt.Errorf("scanning job_error row: %w", err)
		}
		re.JobID = job.ID(rawJobID)
		re.CreatedAt = parseTime(createdStr)
		result = append(result, re)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating job_error rows: %w", err)
	}
	return result, nil
}

func scanJobRow(row *sql.Row) (*job.Job, error) {
	var (
		j          job.Job
		rawID      string
		rawDsID    string
		statusStr  string
		startedNS  sql.NullString
		finishedNS sql.NullString
		createdStr string
	)

	err := row.Scan(
		&rawID,
		&rawDsID,
		&statusStr,
		&j.TotalRows,
		&j.ProcessedRows,
		&j.ValidRows,
		&j.ErrorRows,
		&j.MaxErrors,
		&j.ErrorMessage,
		&startedNS,
		&finishedNS,
		&createdStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, job.ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning job row: %w", err)
	}

	j.ID = job.ID(rawID)
	j.DatasetID = dataset.ID(rawDsID)
	j.Status = job.Status(statusStr)
	j.StartedAt = parseNullableTime(startedNS)
	j.FinishedAt = parseNullableTime(finishedNS)
	j.CreatedAt = parseTime(createdStr)

	return &j, nil
}
