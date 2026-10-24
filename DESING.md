# Visual Design System & UI Rules (`DESING.md`)

This document defines the visual language, design tokens, typography, layout primitives, and UI component rules for **Go-Refinery**.

---

## 1. Design Philosophy

1. **Industrial Data Clarity:** High contrast, tabular numeric alignment, and zero visual clutter. Designed for inspecting dense datasets, schemas, and error logs.
2. **Semantic HTML First:** Visual styling attaches directly to semantic HTML5 tags (`table`, `form`, `fieldset`, `dialog`, `progress`, `details`) rather than proliferating utility classes.
3. **Class-Light CSS Architecture:**
   - **80% Element Selectors:** Native HTML elements look complete by default.
   - **15% Layout Classes (`.l-*`):** Structural composition using CSS Grid and Flexbox.
   - **5% Utility Classes (`.u-*`):** Single-purpose overrides (alignment, monospace, badges).
4. **Progressive Hypermedia:** Fast fragment updates via **Unpoly**, augmented with lightweight native **Web Components** only where stateful client interactivity is required.

---

## 2. Design Tokens (CSS Custom Properties)

All visual values are centralized in `:root` inside `web/static/css/tokens.css`.

```css
:root {
  /* Surface & Background Palette (Light Mode Default) */
  --color-bg-canvas: #f8fafc;
  --color-bg-surface: #ffffff;
  --color-bg-muted: #f1f5f9;
  --color-bg-elevated: #ffffff;

  /* Text Hierarchy */
  --color-text-primary: #0f172a;
  --color-text-secondary: #475569;
  --color-text-muted: #64748b;
  --color-text-on-accent: #ffffff;

  /* Brand & Interactive Accent (Refinery Amber / Industrial Blue) */
  --color-accent: #0284c7;
  --color-accent-hover: #0369a1;
  --color-accent-subtle: #e0f2fe;

  /* Semantic Status Colors */
  --color-success: #15803d;
  --color-success-bg: #dcfce7;
  --color-warning: #b45309;
  --color-warning-bg: #fef3c7;
  --color-danger: #b91c1c;
  --color-danger-bg: #fee2e2;

  /* Borders & Dividers */
  --color-border: #cbd5e1;
  --color-border-subtle: #e2e8f0;
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;

  /* Typography */
  --font-sans: system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
  --font-mono: ui-monospace, "Cascadia Code", "Source Code Pro", Menlo, monospace;
  --text-xs: 0.75rem;
  --text-sm: 0.875rem;
  --text-base: 1rem;
  --text-lg: 1.125rem;
  --text-xl: 1.25rem;

  /* Spacing Scale (4px / 0.25rem base) */
  --space-1: 0.25rem;
  --space-2: 0.5rem;
  --space-3: 0.75rem;
  --space-4: 1rem;
  --space-6: 1.5rem;
  --space-8: 2rem;
}

@media (prefers-color-scheme: dark) {
  :root {
    --color-bg-canvas: #0b0f19;
    --color-bg-surface: #111827;
    --color-bg-muted: #1e293b;
    --color-bg-elevated: #1f2937;
    --color-text-primary: #f8fafc;
    --color-text-secondary: #cbd5e1;
    --color-text-muted: #94a3b8;
    --color-accent: #38bdf8;
    --color-accent-hover: #7dd3fc;
    --color-accent-subtle: #0c4a6e;
    --color-border: #334155;
    --color-border-subtle: #1e293b;
  }
}
```

---

## 3. Typography & Data Readability Rules

1. **Tabular Numerics:** All table cells containing numbers, percentages, row counts, timestamps, or byte sizes MUST apply `font-variant-numeric: tabular-nums`.
2. **Monospace for Raw Data & Schemas:** Column identifiers, inferred types (`INTEGER`, `DATETIME`), regex patterns, and raw log lines use `var(--font-mono)` (`<code>`, `<kbd>`, `<samp>`, or `.u-mono`).
3. **Line Length:** Prose blocks (`<p>`, `<article>`) are constrained to `65ch–75ch`. Data tables (`<table>`) expand to `100%` of their scroll container.

---

## 4. Semantic Element Styling Rules

Instead of `.btn`, `.card`, or `.form-control`, style native HTML elements directly:

| HTML Element | Visual Rule |
| :--- | :--- |
| `<header>`, `<footer>` | Top/bottom landmark bars with `1px solid var(--color-border)` and horizontal flex alignment. |
| `<main>` | Centered container (`max-width: 88rem`), padded with `var(--space-6)`. |
| `<article>`, `<section>` | Surface panels (`background: var(--color-bg-surface)`, subtle border, `--radius-md`, padding `--space-4`). |
| `<table>` | Full width, `border-collapse: collapse`, sticky `<thead>`, zebra-striped `<tbody> tr:nth-child(even)`, hover highlight. |
| `<fieldset>` | Grouped rule configuration box with a bold `<legend>` badge. |
| `<button>`, `input[type="submit"]` | Primary action style by default; `[data-variant="secondary"]` or `[data-variant="danger"]` for variations. |
| `<progress>` | Styled bar (`height: 0.75rem`, rounded) reflecting job completion percentage. |
| `<dialog>` | Centered modal with `::backdrop` blur, managed by Unpoly overlays (`up-layer="new"`) or native JS. |
| `<details>` & `<summary>` | Collapsible inspection panels for raw error samples and JSON metadata. |

---

## 5. Layout & Utility Classes

### 5.1 Layout Primitives (`.l-*`)
- `.l-container`: Constrains max width and applies responsive horizontal padding.
- `.l-stack`: Vertical flex container with `gap: var(--space-4)`.
- `.l-cluster`: Horizontal wrapping flex container (`gap: var(--space-2)`, `align-items: center`).
- `.l-grid`: Responsive CSS Grid (`grid-template-columns: repeat(auto-fit, minmax(18rem, 1fr))`).
- `.l-sidebar`: Two-column split (fixed sidebar for pipeline rules + fluid main area for dataset preview).
- `.l-table-scroll`: Horizontal scroll wrapper (`overflow-x: auto`) for wide datasets (50+ columns).

### 5.2 Utility Classes (`.u-*`)
- `.u-mono`: Forces `var(--font-mono)` and `tabular-nums`.
- `.u-text-right` / `.u-text-center`: Text alignment for numeric/action columns.
- `.u-truncate`: Single-line ellipsis (`max-width: 24ch; overflow: hidden; text-overflow: ellipsis; white-space: nowrap`).
- `.u-sr-only`: Screen-reader-only accessible text.
- `.u-status`: Pill indicator styled via `[data-status="queued|running|completed|failed"]`.

---

## 6. Interactive Patterns (Unpoly + Web Components)

1. **Unpoly Transitions:**
   - Use `[up-main]` on `<main>` and `[up-nav]` on `<nav>`.
   - Highlight loading targets subtly using `.up-loading { opacity: 0.6; pointer-events: none; transition: opacity 150ms ease; }`.
2. **Web Components (`<refinery-*>`):**
   - Use **Light DOM** by default so global element styles (`table`, `input`, `progress`) apply seamlessly without duplicating CSS inside Shadow DOM.
   - `<refinery-dropzone>`: Dashed border box that highlights with `--color-accent` on `dragover`.
   - `<refinery-progress>`: Displays `<progress>`, live row counters, throughput (`rows/s`), and elapsed time updated via SSE.
   - `<refinery-schema-editor>`: Interactive table allowing users to toggle column types and append transformation stages before launching a job.
