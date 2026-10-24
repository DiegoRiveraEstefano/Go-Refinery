# Asynchronous Workers & Channels (`docs/workers/index.md`)

This directory documents background goroutine pools and real-time event brokers. Use [`worker-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/worker-documentation-template.md).

## Documented Workers

- [`RefineryWorkerPool`](file:///home/diego/Proyectos/Go-Refinery/docs/workers/refinery-worker-pool.md): Asynchronous stream processor, batch SQLite writer, and SSE telemetry emitter.

## Workers Catalog

| Worker / Component | Package | Description |
| :--- | :--- | :--- |
| [`RefineryWorkerPool`](file:///home/diego/Proyectos/Go-Refinery/docs/workers/refinery-worker-pool.md) | `internal/application` | Bounded worker pool consuming `chan job.ID`, streaming source files, applying pipeline filters, and executing 500-row batch transactions. |
| `SSEBroker` | `internal/adapters/outbound/sse` | Thread-safe in-memory Pub/Sub hub fanning out `job.ProgressEvent` messages to connected HTTP SSE clients. |
