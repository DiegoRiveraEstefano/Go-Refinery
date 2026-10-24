# Read-Only SQLite Selectors (`docs/selectors/index.md`)

This directory documents optimized read-only SQLite queries executed against the concurrent **Reader Pool** in `internal/adapters/outbound/sqlite/`. Use [`selector-documentation-template.md`](file:///home/diego/Proyectos/Go-Refinery/docs/_templates/selector-documentation-template.md).

## Selectors Catalog

| Selector | Purpose |
| :--- | :--- |
| `ListDatasetsSelector` | Paginated listing of uploaded datasets with latest job status. |
| `GetDatasetSchemaPreviewSelector` | Fetches inferred column definitions and sample rows for the schema editor view. |
| `ListJobErrorsSelector` | Paginated query over `job_errors` filtered by column or error code. |
| `QueryRefinedRecordsSelector` | Cursor-paginated reader for previewing and exporting cleaned records. |
