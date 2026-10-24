# Read Selector: `[SelectorName]`

- **Outbound Port:** `ports.[ReaderInterface]` (`internal/ports/outbound.go`)
- **SQLite Implementation:** `internal/adapters/outbound/sqlite/<file>.go`

---

## 1. Purpose

Describe the read-only query, pagination strategy, and UI view that consumes it.

## 2. SQL Query & Execution Plan

```sql
SELECT id, original_name, file_format, status, created_at
FROM datasets
ORDER BY created_at DESC
LIMIT ? OFFSET ?;
```

- **Connection Pool:** Executed on the concurrent **Reader Pool**.
- **Required Indexes:** `idx_datasets_created_at`.

## 3. Performance Constraints

- Zero N+1 queries.
- Bounded `LIMIT` (max 500 rows per page).
