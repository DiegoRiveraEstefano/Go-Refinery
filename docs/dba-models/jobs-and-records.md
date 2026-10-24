# Model & Table: `jobs`, `job_errors`, & `dataset_records`

- **Domain Packages:** `internal/domain/job`, `internal/domain/dataset`
- **SQLite Adapters:**
  - [`internal/adapters/outbound/sqlite/job_repo.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/job_repo.go)
  - [`internal/adapters/outbound/sqlite/record_repo.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/record_repo.go)
- **Migration File:** [`migrations/00001_initial_schema.sql`](file:///home/diego/Proyectos/Go-Refinery/migrations/00001_initial_schema.sql)

---

## 1. Purpose

Tracks asynchronous refinement jobs, logs row-level validation failures, and stores normalized output records in 500-row atomic batches.

## 2. SQLite Table Schema (`STRICT`)

```sql
CREATE TABLE jobs (
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

CREATE TABLE job_errors (
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

CREATE TABLE dataset_records (
    dataset_id TEXT NOT NULL,
    row_number INTEGER NOT NULL,
    values_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (dataset_id, row_number),
    FOREIGN KEY (dataset_id) REFERENCES datasets (id) ON DELETE CASCADE
) STRICT;
```

## 3. Repository Methods (`sqlite.JobRepo` & `sqlite.RecordRepo`)

- `Jobs().Create(ctx, *job.Job)` / `Jobs().UpdateProgress(ctx, *job.Job)`: Persists job state transitions and counters on `s.writer`.
- `Jobs().GetByID(ctx, id)` / `Jobs().GetLatestByDataset(ctx, datasetID)`: Reads job status from `s.reader`.
- `Jobs().ListErrors(ctx, jobID, filter)`: Paginated query over `job_errors` using `idx_job_errors_job_id`.
- `Records().WriteBatch(ctx, jobID, records, rowErrors)`: Prepared-statement batch insert of cleaned records and row errors inside a single transaction.
- `Records().ListRecords(ctx, datasetID, filter)`: Paginated reader over `dataset_records`.
