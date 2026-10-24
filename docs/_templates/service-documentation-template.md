# Application Service: `[ServiceName]`

- **Inbound Port:** `ports.[InboundInterface]` (`internal/ports/inbound.go`)
- **Implementation:** `application.[ServiceStruct]` (`internal/application/<file>.go`)
- **Outbound Dependencies:** `ports.[Repository]`, `ports.[Broker]`

---

## 1. Responsibility

Describe the use cases orchestrated by this service.

## 2. Exported Methods

### `MethodName(ctx context.Context, ...) (..., error)`
- **Inputs:** Parameters and domain preconditions.
- **Outputs:** Returned domain structs or sentinel errors.
- **Transaction Boundaries:** Whether the operation runs inside a SQLite transaction or spawns an async job.

## 3. Error Conditions

| Domain Error | Cause | HTTP Mapping |
| :--- | :--- | :--- |
| `domain.ErrNotFound` | Entity ID does not exist | `404 Not Found` |
| `domain.ErrInvalidSchema` | Column rules failed validation | `422 Unprocessable Entity` |
