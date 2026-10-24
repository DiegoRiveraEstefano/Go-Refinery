# Documentation Guide & Structure (`DOCS.md`)

**Go-Refinery** uses **GitHub-Flavored Markdown (`.md`)** for technical and architectural documentation under [`docs/`](file:///home/diego/Proyectos/Go-Refinery/docs), paired with strict **Go Doc comments** in the Go source code (compatible with `go doc` and `pkgsite`).

---

## 1. Documentation Structure (`docs/`)

The [`docs/`](file:///home/diego/Proyectos/Go-Refinery/docs) directory at the project root is organized by hexagonal layers, business workflows, and Architecture Decision Records (ADRs):

### Core Entrypoint
- [`docs/index.md`](file:///home/diego/Proyectos/Go-Refinery/docs/index.md): Root table of contents linking every section and document in the repository.

### Subdirectories
- [`docs/architecture/adr/`](file:///home/diego/Proyectos/Go-Refinery/docs/architecture/adr/index.md): Architecture Decision Records numbered sequentially (e.g., `001-hexagonal-stdlib-architecture.md`, `002-sqlite-pure-go-and-goose.md`, `003-semantic-frontend-unpoly-sse.md`).
- [`docs/specs/`](file:///home/diego/Proyectos/Go-Refinery/docs/specs/index.md): Functional and product specifications written before feature implementation.
- [`docs/development/`](file:///home/diego/Proyectos/Go-Refinery/docs/development/index.md): Guides on local setup, running migrations, testing cycles, and benchmarks.
- [`docs/services/`](file:///home/diego/Proyectos/Go-Refinery/docs/services/index.md): Application layer write/orchestration services (`internal/application/`).
- [`docs/selectors/`](file:///home/diego/Proyectos/Go-Refinery/docs/selectors/index.md): Read-only SQLite query optimizations, pagination, and dataset preview queries (`internal/adapters/outbound/sqlite/`).
- [`docs/views/`](file:///home/diego/Proyectos/Go-Refinery/docs/views/index.md): Inbound HTTP handlers (`net/http`), Unpoly fragment endpoints, SSE streams, and HTML templates.
- [`docs/workers/`](file:///home/diego/Proyectos/Go-Refinery/docs/workers/index.md): Asynchronous background worker pools, channels, and batch processors.
- [`docs/filters/`](file:///home/diego/Proyectos/Go-Refinery/docs/filters/index.md): Data cleaning, validation, type casting, and normalization pipeline rules (`internal/domain/pipeline/`).
- [`docs/workflows/`](file:///home/diego/Proyectos/Go-Refinery/docs/workflows/index.md): End-to-end multi-step business flows (upload -> infer schema -> refine -> export).
- [`docs/dba-models/`](file:///home/diego/Proyectos/Go-Refinery/docs/dba-models/index.md): SQLite schema layout, `STRICT` tables, indexes, and Goose migrations.

---

## 2. Document Templates (`docs/_templates/`)

To keep documentation consistent, always start from the templates in [`docs/_templates/`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates):

| Template File | Purpose / When to Use |
| :--- | :--- |
| [`adr-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/adr-template.md) | Technical architecture decision records (ADRs). |
| [`spec-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/spec-documentation-template.md) | New feature and product specifications. |
| [`model-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/model-documentation-template.md) | SQLite table schemas, columns, indexes, and domain structs. |
| [`service-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/service-documentation-template.md) | Core application services (`internal/application/`). |
| [`selector-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/selector-documentation-template.md) | Read-only SQLite queries and filtering/pagination selectors. |
| [`view-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/view-documentation-template.md) | `net/http` endpoints, Unpoly targets, and SSE handlers. |
| [`workflow-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/workflow-documentation-template.md) | Sequential multi-component operations and state machines. |
| [`worker-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/worker-documentation-template.md) | Asynchronous worker pools, job queues, and batching rules. |

---

## 3. Viewing & Serving Documentation Locally

### A. Markdown Documentation (`docs/`)
All documentation files use standard Markdown and Mermaid diagrams, readable directly in GitHub/GitLab or any Markdown viewer.

### B. Go Source Code Documentation (`go doc` / `pkgsite`)
1. Inspect package documentation from the terminal:
   ```bash
   go doc ./internal/...
   ```
2. Launch a local interactive documentation server with `pkgsite`:
   ```bash
   go run golang.org/x/pkgsite/cmd/pkgsite@latest -http=:6060
   ```
   Open `http://localhost:6060` in your browser.

---

## 4. Documentation Guidelines & Best Practices

### A. Avoid Orphan Files
Every `.md` file created inside `docs/` MUST be linked in its section's `index.md` and reachable from [`docs/index.md`](file:///home/diego/Proyectos/Go-Refinery/docs/index.md).

### B. Standard Go Doc Comments
Every exported package, struct, interface, function, and method must include a clear Go Doc comment starting with the exported identifier's name, detailing parameters, return behavior, concurrency guarantees, and error conditions:

```go
// RefineBatch validates and transforms a slice of raw dataset rows according
// to the provided pipeline schema and persists valid records within a single
// SQLite transaction.
//
// Parameters:
//   - ctx: Controls cancellation and deadlines for the batch transaction.
//   - jobID: Unique identifier of the running asynchronous refinement job.
//   - schema: Ordered column definitions and transformation rules.
//   - rows: Slice of raw string slices read from the source stream.
//
// Returns the count of valid persisted rows, the slice of row-level validation
// errors, or a fatal infrastructure error if the SQLite transaction fails.
func (s *RefineryService) RefineBatch(
	ctx context.Context,
	jobID job.ID,
	schema dataset.Schema,
	rows [][]string,
) (int, []job.RowError, error)
```

### C. Boy Scout Rule for Documentation
Documentation must be updated in the same commit as code changes:
1. Modify a service or port signature -> Update the Go Doc comment and [`docs/services/`](file:///home/diego/Proyectos/Go-Refinery/docs/services/index.md).
2. Add a Goose migration or table column -> Update [`docs/dba-models/`](file:///home/diego/Proyectos/Go-Refinery/docs/dba-models/index.md).
3. Change an ingestion or cleaning flow -> Update [`docs/workflows/`](file:///home/diego/Proyectos/Go-Refinery/docs/workflows/index.md).
