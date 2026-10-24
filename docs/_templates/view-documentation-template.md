# HTTP View / Handler: `[EndpointName]`

- **Route Pattern (Go 1.22+):** `METHOD /path/{param}`
- **Handler Package:** `internal/adapters/inbound/http`
- **HTML Template:** `web/templates/<name>.html`

---

## 1. Purpose

Describe the user interaction handled by this endpoint.

## 2. Request & Response Contract

- **Request Headers:** Standard HTTP or Unpoly (`X-Up-Target`, `X-Up-Validate`).
- **Request Payload:** Form fields, query params, or `multipart/form-data` stream.
- **Response Type:** Full HTML5 page, Unpoly HTML fragment, or `text/event-stream` (SSE).

## 3. Interactive Elements

- **Unpoly Attributes:** `up-target`, `up-submit`, `up-layer`.
- **Web Components Used:** `<refinery-...>`
