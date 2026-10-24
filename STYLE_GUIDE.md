# Code & Markup Style Guide (`STYLE_GUIDE.md`)

This style guide governs Go code, SQL queries, HTML5 markup, CSS3 stylesheets, Unpoly attributes, and native Web Components in **Go-Refinery**.

---

## 1. Go Style Conventions

We follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).

1. **Formatting & Linting:** All code must pass `gofmt -s` and `go vet ./...` with zero warnings.
2. **Naming:**
   - Package names: short, lowercase, single-word nouns (`dataset`, `pipeline`, `sqlite`, `sse`). Avoid `util`, `common`, or `helpers`.
   - Interfaces: single-method interfaces end in `-er` (`RowReader`, `RuleEvaluator`); multi-method ports describe their role (`DatasetRepository`, `ProgressBroker`).
   - Receivers: 1–2 letter abbreviations consistent across all methods of a type (`s *SchemaService`, `r *SQLiteRepository`).
3. **Context Propagation:** `ctx context.Context` is always the first parameter of any function performing I/O, database queries, or long-running loops.
4. **Error Handling:**
   - Always wrap errors with context using `fmt.Errorf("parsing row %d: %w", rowIdx, err)`.
   - Define sentinel domain errors in `internal/domain` (`var ErrDatasetNotFound = errors.New("dataset not found")`) and inspect them with `errors.Is` / `errors.As`.
   - Never ignore errors with `_` unless explicitly documented why it is safe (e.g., `defer rows.Close()`).
5. **Go Doc Comments:** Every exported type, interface, function, and constant must have a standard Go Doc comment starting with the symbol's name.

---

## 2. SQL & Goose Migration Conventions

1. **Migration Filenames:** Sequential numbered files in `migrations/`:
   `00001_create_datasets.sql`, `00002_create_jobs.sql`.
2. **Keywords & Identifiers:**
   - SQL keywords in `UPPERCASE` (`SELECT`, `FROM`, `WHERE`, `INSERT INTO`, `CREATE TABLE`).
   - Table and column names in `snake_case` (`dataset_id`, `created_at`).
3. **No `SELECT *`:** Always list columns explicitly in `SELECT` and `INSERT` statements so schema changes never silently break `rows.Scan()`.
4. **Timestamps & Booleans in SQLite:**
   - Store UTC timestamps as ISO-8601 strings (`TEXT` in `RFC3339` format) or Unix milliseconds (`INTEGER`).
   - Store booleans as `INTEGER` (`0` or `1`) with `CHECK (col IN (0, 1))`.

```sql
-- +goose Up
CREATE TABLE datasets (
    id TEXT PRIMARY KEY,
    original_name TEXT NOT NULL,
    file_format TEXT NOT NULL,
    size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
    status TEXT NOT NULL DEFAULT 'DRAFT',
    created_at TEXT NOT NULL
) STRICT;

-- +goose Down
DROP TABLE IF EXISTS datasets;
```

---

## 3. HTML5 Style Rules (Semantic & Accessible)

1. **Semantic Elements Over `<div>`:**
   - Use `<header>`, `<nav>`, `<main>`, `<section>`, `<article>`, `<aside>`, `<footer>`.
   - Use `<figure>` + `<figcaption>` for table previews and metrics.
   - Use `<dl>`, `<dt>`, `<dd>` for key-value metadata (file size, row count, format).
   - Use `<progress>` for job progress and `<output>` for live counters.
2. **Forms:** Every `<input>`, `<select>`, and `<textarea>` must be associated with a `<label>` (wrapping or via `for="id"`). Group related controls inside `<fieldset>` with a `<legend>`.
3. **No Inline Styles or Handlers:** Never use `style="..."` or `onclick="..."`.

```html
<section aria-labelledby="upload-heading">
  <h2 id="upload-heading">Upload Raw Dataset</h2>
  <form action="/datasets" method="post" enctype="multipart/form-data" up-submit up-target="main">
    <fieldset>
      <legend>File Source</legend>
      <label for="dataset-file">Select CSV, XLSX, JSON, or Log file</label>
      <input id="dataset-file" name="file" type="file" required accept=".csv,.xlsx,.json,.jsonl,.log,.txt">
    </fieldset>
    <button type="submit">Analyze Schema</button>
  </form>
</section>
```

---

## 4. CSS3 Style Rules (Element-First, Class-Light)

1. **Selector Priority:**
   - Target native elements (`table`, `th`, `td`, `fieldset`, `button`, `dialog`) and HTML attributes (`[aria-busy="true"]`, `button[data-variant="secondary"]`).
   - Use `.l-*` prefix strictly for layout containers (`.l-stack`, `.l-grid`, `.l-cluster`, `.l-sidebar`).
   - Use `.u-*` prefix strictly for single-property utilities (`.u-mono`, `.u-text-right`, `.u-sr-only`).
2. **No Preprocessors or CSS-in-JS:** Use native CSS3 features: Custom Properties (`var(--*)`), CSS Nesting, `:has()`, `clamp()`, Flexbox, and Grid.
3. **Low Specificity:** Never use `#id` selectors in CSS and never use `!important`. Keep selector depth to at most 2–3 levels.

---

## 5. JavaScript, Unpoly & Web Components

1. **ES6+ Modules:** Use native browser ES Modules (`<script type="module" src="/static/js/app.js">`). Zero bundler required.
2. **Unpoly First:** Prefer declarative Unpoly attributes (`up-follow`, `up-submit`, `up-target`, `up-layer="new"`, `up-poll`) over writing custom `fetch()` calls.
3. **Web Components (`CustomElementRegistry`):**
   - Prefix all custom elements with `refinery-` (`<refinery-progress>`, `<refinery-dropzone>`, `<refinery-schema-editor>`).
   - Clean up event listeners and `EventSource` connections inside `disconnectedCallback()`.
   - Communicate with Unpoly by dispatching standard DOM `CustomEvent`s (`bubbles: true`).
