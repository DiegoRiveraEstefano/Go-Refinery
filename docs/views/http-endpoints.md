# HTTP Endpoints, Unpoly & SSE Views (`internal/adapters/inbound/http`)

- **Inbound Adapter Package:** [`internal/adapters/inbound/http`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/inbound/http/server.go)
- **Embedded Assets (`embed.FS`):** [`web/embed.go`](file:///home/diego/Proyectos/Go-Refinery/web/embed.go)
- **HTML5 Templates:**
  - [`web/templates/layout.html`](file:///home/diego/Proyectos/Go-Refinery/web/templates/layout.html)
  - [`web/templates/dashboard.html`](file:///home/diego/Proyectos/Go-Refinery/web/templates/dashboard.html)
  - [`web/templates/schema.html`](file:///home/diego/Proyectos/Go-Refinery/web/templates/schema.html)
  - [`web/templates/progress.html`](file:///home/diego/Proyectos/Go-Refinery/web/templates/progress.html)
  - [`web/templates/records.html`](file:///home/diego/Proyectos/Go-Refinery/web/templates/records.html)

---

## 1. Modular Router Architecture

The HTTP inbound adapter is split into dedicated router/view files to keep handlers focused:

| File | Responsibility |
| :--- | :--- |
| [`server.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/inbound/http/server.go) | Server struct, configuration (`SetMaxUploadBytes`, `SetLogger`), and top-level route registration (`web.StaticFS`, dashboard, datasets, jobs). |
| [`middleware.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/inbound/http/middleware.go) | Request correlation (`X-Request-ID`), structured HTTP access logging, panic recovery, `http.Flusher` delegation, and security headers (`Content-Security-Policy`, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`). |
| [`templates.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/inbound/http/templates.go) | Parses standalone HTML files from `web.TemplatesFS` (`embed.FS`) with `html/template`. |
| [`dashboard_router.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/inbound/http/dashboard_router.go) | Handles `GET /`, `GET /healthz`, and the main upload/datasets dashboard view. |
| [`datasets_router.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/inbound/http/datasets_router.go) | Handles `POST /datasets` (streaming multipart upload), `GET /datasets/{id}/records` (cleaned output, search/filter engine & row errors), and `GET /datasets/{id}/export` (multi-format download). |
| [`schema_router.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/inbound/http/schema_router.go) | Handles `GET /datasets/{id}/schema` and `POST /datasets/{id}/schema` for configuring column types and cleaning rules. |
| [`jobs_router.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/inbound/http/jobs_router.go) | Handles `POST /datasets/{id}/jobs`, `GET /jobs/{id}`, and `GET /jobs/{id}/events` (SSE stream via `http.Flusher`). |

---

## 2. Implemented Endpoints

| Method & Path | Router File | Description |
| :--- | :--- | :--- |
| `GET /healthz` | `dashboard_router.go` | Liveness check (`200 OK`). |
| `GET /` / `GET /datasets` | `dashboard_router.go` | Renders `<refinery-dropzone>` and the datasets table (`dashboard.html`). |
| `GET /static/*` | `server.go` | Serves embedded CSS and JS assets directly from `web.StaticFS`. |
| `POST /datasets` | `datasets_router.go` | Streams `multipart/form-data` via `r.MultipartReader()`, infers schema, and redirects (`303 See Other` + `X-Up-Location`) to `/datasets/{id}/schema`. |
| `GET /datasets/{id}/schema` | `schema_router.go` | Renders `<refinery-schema-editor>` (`schema.html`). |
| `POST /datasets/{id}/schema` | `schema_router.go` | Saves target column types, nullability, and cleaning stages. |
| `POST /datasets/{id}/jobs` | `jobs_router.go` | Enqueues an asynchronous refinement job and returns `202 Accepted` (`progress.html`). |
| `GET /jobs/{id}` | `jobs_router.go` | Renders the current status and `<refinery-progress>` widget for a job. |
| `GET /jobs/{id}/events` | `jobs_router.go` | Streams `event: progress` JSON snapshots over `text/event-stream` using `http.Flusher`. |
| `GET /datasets/{id}/records` | `datasets_router.go` | Renders `<refinery-search-box>` (supporting free text, `/regex/i`, mini-DSL, Mongo JSON, and field/operator/value filters), cleaned `dataset_records`, `job_errors`, and the export toolbar (`records.html`). |
| `GET /datasets/{id}/export?format={csv\|tsv\|json\|jsonl}` | `datasets_router.go` | Streams all cleaned records (or the active search-filtered subset) as a downloadable file (`Content-Disposition: attachment`) in CSV, TSV, JSON, or JSONL format. |
