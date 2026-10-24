# HTTP Endpoints, Unpoly & SSE Views (`docs/views/index.md`)

This directory documents inbound HTTP handlers in `internal/adapters/inbound/http/` and their semantic HTML5 templates. Use [`view-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/view-documentation-template.md).

## Documented Views

- [`HTTP Endpoints, Unpoly Fragments & SSE Stream`](file:///home/diego/Proyectos/Go-Refinery/docs/views/http-endpoints.md)

## Route Map (`net/http`)

| Route Pattern | Handler | Response / Unpoly Target |
| :--- | :--- | :--- |
| `GET /` | `handleRoot` | Full page or `main` fragment with dataset list and `<refinery-dropzone>`. |
| `POST /datasets` | `handleUploadDataset` | Streams multipart upload, runs schema inference, redirects to `/datasets/{id}/schema`. |
| `GET /datasets/{id}/schema` | `handleGetSchema` | Renders `<refinery-schema-editor>` with sample rows and inferred column types. |
| `POST /datasets/{id}/schema` | `handlePostSchema` | Saves column target types and pipeline rules. |
| `POST /datasets/{id}/jobs` | `handleStartJob` | Enqueues async job, returns `202 Accepted` `<refinery-progress>` view. |
| `GET /jobs/{id}/events` | `handleJobSSE` | `text/event-stream` (SSE) pushing throttled progress events via `http.Flusher`. |
| `GET /datasets/{id}/records` | `handleListRecords` | Renders semantic `<table>` of cleaned records and `job_errors` summary. |
