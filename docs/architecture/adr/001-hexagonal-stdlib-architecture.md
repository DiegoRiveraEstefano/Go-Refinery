# ADR-001: Hexagonal Architecture & Standard Library First

- **Status:** Accepted
- **Date:** 2026-10-24
- **Deciders:** Engineering Team
- **Related Packages:** `internal/domain`, `internal/ports`, `internal/application`, `internal/adapters`

---

## 1. Context & Problem Statement

Go-Refinery ingests heterogeneous files (CSV, XLSX, JSONL, logs), infers schemas, applies cleaning pipelines, and reports progress asynchronously. We need a structure that keeps business logic independent of HTTP transport, file formats, and SQLite storage while minimizing external dependencies.

## 2. Considered Options

1. **Hexagonal Architecture (Ports & Adapters) + Go 1.22+ `stdlib` (`net/http`, `encoding/csv`, `encoding/json`, `html/template`, `log/slog`) + `excelize` for `.xlsx`.**
2. **Layered MVC with third-party web framework (Gin/Echo) and third-party task queues.**

## 3. Decision Outcome

Chosen option: **Option 1 (Hexagonal Architecture + Standard Library First)**.

- `internal/domain` has zero external dependencies.
- Go 1.22+ `http.ServeMux` provides method and path-parameter routing natively.
- `github.com/xuri/excelize/v2` is approved strictly inside `internal/adapters/outbound/parsers` for streaming `.xlsx` sheets via `Rows()`.

## 4. Consequences

### Positive
- Fast compilation, tiny dependency surface, and trivial unit testing of domain rules.
- Full protection against framework lock-in.

### Negative / Trade-offs
- Requires writing thin middleware helpers (logging, panic recovery, CSRF/headers) using standard `http.Handler` wrappers.
