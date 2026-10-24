# Engineering & Architecture Guidelines (`GUIDELINES.md`)

This document establishes the architectural rules, data streaming patterns, concurrency model, and persistence guidelines for **Go-Refinery**.

---

## 1. Hexagonal Architecture Rules

### 1.1 Dependency Direction
Dependencies always point **inward** toward the domain:

$$\text{Adapters (HTTP / SQLite / Parsers)} \longrightarrow \text{Ports (Interfaces)} \longrightarrow \text{Application Services} \longrightarrow \text{Domain}$$

- **`internal/domain`**:
  - Contains pure business entities (`Dataset`, `ColumnSchema`, `Rule`, `Job`), value objects, and domain errors.
  - **Forbidden imports:** `database/sql`, `net/http`, `os`, `modernc.org/sqlite`, `github.com/xuri/excelize/v2`, `github.com/pressly/goose/v3`.
- **`internal/ports`**:
  - Defines **Inbound Ports** (use cases called by HTTP handlers) and **Outbound Ports** (interfaces implemented by SQLite, filesystem storage, file parsers, and SSE broker).
- **`internal/application`**:
  - Implements Inbound Ports by coordinating domain objects and Outbound Ports.
  - Manages job lifecycles, batch sizing, and error thresholds.
- **`internal/adapters`**:
  - **Inbound (`http`)**: Parses HTTP requests, invokes Inbound Ports, and renders HTML templates or SSE streams.
  - **Outbound (`sqlite`, `parsers`, `storage`, `sse`)**: Implements Outbound Ports using concrete drivers and OS primitives.

---

## 2. Streaming & Bounded-Memory Processing

Go-Refinery must process multi-hundred-megabyte files without loading entire files into RAM.

1. **Never Read Whole Files into Memory:**
   - Forbidden: `os.ReadFile()`, `io.ReadAll()` on uploaded datasets, or `excelize.GetRows()` (which loads the full sheet).
   - Required: `csv.NewReader(r)` row-by-row, `excelize.Rows(sheetName)` iterator, `bufio.Scanner` for JSONL/Logs, and `json.NewDecoder(r)` streaming tokens.
2. **Batch Persistence:**
   - Collect transformed rows in fixed-size slices (default: `500` rows per batch).
   - Flush each batch inside a single SQLite transaction (`BEGIN IMMEDIATE ... COMMIT`) using a prepared statement, then reuse the slice buffer (`batch = batch[:0]`).
3. **Error Tolerance Budget:**
   - A single malformed row must not abort an entire job unless the job's configured `MaxErrorLimit` (e.g., 1,000 errors or 10% error rate) is exceeded.
   - Row-level validation failures are recorded in `job_errors` with row number, column name, raw value, and reason.

---

## 3. Concurrency & Real-Time Progress (SSE)

1. **Worker Pool Lifecycle:**
   - The background worker pool starts in `main.go` with a bounded channel (`chan job.ID`) and a configurable number of workers (`runtime.NumCPU()`).
   - On `SIGINT` / `SIGTERM`, cancel the root `context.Context` and wait for in-flight batches to checkpoint via `sync.WaitGroup` before closing the SQLite connection.
2. **Throttled Progress Broadcasts:**
   - Do not emit an SSE message on every single row (which floods the network and browser DOM).
   - Emit progress events every `N` rows (e.g., every `500` rows) or every `250ms` via `time.Ticker`, plus immediate emission on state transitions (`RUNNING`, `COMPLETED`, `FAILED`).

---

## 4. SQLite Best Practices (No ORM)

1. **Connection Configuration:**
   - Enable `STRICT` tables in SQLite (`CREATE TABLE ... (...) STRICT;`) to enforce type safety at the storage engine level.
   - Apply `PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000; PRAGMA synchronous = NORMAL; PRAGMA foreign_keys = ON;`.
2. **Read vs. Write Pools:**
   - Configure a dedicated **Writer DB handle** with `SetMaxOpenConns(1)` to serialize write transactions cleanly and avoid `SQLITE_BUSY` contention.
   - Configure a **Reader DB handle** with `SetMaxOpenConns(runtime.NumCPU())` for concurrent HTTP queries and UI previews.
3. **Resource Cleanup:**
   - Every `db.QueryContext` call must immediately be followed by `defer rows.Close()` and checked with `if err := rows.Err(); err != nil` after iteration.
   - Every transaction helper must safely `defer tx.Rollback()` before committing.

---

## 5. Testing Strategy

1. **Domain & Pipeline Unit Tests:** Table-driven tests (`t.Run`) in `internal/domain/...` with zero I/O or mocks. Must run in `< 50ms`.
2. **Adapter Integration Tests:** Test SQLite repositories against temporary file databases (`t.TempDir()`) with Goose migrations applied automatically.
3. **Parser Tests:** Verify CSV, XLSX, JSONL, and Log parsers against dirty fixtures (BOM headers, broken quotes, mixed encodings, missing columns).
4. **Race Detection:** Always run `go test -race ./...` in local development and CI.
