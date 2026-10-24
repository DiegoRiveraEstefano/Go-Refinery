# Go-Refinery Technical Documentation (`docs/index.md`)

Welcome to the central documentation hub for **Go-Refinery**. Every technical decision, specification, database table, port/service, HTTP view, and asynchronous workflow is indexed below.

---

## Table of Contents

### 1. Architecture & Decisions (`architecture/`)
- [Architecture Index](file:///home/diego/Proyectos/Go-Refinery/docs/architecture/index.md)
  - [ADR-001: Hexagonal Architecture & Standard Library First](file:///home/diego/Proyectos/Go-Refinery/docs/architecture/adr/001-hexagonal-stdlib-architecture.md)
  - [ADR-002: Pure-Go SQLite (`modernc.org/sqlite`), Raw SQL, and Goose Migrations](file:///home/diego/Proyectos/Go-Refinery/docs/architecture/adr/002-sqlite-pure-go-and-goose.md)
  - [ADR-003: Semantic HTML5, Class-Light CSS3, Unpoly, Web Components & SSE](file:///home/diego/Proyectos/Go-Refinery/docs/architecture/adr/003-semantic-frontend-unpoly-sse.md)

### 2. Product & Feature Specifications (`specs/`)
- [Specifications Index](file:///home/diego/Proyectos/Go-Refinery/docs/specs/index.md)

### 3. Development & Operations (`development/`)
- [Development Guides Index](file:///home/diego/Proyectos/Go-Refinery/docs/development/index.md)

### 4. Hexagonal Layer Documentation
- [Database & Domain Models (`dba-models/`)](file:///home/diego/Proyectos/Go-Refinery/docs/dba-models/index.md)
- [Application Services (`services/`)](file:///home/diego/Proyectos/Go-Refinery/docs/services/index.md)
- [Read-Only SQLite Selectors (`selectors/`)](file:///home/diego/Proyectos/Go-Refinery/docs/selectors/index.md)
- [Transformation & Cleaning Filters (`filters/`)](file:///home/diego/Proyectos/Go-Refinery/docs/filters/index.md)
- [HTTP Endpoints, Unpoly & SSE Views (`views/`)](file:///home/diego/Proyectos/Go-Refinery/docs/views/index.md)
- [Asynchronous Workers & Channels (`workers/`)](file:///home/diego/Proyectos/Go-Refinery/docs/workers/index.md)
- [Business & Data Workflows (`workflows/`)](file:///home/diego/Proyectos/Go-Refinery/docs/workflows/index.md)

### 5. Documentation Templates (`_templates/`)
- [`adr-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/adr-template.md)
- [`spec-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/spec-documentation-template.md)
- [`model-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/model-documentation-template.md)
- [`service-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/service-documentation-template.md)
- [`selector-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/selector-documentation-template.md)
- [`view-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/view-documentation-template.md)
- [`workflow-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/workflow-documentation-template.md)
- [`worker-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/worker-documentation-template.md)
