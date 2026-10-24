# Application Services & Outbound Adapters (`docs/services/index.md`)

This directory documents the application services in `internal/application/` and core outbound adapters. Use [`service-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/service-documentation-template.md).

## Documented Components

- [`StreamParserFactory` & Iterators (`CSV`, `JSON`, `JSONL`, `LOG`, `XLSX`)](file:///home/diego/Proyectos/Go-Refinery/docs/services/stream-parsers.md)

## Services Catalog

| Service | Package / File | Responsibility |
| :--- | :--- | :--- |
| `IngestService` | `internal/application/ingest_service.go` | Streams uploaded files to storage and registers initial `Dataset` metadata. |
| `SchemaService` | `internal/application/schema_service.go` | Samples the first $N$ rows via stream parsers and infers column types and quality metrics. |
| `RefineryService` | `internal/application/refinery_service.go` | Enqueues and executes asynchronous cleaning jobs, batching writes to SQLite and publishing SSE events. |
