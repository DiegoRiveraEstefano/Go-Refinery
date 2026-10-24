# Database & Domain Models (`docs/dba-models/index.md`)

This directory documents SQLite `STRICT` tables ([`migrations/`](file:///home/diego/Proyectos/Go-Refinery/migrations/embed.go)), embedded raw SQL queries ([`internal/adapters/outbound/sqlite/queries/`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries.go)), and their mapping to Go domain entities (`internal/domain/`). Use [`model-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/model-documentation-template.md).

## Documented Models & Repositories

- [`datasets` & `dataset_columns`](file:///home/diego/Proyectos/Go-Refinery/docs/dba-models/datasets.md): Dataset metadata, sample snapshots, and column schema persistence (`sqlite.DatasetRepo`).
- [`jobs`, `job_errors`, & `dataset_records`](file:///home/diego/Proyectos/Go-Refinery/docs/dba-models/jobs-and-records.md): Asynchronous job tracking, row-level validation errors, and batch record persistence (`sqlite.JobRepo` & `sqlite.RecordRepo`).

## Embedded SQL Query Files (`internal/adapters/outbound/sqlite/queries/`)

All SQL statements are maintained as standalone `.sql` files and embedded into the binary via `//go:embed queries/*.sql` in [`queries.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries.go):

| Query File | Repository / Component | Purpose |
| :--- | :--- | :--- |
| [`pragmas.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/pragmas.sql) | `sqlite.OpenStore` | Configures `WAL` mode, `busy_timeout = 5000`, `synchronous = NORMAL`, and `foreign_keys = ON`. |
| [`datasets_insert.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/datasets_insert.sql) | `sqlite.DatasetRepo` | Inserts a new `datasets` row. |
| [`datasets_get_by_id.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/datasets_get_by_id.sql) | `sqlite.DatasetRepo` | Fetches a single dataset by ID. |
| [`datasets_list.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/datasets_list.sql) | `sqlite.DatasetRepo` | Paginated listing of datasets ordered by `created_at DESC`. |
| [`datasets_update_status.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/datasets_update_status.sql) | `sqlite.DatasetRepo` | Updates dataset status and optional total row count. |
| [`datasets_exists.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/datasets_exists.sql) | `sqlite.DatasetRepo` | Checks dataset existence prior to schema replacement. |
| [`datasets_touch_updated_at.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/datasets_touch_updated_at.sql) | `sqlite.DatasetRepo` | Updates `updated_at` timestamp when saving schema changes. |
| [`columns_delete_by_dataset.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/columns_delete_by_dataset.sql) | `sqlite.DatasetRepo` | Clears previous columns before re-inserting updated schema. |
| [`columns_insert.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/columns_insert.sql) | `sqlite.DatasetRepo` | Prepared statement for inserting `dataset_columns` rows. |
| [`columns_list_by_dataset.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/columns_list_by_dataset.sql) | `sqlite.DatasetRepo` | Fetches ordered column schemas for a dataset. |
| [`jobs_insert.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/jobs_insert.sql) | `sqlite.JobRepo` | Inserts a new `jobs` row. |
| [`jobs_get_by_id.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/jobs_get_by_id.sql) | `sqlite.JobRepo` | Fetches a job by ID. |
| [`jobs_get_latest_by_dataset.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/jobs_get_latest_by_dataset.sql) | `sqlite.JobRepo` | Fetches the most recent job for a dataset. |
| [`jobs_update_progress.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/jobs_update_progress.sql) | `sqlite.JobRepo` | Updates job status, row counters, and timestamps. |
| [`job_errors_insert.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/job_errors_insert.sql) | `sqlite.RecordRepo` | Prepared statement for batch inserting `job_errors`. |
| [`job_errors_list_by_job.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/job_errors_list_by_job.sql) | `sqlite.JobRepo` | Paginated query of row errors for a job. |
| [`records_upsert.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/records_upsert.sql) | `sqlite.RecordRepo` | Prepared statement for batch inserting `dataset_records`. |
| [`records_list_by_dataset.sql`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/queries/records_list_by_dataset.sql) | `sqlite.RecordRepo` | Paginated query of refined records for a dataset. |
