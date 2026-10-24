package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/domain/job"
	"github.com/diego/go-refinery/internal/ports"
)

// WriteBatch atomically inserts a batch of validated dataset.Record items and any
// job.RowError items within a single serialized SQLite transaction using prepared statements.
func (s *Store) WriteBatch(
	ctx context.Context,
	jobID job.ID,
	records []dataset.Record,
	rowErrors []job.RowError,
) error {
	if len(records) == 0 && len(rowErrors) == 0 {
		return nil
	}

	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning write batch tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if len(records) > 0 {
		recStmt, err := tx.PrepareContext(ctx, sqlUpsertRecord)
		if err != nil {
			return fmt.Errorf("preparing record batch insert: %w", err)
		}
		defer recStmt.Close()

		for _, rec := range records {
			valuesJSON, err := json.Marshal(rec.Values)
			if err != nil {
				return fmt.Errorf("marshaling record values at row %d: %w", rec.RowNumber, err)
			}
			if _, err := recStmt.ExecContext(
				ctx,
				string(rec.DatasetID),
				rec.RowNumber,
				string(valuesJSON),
				formatTime(rec.CreatedAt),
			); err != nil {
				return fmt.Errorf("inserting record at row %d: %w", rec.RowNumber, err)
			}
		}
	}

	if len(rowErrors) > 0 {
		errStmt, err := tx.PrepareContext(ctx, sqlInsertJobError)
		if err != nil {
			return fmt.Errorf("preparing job_error batch insert: %w", err)
		}
		defer errStmt.Close()

		for _, re := range rowErrors {
			targetJobID := re.JobID
			if targetJobID == "" {
				targetJobID = jobID
			}
			if _, err := errStmt.ExecContext(
				ctx,
				string(targetJobID),
				re.RowNumber,
				re.ColumnName,
				re.RawValue,
				re.ErrorCode,
				re.Message,
				formatTime(re.CreatedAt),
			); err != nil {
				return fmt.Errorf("inserting job_error at row %d: %w", re.RowNumber, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing write batch tx: %w", err)
	}
	return nil
}

// ListRecords retrieves a paginated list of cleaned and normalized records for a Dataset.
func (s *Store) ListRecords(ctx context.Context, datasetID dataset.ID, filter ports.PageFilter) ([]dataset.Record, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := s.reader.QueryContext(ctx, sqlListRecords, string(datasetID), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("querying dataset_records for %s: %w", datasetID, err)
	}
	defer rows.Close()

	var result []dataset.Record
	for rows.Next() {
		var (
			rec        dataset.Record
			rawDsID    string
			valuesJSON string
			createdStr string
		)
		if err := rows.Scan(
			&rawDsID,
			&rec.RowNumber,
			&valuesJSON,
			&createdStr,
		); err != nil {
			return nil, fmt.Errorf("scanning dataset_record row: %w", err)
		}

		rec.DatasetID = dataset.ID(rawDsID)
		rec.CreatedAt = parseTime(createdStr)
		rec.Values = make(map[string]any)
		if valuesJSON != "" {
			if err := json.Unmarshal([]byte(valuesJSON), &rec.Values); err != nil {
				return nil, fmt.Errorf("unmarshaling record values at row %d: %w", rec.RowNumber, err)
			}
		}
		result = append(result, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating dataset_record rows: %w", err)
	}
	return result, nil
}
