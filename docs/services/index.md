# Application Services & Outbound Adapters (`docs/services/index.md`)

This directory documents the application services in `internal/application/`, cross-cutting observability in `internal/logger/`, and core outbound adapters. Use [`service-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/service-documentation-template.md).

## Documented Components

- [`StreamParserFactory` & Iterators (`CSV`, `JSON`, `JSONL`, `LOG`, `XLSX`)](file:///home/diego/Proyectos/Go-Refinery/docs/services/stream-parsers.md)

## Services Catalog

| Service | Package / File | Responsibility |
| :--- | :--- | :--- |
| `IngestService` | [`internal/application/ingest_service.go`](file:///home/diego/Proyectos/Go-Refinery/internal/application/ingest_service.go) | Streams uploaded files to storage, infers initial schema, and registers `Dataset` metadata. |
| `SchemaService` | [`internal/application/schema_service.go`](file:///home/diego/Proyectos/Go-Refinery/internal/application/schema_service.go) | Samples the first $N$ rows via stream parsers, infers column types/metrics, and persists user schema updates. |
| `RefineryService` | [`internal/application/refinery_service.go`](file:///home/diego/Proyectos/Go-Refinery/internal/application/refinery_service.go) | Enqueues and executes asynchronous cleaning jobs via `WorkerPool`, batching writes to SQLite and publishing SSE events. |
| `ExportRecords` | [`internal/application/export_service.go`](file:///home/diego/Proyectos/Go-Refinery/internal/application/export_service.go) | Streams cleaned records in paginated chunks (500 rows/page) into `CSV`, `TSV`, `JSON`, or `JSONL` with spreadsheet formula injection sanitization. |
| `Logger` | [`internal/logger/logger.go`](file:///home/diego/Proyectos/Go-Refinery/internal/logger/logger.go) | Thread-safe structured logger (`DEBUG`, `INFO`, `WARN`, `ERROR`) supporting `text` (`key=value`) and `json` formats, component scoping, and context `request_id` propagation. |
