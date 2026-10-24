# ADR-003: Semantic HTML5, Class-Light CSS3, Unpoly, Web Components & SSE

- **Status:** Accepted
- **Date:** 2026-10-24
- **Deciders:** Engineering Team
- **Related Packages:** `internal/adapters/inbound/http`, `internal/adapters/outbound/sse`, `web/`

---

## 1. Context & Problem Statement

The UI must feel responsive when uploading files, configuring cleaning rules, and watching real-time job progress, while avoiding SPA complexity, virtual DOM frameworks, or utility-class bloat.

## 2. Considered Options

1. **Semantic HTML5 + Element-First CSS3 (`.l-*` / `.u-*`) + Unpoly + Native Web Components + Server-Sent Events (SSE).**
2. **React/Vue SPA with REST/WebSockets and Tailwind CSS.**

## 3. Decision Outcome

Chosen option: **Option 1**.

- Server renders semantic HTML5 via `html/template`.
- **Unpoly** handles partial fragment replacements (`up-target`), forms (`up-submit`), and modal layers (`up-layer`).
- **Native Web Components** (`<refinery-dropzone>`, `<refinery-progress>`, `<refinery-schema-editor>`) encapsulate rich client behavior in Light DOM.
- **SSE** (`text/event-stream` via `http.Flusher`) streams real-time job progress from the Go worker pool directly to `<refinery-progress>`.

## 4. Consequences

### Positive
- Zero JS build step required; assets embed cleanly via `embed.FS`.
- Accessible, lightweight markup and automatic browser reconnection for SSE.

### Negative / Trade-offs
- Developers must adhere strictly to semantic HTML structure so element-level CSS rules apply consistently.
