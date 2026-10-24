# Background Worker: `[WorkerName]`

- **Package:** `internal/application` / `internal/adapters/outbound/sse`
- **Channel / Queue:** `chan job.ID`

---

## 1. Purpose

Describe the asynchronous task processed by this worker.

## 2. Concurrency & Batching Configuration

- **Worker Count:** Configurable (default: `runtime.NumCPU()`).
- **Batch Size:** `500` rows per SQLite write transaction.
- **SSE Throttle Interval:** `250ms` or every batch completion.

## 3. Cancellation & Graceful Shutdown

How `ctx.Done()` is observed between batches and how partial progress or `CANCELLED` state is saved.
