# ADR-002: Pure-Go SQLite (`modernc.org/sqlite`), Raw SQL, and Goose Migrations

- **Status:** Accepted
- **Date:** 2026-10-24
- **Deciders:** Engineering Team
- **Related Packages:** `internal/adapters/outbound/sqlite`, `migrations/`

---

## 1. Context & Problem Statement

Go-Refinery requires an embedded, zero-configuration relational database capable of concurrent reads and high-speed batch writes without using an ORM or requiring C compilers (`CGO`) during builds.

## 2. Considered Options

1. **`database/sql` + `modernc.org/sqlite` (Pure Go) + Raw SQL + `pressly/goose/v3`.**
2. **`mattn/go-sqlite3` (CGO) + GORM / sqlx.**

## 3. Decision Outcome

Chosen option: **Option 1 (`database/sql` + `modernc.org/sqlite` + Raw SQL + `pressly/goose/v3`)**.

- All SQL queries are explicit, parameterized, and use SQLite `STRICT` tables in `WAL` mode.
- Separate Reader (`MaxOpenConns = N`) and Writer (`MaxOpenConns = 1`) pools prevent `SQLITE_BUSY` errors during batch ingestion.
- Schema evolution is managed via sequential SQL files in `migrations/` executed by `goose`.

## 4. Consequences

### Positive
- `CGO_ENABLED=0` static binaries across Linux, macOS, and Windows.
- Predictable SQL execution plans and zero ORM overhead during 500-row batch inserts.

### Negative / Trade-offs
- Manual `rows.Scan()` mapping in outbound repository adapters.
