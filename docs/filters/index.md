# Transformation & Cleaning Filters (`docs/filters/index.md`)

This directory documents the modular pipeline rules inside `internal/domain/pipeline/`.

## Package Organization

| File | Responsibility |
| :--- | :--- |
| [`pipeline.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/pipeline/pipeline.go) | Core `Stage` interface, `ColumnPipeline.Execute`, sentinel errors, and target `DataType` casting. |
| [`registry.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/pipeline/registry.go) | `AvailableRules()` catalog with `SupportedTypes` metadata for reactive UI filtering and `BuildStage` factory. |
| [`text_stages.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/pipeline/text_stages.go) | Text sanitization, casing, HTML stripping, digit extraction, regex replacement, and CSV formula escaping. |
| [`numeric_stages.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/pipeline/numeric_stages.go) | Currency stripping, absolute value, decimal rounding, and `>= 0` / `> 0` range validation. |
| [`validation_stages.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/pipeline/validation_stages.go) | Null handling (`null_if_empty`, `drop_row_if_null`, defaults), RFC-5322 email validation/masking, UUID check, ISO date normalization, and JSON compaction. |

---

## Built-In Filter Catalog by Column Type

| Rule ID | Label | Supported Column Types | Description |
| :--- | :--- | :--- | :--- |
| `trim_space` | Trim Whitespace | All (`STRING`, `INTEGER`, `FLOAT`, `BOOLEAN`, `DATETIME`, `EMAIL`, `UUID`, `JSON`) | Strips leading and trailing spaces. |
| `null_if_empty` | Normalize Nulls | All | Maps `'null'`, `'N/A'`, `'-'`, `'none'` to empty/NULL. |
| `drop_row_if_null` | Drop Row if Null | All | Discards the entire row if the cell is empty or null. |
| `collapse_whitespace` | Collapse Spaces | `STRING` | Collapses consecutive spaces/tabs into a single space. |
| `to_lower` | Lowercase | `STRING`, `EMAIL`, `UUID` | Converts characters to lowercase. |
| `to_upper` | Uppercase | `STRING` | Converts characters to UPPERCASE. |
| `to_title` | Title Case | `STRING` | Capitalizes the first letter of each word. |
| `strip_html` | Strip HTML Tags | `STRING` | Removes inline `<...>` HTML/XML tags. |
| `remove_non_printable` | Strip Non-Printable | `STRING` | Removes control characters and non-printable Unicode runes. |
| `extract_digits` | Extract Digits Only | `STRING`, `INTEGER` | Strips all non-numeric characters (`0-9`). |
| `sanitize_csv_formula` | Escape CSV Formulas | `STRING` | Prefixes `=`, `+`, `-`, `@` with `'` to prevent spreadsheet formula injection. |
| `strip_currency` | Strip Currency & Commas | `INTEGER`, `FLOAT` | Cleans `$`, `,`, and currency codes before numeric parsing. |
| `default_zero_if_null` | Default 0 if Null | `INTEGER`, `FLOAT` | Replaces empty/null cells with `0`. |
| `abs_number` | Absolute Value (`\|x\|`) | `INTEGER`, `FLOAT` | Converts negative numbers to positive. |
| `non_negative` | Require `>= 0` | `INTEGER`, `FLOAT` | Rejects negative numbers with `ErrValidationCheck`. |
| `positive_only` | Require `> 0` | `INTEGER`, `FLOAT` | Rejects zero or negative numbers with `ErrValidationCheck`. |
| `round_2_decimals` | Round (2 Decimals) | `FLOAT` | Rounds floating-point numbers to 2 decimal places. |
| `round_integer` | Round to Nearest Int | `INTEGER`, `FLOAT` | Rounds decimal values to the nearest integer. |
| `default_false_if_null` | Default false if Null | `BOOLEAN` | Replaces empty/null cells with `false`. |
| `normalize_iso_date` | Format `YYYY-MM-DD` | `DATETIME` | Normalizes parsed dates to ISO calendar date `YYYY-MM-DD`. |
| `validate_email` | Strict RFC-5322 Email | `EMAIL` | Validates email syntax and lowercases the address. |
| `mask_email_user` | Mask Email Username | `EMAIL`, `STRING` | Masks local part for privacy (`a***@domain.com`). |
| `validate_uuid` | Strict UUID Format | `UUID` | Validates 36-char UUID and normalizes to lowercase. |
| `compact_json` | Validate & Compact JSON | `JSON` | Validates JSON syntax and minifies whitespace. |
