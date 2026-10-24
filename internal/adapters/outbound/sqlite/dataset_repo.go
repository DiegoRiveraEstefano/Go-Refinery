package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/diego/go-refinery/internal/domain/dataset"
	"github.com/diego/go-refinery/internal/ports"
)

// Create inserts a new Dataset and all of its inferred ColumnSchemas within a single
// serialized write transaction.
func (s *Store) Create(ctx context.Context, ds *dataset.Dataset) error {
	sampleJSON, err := json.Marshal(ds.SampleRows)
	if err != nil {
		return fmt.Errorf("marshaling sample rows: %w", err)
	}

	delim := ","
	if ds.Delimiter != 0 {
		delim = string(ds.Delimiter)
	}

	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning dataset create tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
		ctx,
		sqlInsertDataset,
		string(ds.ID),
		ds.OriginalName,
		ds.StoragePath,
		string(ds.Format),
		delim,
		boolToInt(ds.HasHeader),
		ds.SizeBytes,
		ds.TotalRows,
		string(ds.Status),
		string(sampleJSON),
		formatTime(ds.CreatedAt),
		formatTime(ds.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("inserting dataset row: %w", err)
	}

	if err := insertColumnsTx(ctx, tx, ds.ID, ds.Schema.Columns); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing dataset create tx: %w", err)
	}
	return nil
}

// GetByID retrieves a Dataset along with its ordered ColumnSchemas from the reader pool.
func (s *Store) GetByID(ctx context.Context, id dataset.ID) (*dataset.Dataset, error) {
	var (
		ds         dataset.Dataset
		rawID      string
		formatStr  string
		delimStr   string
		hasHeader  int
		statusStr  string
		sampleJSON string
		createdStr string
		updatedStr string
	)

	err := s.reader.QueryRowContext(ctx, sqlGetDatasetByID, string(id)).Scan(
		&rawID,
		&ds.OriginalName,
		&ds.StoragePath,
		&formatStr,
		&delimStr,
		&hasHeader,
		&ds.SizeBytes,
		&ds.TotalRows,
		&statusStr,
		&sampleJSON,
		&createdStr,
		&updatedStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, dataset.ErrDatasetNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying dataset %s: %w", id, err)
	}

	ds.ID = dataset.ID(rawID)
	ds.Format = dataset.FileFormat(formatStr)
	if len(delimStr) > 0 {
		ds.Delimiter = rune(delimStr[0])
	} else {
		ds.Delimiter = ','
	}
	ds.HasHeader = intToBool(hasHeader)
	ds.Status = dataset.Status(statusStr)
	ds.CreatedAt = parseTime(createdStr)
	ds.UpdatedAt = parseTime(updatedStr)

	if sampleJSON != "" {
		if err := json.Unmarshal([]byte(sampleJSON), &ds.SampleRows); err != nil {
			return nil, fmt.Errorf("unmarshaling sample rows for dataset %s: %w", id, err)
		}
	}

	cols, err := s.queryColumns(ctx, ds.ID)
	if err != nil {
		return nil, err
	}
	ds.Schema = dataset.Schema{Columns: cols}

	return &ds, nil
}

// List returns a paginated slice of Datasets ordered by creation date descending.
func (s *Store) List(ctx context.Context, filter ports.PageFilter) ([]dataset.Dataset, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := s.reader.QueryContext(ctx, sqlListDatasets, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing datasets: %w", err)
	}
	defer rows.Close()

	var result []dataset.Dataset
	for rows.Next() {
		var (
			ds         dataset.Dataset
			rawID      string
			formatStr  string
			delimStr   string
			hasHeader  int
			statusStr  string
			createdStr string
			updatedStr string
		)
		if err := rows.Scan(
			&rawID,
			&ds.OriginalName,
			&ds.StoragePath,
			&formatStr,
			&delimStr,
			&hasHeader,
			&ds.SizeBytes,
			&ds.TotalRows,
			&statusStr,
			&createdStr,
			&updatedStr,
		); err != nil {
			return nil, fmt.Errorf("scanning dataset row: %w", err)
		}
		ds.ID = dataset.ID(rawID)
		ds.Format = dataset.FileFormat(formatStr)
		if len(delimStr) > 0 {
			ds.Delimiter = rune(delimStr[0])
		}
		ds.HasHeader = intToBool(hasHeader)
		ds.Status = dataset.Status(statusStr)
		ds.CreatedAt = parseTime(createdStr)
		ds.UpdatedAt = parseTime(updatedStr)
		result = append(result, ds)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating dataset rows: %w", err)
	}
	return result, nil
}

