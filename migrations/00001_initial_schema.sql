-- +goose Up
CREATE TABLE IF NOT EXISTS datasets (
    id TEXT PRIMARY KEY,
    original_name TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    file_format TEXT NOT NULL,
    delimiter TEXT NOT NULL DEFAULT ',',
    has_header INTEGER NOT NULL DEFAULT 1 CHECK (has_header IN (0, 1)),
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
    total_rows INTEGER NOT NULL DEFAULT 0 CHECK (total_rows >= 0),
    status TEXT NOT NULL DEFAULT 'DRAFT',
    sample_rows_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;

CREATE INDEX IF NOT EXISTS idx_datasets_created_at ON datasets (created_at DESC);

CREATE TABLE IF NOT EXISTS dataset_columns (
    dataset_id TEXT NOT NULL,
    col_index INTEGER NOT NULL CHECK (col_index >= 0),
    original_name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    inferred_type TEXT NOT NULL,
    target_type TEXT NOT NULL,
    nullable INTEGER NOT NULL DEFAULT 1 CHECK (nullable IN (0, 1)),
    null_count INTEGER NOT NULL DEFAULT 0,
    sample_distinct INTEGER NOT NULL DEFAULT 0,
    sample_values_json TEXT NOT NULL DEFAULT '[]',
    rules_json TEXT NOT NULL DEFAULT '[]',
    PRIMARY KEY (dataset_id, col_index),
    FOREIGN KEY (dataset_id) REFERENCES datasets (id) ON DELETE CASCADE
) STRICT;

CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    dataset_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'QUEUED',
    total_rows INTEGER NOT NULL DEFAULT 0,
    processed_rows INTEGER NOT NULL DEFAULT 0,
    valid_rows INTEGER NOT NULL DEFAULT 0,
    error_rows INTEGER NOT NULL DEFAULT 0,
    max_errors INTEGER NOT NULL DEFAULT 1000,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TEXT,
    finished_at TEXT,
    created_at TEXT NOT NULL,
    FOREIGN KEY (dataset_id) REFERENCES datasets (id) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_jobs_dataset_id ON jobs (dataset_id, created_at DESC);

CREATE TABLE IF NOT EXISTS job_errors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL,
    row_number INTEGER NOT NULL,
    column_name TEXT NOT NULL,
    raw_value TEXT NOT NULL,
    error_code TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
) STRICT;

CREATE INDEX IF NOT EXISTS idx_job_errors_job_id ON job_errors (job_id, row_number ASC);

CREATE TABLE IF NOT EXISTS dataset_records (
    dataset_id TEXT NOT NULL,
    row_number INTEGER NOT NULL,
    values_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (dataset_id, row_number),
    FOREIGN KEY (dataset_id) REFERENCES datasets (id) ON DELETE CASCADE
) STRICT;

-- +goose Down
DROP TABLE IF EXISTS dataset_records;
DROP TABLE IF EXISTS job_errors;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS dataset_columns;
DROP TABLE IF EXISTS datasets;
