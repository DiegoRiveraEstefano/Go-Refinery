# Go-Refinery

**Go-Refinery** is a high-throughput data ingestion, schema inference, and transformation service written in Go. Users upload raw, messy files (**CSV**, **Excel `.xlsx`**, **JSON/JSONL**, or **plain-text logs**), and the system asynchronously infers schemas, validates and normalizes records, persists clean data into SQLite, and streams real-time progress via Server-Sent Events (SSE).

---

## Core Features

- **Streaming Multi-Format Ingestion:** Bounded-memory processing for massive files using `encoding/csv`, `encoding/json`, `bufio.Scanner`, and `excelize` streaming iterators.
- **Automatic Schema Inference:** Heuristic detection of delimiters, headers, data types (`INTEGER`, `FLOAT`, `BOOLEAN`, `DATETIME`, `EMAIL`, `UUID`, `STRING`), null ratios, and anomalies.
- **Configurable Transformation Pipeline:** Composable column rules (trimming, Unicode normalization, type casting, null replacement, regex extraction, range checks, deduplication).
- **Asynchronous Engine & Real-Time Telemetry:** Native Go worker pool (`chan` + `context.Context`) broadcasting progress over **Server-Sent Events (SSE)**.
- **Hexagonal Architecture (Ports & Adapters):** Strict separation between pure domain logic and infrastructure adapters.
- **Standard Library First:** Built primarily on the Go 1.22+ standard library (`net/http`, `database/sql`, `html/template`, `log/slog`), with zero ORMs.
- **Semantic Hypermedia Frontend:** Semantic **HTML5**, element-first **CSS3** with minimal layout/utility classes, **Unpoly** for progressive navigation, and native **Web Components** for interactive widgets.

---

## Tech Stack & Approved Libraries

| Layer | Technology / Package | Rationale |
| :--- | :--- | :--- |
| **Runtime & HTTP** | Go 1.22+ (`net/http`, `html/template`, `log/slog`) | Native method/wildcard routing, structured logging, zero external web frameworks. |
| **Database** | SQLite via `database/sql` + `modernc.org/sqlite` | Pure-Go SQLite driver (no CGO), WAL mode concurrency, raw SQL only (no ORM). |
| **Migrations** | `github.com/pressly/goose/v3` | Versioned SQL migrations via CLI or embedded `embed.FS`. |
| **Excel Parsing** | `github.com/xuri/excelize/v2` | Streaming row iterator (`Rows()`) for large `.xlsx` files without RAM exhaustion. |
| **CSV / JSON / Logs** | `encoding/csv`, `encoding/json`, `bufio` | Go standard library streaming over `io.Reader`. |
| **Real-Time Events** | Server-Sent Events (SSE) (`http.Flusher`) | Native unidirectional streaming over standard HTTP. |
| **Frontend UI** | HTML5 + CSS3 + JS (ES6+) + Unpoly + Web Components | Semantic markup, low-class CSS, hypermedia fragments, and native Custom Elements. |

---

## Project Layout (Hexagonal Architecture)

```text
Go-Refinery/
├── cmd/
│   └── refinery/
│       └── main.go                 # Composition root, dependency wiring, graceful shutdown
├── internal/
│   ├── domain/                     # Pure business logic (zero external imports)
│   │   ├── dataset/                # Dataset, Row, ColumnSchema, DataType entities
│   │   ├── pipeline/               # Validation, normalization, and transformation rules
│   │   └── job/                    # Async job state machine and progress events
│   ├── ports/                      # Primary (inbound) and Secondary (outbound) interfaces
│   │   ├── inbound.go              # Application use-case interfaces
│   │   └── outbound.go             # Repositories, file store, parser, and event broker interfaces
│   ├── application/                # Use-case orchestration
│   │   ├── ingest_service.go       # Upload streaming and initial registration
│   │   ├── schema_service.go       # Sample reading and type inference
│   │   └── refinery_service.go     # Worker pool execution and batch persistence
│   └── adapters/                   # Infrastructure implementations
│       ├── inbound/
│       │   └── http/               # net/http handlers, middlewares, SSE, HTML templates
│       └── outbound/
│           ├── sqlite/             # Raw SQL queries, prepared statements, transactions
│           ├── parsers/            # CSV, Excelize, JSONL, and Log stream readers
│           ├── storage/            # Local filesystem blob management
│           └── sse/                # In-memory pub/sub progress broker
├── migrations/                     # Goose SQL migration files (00001_*.sql)
├── web/                            # Embedded frontend assets (embed.FS)
│   ├── templates/                  # Semantic HTML5 templates & Unpoly partials
│   └── static/
│       ├── css/                    # Element-first CSS3, design tokens, layout utilities
│       └── js/                     # Unpoly configuration & native Web Components
└── docs/                           # Technical documentation, ADRs, specs, and templates
```

---

## Quick Start

### Prerequisites
- **Go** `1.22` or higher.
- **Goose CLI** (optional, migrations also run via application flags):
  ```bash
  go install github.com/pressly/goose/v3/cmd/goose@latest
  ```

### Run Locally
```bash
# 1. Apply SQLite migrations
goose -dir migrations sqlite3 ./data/refinery.db up

# 2. Start the server
go run ./cmd/refinery/main.go

# 3. Run tests with race detector
go test -v -race ./...
```

---

## License

Licensed under the MIT License. See [LICENSE](file:///home/diego/Proyectos/Go-Refinery/LICENSE) for details.
