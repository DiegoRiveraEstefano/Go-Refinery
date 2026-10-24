# Stream Parsers (`internal/adapters/outbound/parsers`)

- **Outbound Port:** [`ports.StreamParserFactory`](file:///home/diego/Proyectos/Go-Refinery/internal/ports/outbound.go#L54-L56) & [`ports.RowIterator`](file:///home/diego/Proyectos/Go-Refinery/internal/ports/outbound.go#L44-L51)
- **Adapter Package:** `internal/adapters/outbound/parsers`

---

## 1. Purpose

Provides bounded-memory `O(1)` sequential iterators over raw uploaded files across all 5 supported formats (`CSV`, `JSON`, `JSONL`, `LOG`, `XLSX`).

---

## 2. Format Iterators

| Format | File | Streaming Mechanism & Features |
| :--- | :--- | :--- |
| `CSV` / `TSV` | [`factory.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/parsers/factory.go) | `bufio.Reader` + `encoding/csv`. Strips UTF-8 BOM, auto-detects delimiter (`,`, `;`, `\t`, `\|`), enables `LazyQuotes`. |
| `JSON` | [`json_iterator.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/parsers/json_iterator.go) | `json.Decoder` token stream (`Token()` + `More()`). Preserves exact numbers via `UseNumber()` and dynamically appends newly discovered object keys. |
| `JSONL` | [`json_iterator.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/parsers/json_iterator.go) | `bufio.Scanner` (4 MB max line buffer) + per-line `json.Decoder`. Skips blank lines and strips UTF-8 BOM. |
| `LOG` | [`log_iterator.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/parsers/log_iterator.go) | Auto-detects 3 log modes from the first non-empty line: (1) `timestamp / level / message`, (2) `logfmt` (`key=value`), or (3) `raw_line`. |
| `XLSX` | [`xlsx_iterator.go`](file:///home/diego/Proyectos/Go-Refinery/internal/adapters/outbound/parsers/xlsx_iterator.go) | `archive/zip` + `encoding/xml.Decoder` streaming `<row>` elements. Resolves `sharedStrings.xml`, sparse cell references (`A1`, `C2`), inline strings, and booleans while enforcing a `512 MB` decompression cap against Zip Bombs. |
