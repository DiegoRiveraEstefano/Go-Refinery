# Model & Table: `[table_name]` / `[domain.StructName]`

- **Domain Package:** `internal/domain/<pkg>`
- **SQLite Adapter:** `internal/adapters/outbound/sqlite`
- **Migration File:** `migrations/0000X_<name>.sql`

---

## 1. Purpose

Describe the domain entity and how it is persisted in SQLite.

## 2. SQLite Table Schema (`STRICT`)

```sql
CREATE TABLE example_table (
    id TEXT PRIMARY KEY,
    created_at TEXT NOT NULL
) STRICT;
```

## 3. Columns & Domain Mapping

| SQL Column | SQLite Type | Go Struct Field | Go Type | Constraints / Notes |
| :--- | :--- | :--- | :--- | :--- |
| `id` | `TEXT` | `ID` | `string` | Primary Key (UUID) |
| `created_at` | `TEXT` | `CreatedAt` | `time.Time` | UTC `RFC3339` |

## 4. Indexes & Query Patterns

- `idx_example_created_at`: Supports chronological listing.
