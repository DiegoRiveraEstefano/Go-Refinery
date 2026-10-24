# Contributing to Go-Refinery (`CONTRIBUTING.md`)

Thank you for contributing to **Go-Refinery**. Follow these rules to keep the codebase lean, maintainable, and aligned with our architectural principles.

---

## 1. Core Constraints (Read Before Coding)

1. **Zero Unauthorized Dependencies:** Do not add packages to `go.mod` beyond `modernc.org/sqlite`, `github.com/pressly/goose/v3`, and `github.com/xuri/excelize/v2` without a prior approved ADR in `docs/architecture/adr/`.
2. **No ORMs:** Write explicit, parameterized SQL using `database/sql`.
3. **Respect the Hexagon:** Never import `net/http`, `database/sql`, or third-party libraries inside `internal/domain`.
4. **Boy Scout Rule for Docs:** Any change to an architectural decision, database table, application service, or HTTP endpoint must update its corresponding Markdown file in `docs/` in the same commit.

---

## 2. Development Workflow

1. **Branch Naming:**
   - `feat/<short-slug>` — New capability or pipeline rule
   - `fix/<short-slug>` — Bug fix
   - `docs/<short-slug>` — Documentation update
   - `refactor/<short-slug>` — Structural improvement without behavior change
2. **Conventional Commits:**
   Format commit messages as `<type>(<scope>): <concise summary in imperative mood>`:
   - `feat(pipeline): add unicode normalization and regex extract rules`
   - `fix(sqlite): prevent SQLITE_BUSY by serializing batch writer pool`
   - `docs(adr): record decision to use modernc.org/sqlite without CGO`

---

## 3. Pre-Submit Verification Checklist

Before opening a Pull Request, run the following checks locally:

```bash
# 1. Format and static analysis
gofmt -s -w .
go vet ./...

# 2. Run all tests with the race detector enabled
go test -v -race ./...

# 3. Verify SQL migrations up and down on a scratch DB
goose -dir migrations sqlite3 /tmp/test_refinery.db up
goose -dir migrations sqlite3 /tmp/test_refinery.db down
```

---

## 4. Pull Request Guidelines

- Keep PRs focused and small (ideally under 400 lines of diff).
- Reference the relevant specification (`docs/specs/`) or ADR (`docs/architecture/adr/`).
- Include table-driven unit tests for every new transformation rule or parser edge case.
- When changing UI templates or CSS, verify both light/dark schemes and keyboard navigation.
