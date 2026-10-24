# Development & Operations Guides (`docs/development/index.md`)

Guides for setting up, migrating, testing, and profiling **Go-Refinery**.

## Commands Reference

| Task | Command |
| :--- | :--- |
| Run server locally | `go run ./cmd/refinery/main.go` |
| Apply migrations | `goose -dir migrations sqlite3 ./data/refinery.db up` |
| Rollback last migration | `goose -dir migrations sqlite3 ./data/refinery.db down` |
| Create new SQL migration | `goose -dir migrations create <name> sql` |
| Run unit & integration tests | `go test -v -race ./...` |
| Static analysis | `go vet ./...` |
| Serve Go package docs | `go run golang.org/x/pkgsite/cmd/pkgsite@latest -http=:6060` |
