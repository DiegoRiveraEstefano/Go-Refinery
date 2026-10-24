# Transformation & Cleaning Filters (`docs/filters/index.md`)

This directory documents pure domain rules and filters inside `internal/domain/pipeline/`.

## Built-In Filter Stages

| Category | Filter Name | Description |
| :--- | :--- | :--- |
| **Sanitization** | `TrimSpace`, `CollapseWhitespace`, `NormalizeUnicode` | Removes leading/trailing spaces, normalizes UTF-8 NFC, strips non-printable chars. |
| **Case & Format** | `ToLower`, `ToUpper`, `RegexReplace`, `RegexExtract` | Standardizes casing and extracts structured substrings using RE2 regular expressions. |
| **Null Handling** | `NullIfEmpty`, `DefaultIfNull`, `DropRowIfNull` | Maps sentinel empty values (`N/A`, `null`, `-`) to `NULL` or default values, or rejects the row. |
| **Type Casting & Validation** | `CastInteger`, `CastFloat`, `CastBoolean`, `ParseDate`, `ValidateEmail`, `RangeCheck` | Converts raw strings into strongly typed values and validates domain constraints. |
