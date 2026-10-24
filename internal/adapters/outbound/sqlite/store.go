// Package sqlite implements the outbound persistence ports using database/sql
// and embedded parameterized SQLite query files (queries/*.sql) without any ORM.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/diego/go-refinery/internal/ports"
)

// Compile-time assertions that Store and its repository views satisfy all SQLite outbound ports.
var (
	_ ports.DatasetRepository = (*DatasetRepo)(nil)
	_ ports.JobRepository     = (*JobRepo)(nil)
	_ ports.RecordBatchWriter = (*RecordRepo)(nil)
)

// Store holds separated SQLite connection pools for concurrent reads and serialized writes.
type Store struct {
	writer *sql.DB
	reader *sql.DB
}

// DatasetRepo adapts Store to the ports.DatasetRepository interface.
type DatasetRepo struct{ *Store }

// JobRepo adapts Store to the ports.JobRepository interface.
type JobRepo struct{ *Store }

// RecordRepo adapts Store to the ports.RecordBatchWriter interface.
type RecordRepo struct{ *Store }

// Datasets returns the ports.DatasetRepository implementation backed by Store.
func (s *Store) Datasets() *DatasetRepo { return &DatasetRepo{Store: s} }

// Jobs returns the ports.JobRepository implementation backed by Store.
func (s *Store) Jobs() *JobRepo { return &JobRepo{Store: s} }

// Records returns the ports.RecordBatchWriter implementation backed by Store.
func (s *Store) Records() *RecordRepo { return &RecordRepo{Store: s} }

// NewStore wraps a configured writer (*sql.DB with MaxOpenConns=1) and reader pool.
func NewStore(writer, reader *sql.DB) *Store {
	return &Store{
		writer: writer,
		reader: reader,
	}
}

// OpenStore opens a SQLite database file using the given driverName (default "sqlite"),
// configures separated Writer (MaxOpenConns=1) and Reader pools,
// and applies WAL and foreign_keys PRAGMAs loaded from queries/pragmas.sql.
func OpenStore(ctx context.Context, driverName, dsn string) (*Store, error) {
	writer, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("opening writer db: %w", err)
	}
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)
	writer.SetConnMaxLifetime(0)

	if err := applyPragmas(ctx, writer); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("applying writer pragmas: %w", err)
	}

	reader, err := sql.Open(driverName, dsn)
	if err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("opening reader db: %w", err)
	}
	maxReaders := runtime.NumCPU()
	if maxReaders < 2 {
		maxReaders = 2
	}
	reader.SetMaxOpenConns(maxReaders)
	reader.SetMaxIdleConns(maxReaders)

	if err := applyPragmas(ctx, reader); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return nil, fmt.Errorf("applying reader pragmas: %w", err)
	}

	return NewStore(writer, reader), nil
}

// ExecSchema executes DDL statements (such as Goose SQL migrations) on the writer pool.
func (s *Store) ExecSchema(ctx context.Context, ddl string) error {
	clean := extractUpMigration(ddl)
	if _, err := s.writer.ExecContext(ctx, clean); err != nil {
		return fmt.Errorf("executing schema ddl: %w", err)
	}
	return nil
}

// WriterDB returns the underlying serialized writer *sql.DB (used by Goose migrations).
func (s *Store) WriterDB() *sql.DB {
	return s.writer
}

// ReaderDB returns the underlying concurrent reader *sql.DB.
func (s *Store) ReaderDB() *sql.DB {
	return s.reader
}

// Ping verifies database connectivity on both pools.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.writer.PingContext(ctx); err != nil {
		return err
	}
	return s.reader.PingContext(ctx)
}

// Close closes both reader and writer database pools.
func (s *Store) Close() error {
	errRead := s.reader.Close()
	errWrite := s.writer.Close()
	if errWrite != nil {
		return errWrite
	}
	return errRead
}

func applyPragmas(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, sqlPragmas); err != nil {
		return err
	}
	return nil
}

func extractUpMigration(sqlContent string) string {
	const downMarker = "-- +goose Down"
	if idx := strings.Index(sqlContent, downMarker); idx != -1 {
		return sqlContent[:idx]
	}
	return sqlContent
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(v int) bool {
	return v != 0
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC()
	}
	return time.Time{}
}

func formatNullableTime(t *time.Time) sql.NullString {
	if t == nil || t.IsZero() {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{
		String: t.UTC().Format(time.RFC3339Nano),
		Valid:  true,
	}
}

func parseNullableTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t := parseTime(ns.String)
	if t.IsZero() {
		return nil
	}
	return &t
}
