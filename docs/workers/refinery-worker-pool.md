# Background Worker: `RefineryWorkerPool`

- **Package:** [`internal/application/worker_pool.go`](file:///home/diego/Proyectos/Go-Refinery/internal/application/worker_pool.go)
- **Service Owner:** [`application.RefineryService`](file:///home/diego/Proyectos/Go-Refinery/internal/application/refinery_service.go)
- **Channel / Queue:** `chan job.ID` (default buffer: `64`)

---

## 1. Purpose

Processes queued dataset refinement jobs asynchronously: opens the raw uploaded file as an `O(1)` memory stream, compiles and applies per-column cleaning/validation stages (`pipeline.ColumnPipeline`), writes cleaned records and row-level errors in atomic SQLite batches, and broadcasts throttled progress snapshots to `ports.ProgressBroker` (SSE).

---

## 2. Concurrency & Batching Configuration

- **Worker Count:** Configurable via `-workers` CLI flag or `StartWorkers(ctx, n)` (default: `runtime.NumCPU()`).
- **Batch Size:** `500` rows per SQLite write transaction (`SetBatchSize(size)` overrides for testing/tuning).
- **SSE Throttle Interval:** `250ms` between intermediate batch progress events, plus immediate events on state transitions (`QUEUED`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`).
- **Error Budget Enforcement:** Aborts the job with `job.ErrMaxErrorsReached` if `ErrorRows >= MaxErrors` (default `1,000`), flushing any pending errors before marking the job `FAILED`.

---

## 3. Cancellation & Graceful Shutdown

1. Each worker loop selects on `ctx.Done()` between jobs and before reading each row.
2. When `SIGINT` / `SIGTERM` cancels the root context:
   - In-flight workers flush their current partial batch to SQLite.
   - The interrupted job transitions to `CANCELLED` using a short bounded cleanup context (`3s`).
   - `refinerySvc.Wait()` blocks `main.go` until `sync.WaitGroup` reaches zero before closing the SQLite connection pools.
