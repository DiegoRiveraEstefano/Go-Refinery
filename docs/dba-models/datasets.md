# Model & Table: `datasets` & `dataset_columns` / `dataset.Dataset`

- **Domain Package:** `internal/domain/dataset`
- **SQLite Adapter:** [`internal/adapters/outbound/sqlite/dataset_repo.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/sqlite/dataset_repo.go)
- **Migration File:** [`migrations/00001_initial_schema.sql`](file:///home/diego/Proyectos/Go-Refinery/migrations/00001_initial_schema.sql)

---

## 1. Purpose

Stores metadata, sampling snapshots, and ordered column schema definitions (with pipeline rule configurations) for each uploaded file.

## 2. SQLite Table Schema (`STRICT`)

```sql
CREATE TABLE datasets (
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

CREATE TABLE dataset_columns (
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
```

## 3. Repository Methods (`sqlite.DatasetRepo`)

- `Create(ctx, *dataset.Dataset)`: Inserts dataset and all column rows inside a single write transaction (`s.writer`).
- `GetByID(ctx, dataset.ID)`: Reads dataset metadata and ordered columns using `s.reader`.
- `List(ctx, ports.PageFilter)`: Paginated listing ordered by `created_at DESC`.
- `UpdateStatus(ctx, id, status, totalRows)`: Updates lifecycle status and total row count.
- `SaveSchema(ctx, datasetID, schema)`: Atomically replaces `dataset_columns` rows for the dataset in a write transaction.
