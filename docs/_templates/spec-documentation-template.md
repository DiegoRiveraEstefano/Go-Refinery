# Specification: [Feature Name]

- **Status:** Draft | Approved | Implemented
- **Owner:** Engineering
- **Target Packages:** `internal/domain/...`, `internal/application/...`, `internal/adapters/...`

---

## 1. Overview & Goal

Concise description of what this feature achieves for users uploading and refining messy files.

## 2. Functional Requirements

- **FR-1:** Description.
- **FR-2:** Description.

## 3. Data Schema & Validation Rules

| Field / Parameter | Type | Constraints | Error Code |
| :--- | :--- | :--- | :--- |
| `field_name` | `string` | Non-empty, max 255 chars | `ERR_INVALID_FIELD` |

## 4. Edge Cases & Failure Modes

- Empty files, corrupted streams, exceeded row error budget, client disconnects.

## 5. Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2
