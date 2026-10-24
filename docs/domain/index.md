# Domain Layer (`internal/domain/`)

This directory documents the core domain entities, value objects, search predicates, and pipeline transformation stages:

- [`dataset.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/dataset/dataset.go) & [`inference.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/dataset/inference.go): Core `Dataset`, `Schema`, `ColumnSchema`, `Record` entities and heuristic type inference.
- [`search.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/dataset/search.go): Mini MongoDB-inspired query and search engine (`SearchQuery`, `CompiledQuery`) supporting:
  - Free-text case-insensitive search (`$contains`)
  - Regular expressions (`/pattern/i` or `$regex`)
  - Relational and set operators (`$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`, `$in`, `$exists`)
  - Compound MongoDB-style JSON documents (`{"age": {"$gte": 25}, "$or": [...]}`) and Mini-DSL expressions (`age >= 25 AND country != MX`)
- [`job.go`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/job/job.go): Asynchronous refinement `Job`, `ProgressEvent`, and `RowError` models.
- [`pipeline/`](file:///home/diego/Proyectos/Go-Refinery/internal/domain/pipeline/pipeline.go): 21 modular data cleaning, normalization, and validation stages across `registry.go`, `text_stages.go`, `numeric_stages.go`, and `validation_stages.go`.