// UpdateStatus updates the status and optionally the total row count of a Dataset.
func (s *Store) UpdateStatus(ctx context.Context, id dataset.ID, status dataset.Status, totalRows int64) error {
	nowStr := formatTime(time.Now().UTC())

	res, err := s.writer.ExecContext(ctx, sqlUpdateDatasetStatus, string(status), totalRows, totalRows, nowStr, string(id))
	if err != nil {
		return fmt.Errorf("updating dataset %s status: %w", id, err)
	}

	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return dataset.ErrDatasetNotFound
	}
	return nil
}

// SaveSchema replaces the column schema and pipeline rule specifications for a Dataset
// atomically within a write transaction.
func (s *Store) SaveSchema(ctx context.Context, datasetID dataset.ID, schema dataset.Schema) error {
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning save schema tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	err = tx.QueryRowContext(ctx, sqlDatasetExists, string(datasetID)).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return dataset.ErrDatasetNotFound
	}
	if err != nil {
		return fmt.Errorf("checking dataset existence: %w", err)
	}

	if _, err := tx.ExecContext(ctx, sqlDeleteDatasetColumns, string(datasetID)); err != nil {
		return fmt.Errorf("clearing previous dataset columns: %w", err)
	}

	if err := insertColumnsTx(ctx, tx, datasetID, schema.Columns); err != nil {
		return err
	}

	nowStr := formatTime(time.Now().UTC())
	if _, err := tx.ExecContext(ctx, sqlTouchDatasetUpdatedAt, nowStr, string(datasetID)); err != nil {
		return fmt.Errorf("updating dataset timestamp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing save schema tx: %w", err)
	}
	return nil
}

func insertColumnsTx(ctx context.Context, tx *sql.Tx, datasetID dataset.ID, cols []dataset.ColumnSchema) error {
	if len(cols) == 0 {
		return nil
	}

	stmt, err := tx.PrepareContext(ctx, sqlInsertDatasetColumn)
	if err != nil {
		return fmt.Errorf("preparing column insert statement: %w", err)
	}
	defer stmt.Close()

	for _, col := range cols {
		sampleVals := col.SampleValues
		if sampleVals == nil {
			sampleVals = []string{}
		}
		sampleValsJSON, err := json.Marshal(sampleVals)
		if err != nil {
			return fmt.Errorf("marshaling sample values for column %d: %w", col.Index, err)
		}

		rules := col.RuleSpecs
		if rules == nil {
			rules = []dataset.RuleSpec{}
		}
		rulesJSON, err := json.Marshal(rules)
		if err != nil {
			return fmt.Errorf("marshaling rules for column %d: %w", col.Index, err)
		}

		if _, err := stmt.ExecContext(
			ctx,
			string(datasetID),
			col.Index,
			col.OriginalName,
			col.NormalizedName,
			string(col.InferredType),
			string(col.TargetType),
			boolToInt(col.Nullable),
			col.NullCount,
			col.SampleDistinct,
			string(sampleValsJSON),
			string(rulesJSON),
		); err != nil {
			return fmt.Errorf("inserting column %d (%s): %w", col.Index, col.NormalizedName, err)
		}
	}

	return nil
}

func (s *Store) queryColumns(ctx context.Context, datasetID dataset.ID) ([]dataset.ColumnSchema, error) {
	rows, err := s.reader.QueryContext(ctx, sqlListDatasetColumns, string(datasetID))
	if err != nil {
		return nil, fmt.Errorf("querying columns for dataset %s: %w", datasetID, err)
	}
	defer rows.Close()

	var cols []dataset.ColumnSchema
	for rows.Next() {
		var (
			col            dataset.ColumnSchema
			inferredStr    string
			targetStr      string
			nullableInt    int
			sampleValsJSON string
			rulesJSON      string
		)
		if err := rows.Scan(
			&col.Index,
			&col.OriginalName,
			&col.NormalizedName,
			&inferredStr,
			&targetStr,
			&nullableInt,
			&col.NullCount,
			&col.SampleDistinct,
			&sampleValsJSON,
			&rulesJSON,
		); err != nil {
			return nil, fmt.Errorf("scanning column row: %w", err)
		}
		col.InferredType = dataset.DataType(inferredStr)
		col.TargetType = dataset.DataType(targetStr)
		col.Nullable = intToBool(nullableInt)

		if sampleValsJSON != "" {
			if err := json.Unmarshal([]byte(sampleValsJSON), &col.SampleValues); err != nil {
				return nil, fmt.Errorf("unmarshaling column sample values: %w", err)
			}
		}
		if rulesJSON != "" {
			if err := json.Unmarshal([]byte(rulesJSON), &col.RuleSpecs); err != nil {
				return nil, fmt.Errorf("unmarshaling column rules: %w", err)
			}
		}
		cols = append(cols, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating column rows: %w", err)
	}
	return cols, nil
}
