# AI Coding Agents Instructions (`AGENTS.md`)

This file defines mandatory rules and context for AI assistants and autonomous coding agents working in **Go-Refinery**.

---

## 1. Non-Negotiable Architectural Guardrails

1. **Architecture:** Strictly **Hexagonal (Ports & Adapters)**.
   - `internal/domain`: Pure Go business logic only. **NEVER** import `net/http`, `database/sql`, `os`, or external packages here.
   - `internal/ports`: Inbound and outbound Go interfaces.
   - `internal/application`: Use-case services orchestrating domain and ports.
   - `internal/adapters`: Concrete HTTP (`net/http`), database (`database/sql` + `modernc.org/sqlite`), parser (`csv`, `excelize`, `json`, `bufio`), filesystem (`os`), and realtime (`sse`) implementations.
2. **Standard Library First:**
   - Use Go 1.22+ `net/http` (`http.NewServeMux()`) for routing (`GET /datasets/{id}`). Do NOT introduce Gin, Echo, Chi, or Fiber.
   - Use `html/template` for server-side HTML rendering.
   - Use `log/slog` for structured logging.
3. **Database & Migrations (No ORM):**
   - Use SQLite via `database/sql` and `modernc.org/sqlite` (Pure Go, `CGO_ENABLED=0`).
   - **NEVER** use GORM, Ent, sqlx, or any ORM/query builder. Write explicit, parameterized SQL queries.
   - Manage schema changes exclusively via sequential SQL files in `migrations/` using `github.com/pressly/goose/v3`.
4. **Streaming & Memory Safety:**
   - Process uploaded files (`CSV`, `XLSX`, `JSONL`, `Logs`) as streams (`io.Reader` / row iterators). Never load entire datasets into memory.
5. **Frontend Constraints:**
   - Semantic **HTML5** (`<main>`, `<section>`, `<table>`, `<fieldset>`, `<dialog>`, `<progress>`). Avoid `<div>` soup.
   - Element-first **CSS3** using CSS custom properties, `.l-*` layout classes, and `.u-*` utility classes. Do NOT introduce Tailwind, Bootstrap, or CSS-in-JS.
   - Use **Unpoly** for hypermedia navigation/fragments and native **Web Components** (`<refinery-*>`) for interactive client state and **SSE** progress consumption.

---

## 2. Mandatory Workflow for Agents

1. **Read Documentation First:**
   - Check [GUIDELINES.md](file:///home/diego/Proyectos/Go-Refinery/GUIDELINES.md), [STYLE_GUIDE.md](file:///home/diego/Proyectos/Go-Refinery/STYLE_GUIDE.md), [DESING.md](file:///home/diego/Proyectos/Go-Refinery/DESING.md), and [DOCS.md](file:///home/diego/Proyectos/Go-Refinery/DOCS.md) before modifying code.
2. **Keep Documentation Synchronized (Boy Scout Rule):**
   - When adding or modifying a domain model/table, update `docs/dba-models/`.
   - When adding or modifying an application service, update `docs/services/`.
   - When adding or modifying a read query/selector, update `docs/selectors/`.
   - When adding or modifying an HTTP handler/view, update `docs/views/`.
   - When adding or modifying an async worker/pipeline, update `docs/workers/` and `docs/workflows/`.
   - Always use the templates in `docs/_templates/` and register new files in their parent `index.md`.
3. **Verify Every Change:**
   - Run `gofmt -s -w .`
   - Run `go vet ./...`
   - Run `go test -race ./...`
