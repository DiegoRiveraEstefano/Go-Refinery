# Security Policy & Guidelines (`SECURITY.md`)

**Go-Refinery** processes untrusted, potentially malformed, and massive user-supplied files. Security controls are enforced at every architectural boundary.

---

## 1. Threat Model & Mitigations

### 1.1 Unbounded File Uploads & Denial of Service (DoS)
- **Risk:** Memory exhaustion (`OOM`) or disk saturation via multi-gigabyte uploads or **Zip Bombs** inside `.xlsx` archives.
- **Mitigations:**
  - Wrap incoming request bodies with `http.MaxBytesReader(w, r.Body, maxUploadBytes)` (default: `512 MB`).
  - Stream multipart uploads directly to temporary files on disk via `r.MultipartReader()` instead of calling `r.ParseMultipartForm()`, keeping memory usage constant `O(1)`.
  - Enforce decompression ratio and maximum uncompressed byte limits when reading `.xlsx` ZIP entries.
  - Set strict timeouts on `http.Server`: `ReadHeaderTimeout: 5s`, `ReadTimeout: 5m` (for uploads), `IdleTimeout: 60s`.

### 1.2 Path Traversal & Filesystem Isolation
- **Risk:** Malicious filenames (`../../etc/passwd`) overwriting system files.
- **Mitigations:**
  - Never use client-provided filenames on disk. Generate a new random identifier (UUIDv4 / cryptographic hex via `crypto/rand`) for storage paths.
  - Store original filenames strictly as sanitized metadata in SQLite.
  - Enforce restricted permissions (`0600` for files, `0700` for storage directories).

### 1.3 SQL Injection (Raw SQL Without ORM)
- **Risk:** Arbitrary SQL execution through crafted column names, filter values, or file records.
- **Mitigations:**
  - **100% Parameterized Queries:** Always use `?` placeholders with `db.ExecContext`, `db.QueryContext`, and `tx.PrepareContext`.
  - **Identifier Allowlisting:** Dynamic table or column identifiers (if ever required) must be validated against a strict regex (`^[a-z][a-z0-9_]{0,62}$`) and quoted. Never interpolate raw user strings into SQL queries.

### 1.4 Cross-Site Scripting (XSS) & CSV Formula Injection
- **Risk:** Malicious scripts embedded in CSV/JSON cells rendered in the browser, or spreadsheet formulas (`=CMD(...)`, `+`, `-`, `@`) executed when exporting clean files.
- **Mitigations:**
  - Render all HTML solely via Go's contextual auto-escaping `html/template` package. Never cast untrusted data to `template.HTML`.
  - In Web Components, update DOM content using `textContent` instead of `innerHTML`.
  - Prefix exported cells starting with `=`, `+`, `-`, `@`, `\t`, or `\r` with a single quote (`'`) or sanitize them during pipeline normalization.

### 1.5 Regular Expression Denial of Service (ReDoS)
- **Risk:** User-supplied regex rules causing catastrophic backtracking.
- **Mitigations:**
  - Use Go's standard `regexp` package (RE2 engine), which guarantees linear-time `O(n)` execution and is immune to catastrophic backtracking.

---

## 2. HTTP Security Headers

All responses served by `internal/adapters/inbound/http` include:
- `Content-Security-Policy: default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'`
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`

---

## 3. Reporting a Vulnerability

If you discover a security vulnerability in Go-Refinery:
1. **Do not** open a public GitHub issue.
2. Send a detailed report including reproduction steps and affected versions to the project maintainer.
3. You will receive an acknowledgment within 48 hours and a remediation timeline once verified.
