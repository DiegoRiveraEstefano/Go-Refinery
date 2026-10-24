# Changelog (`CHANGELOG.md`)

All notable changes to **Go-Refinery** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Defined **Hexagonal Architecture (Ports & Adapters)** layout (`internal/domain`, `internal/ports`, `internal/application`, `internal/adapters`).
- Selected core minimal dependency stack:
  - Go 1.22+ standard library (`net/http`, `database/sql`, `html/template`, `encoding/csv`, `encoding/json`, `log/slog`).
  - `modernc.org/sqlite` (Pure-Go SQLite driver, zero CGO, zero ORM).
  - `github.com/pressly/goose/v3` for SQL schema migrations.
  - `github.com/xuri/excelize/v2` for streaming `.xlsx` ingestion.
- Established frontend architecture using semantic **HTML5**, element-first **CSS3** with `.l-*` / `.u-*` classes, **Unpoly**, and native **Web Components** with **Server-Sent Events (SSE)** for real-time job progress.
- Created baseline governance and technical documentation (`README.md`, `DESING.md`, `STYLE_GUIDE.md`, `GUIDELINES.md`, `SECURITY.md`, `CONTRIBUTING.md`, `AGENTS.md`, `DOCS.md`, and `docs/` hierarchy with templates).
