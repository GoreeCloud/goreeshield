# GoreeShield — Development Roadmap

| Field | Value |
| --- | --- |
| Type | Proposed engineering roadmap |
| Status | Planned, not scheduled or implemented |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Source | `GoreeCloud/goreeshield/docs/roadmap.md` |

## Phase 0 — Foundation

Confirm approved feature-family labels, scope, first native operating-system target, and the first participating GoreeCloud product. Document the initial personal-data inventory and processing purpose taxonomy. Create the threat model, privacy impact analysis, minimal receipts, data rights lifecycle, and versioned adapter contract. Choose technologies using verified ecosystem constraints instead of assumptions.

**Acceptance:** Clear architecture decisions, risk model, measurable criteria, and no false support or regulatory claims.

## Phase 1 — Native MVP

Implement a real target-OS service lifecycle, safe configuration, subject isolation, a purpose/data-category registry, the Choices consent and withdrawal engine, minimal receipts, and a synthetic product adapter. Connect verified subject authentication/authorization through GoreeCloud Identity rather than building duplicate identity logic. Provide automated tests.

**Acceptance:** Native build/install/runtime demonstrated on the declared initial OS; opt-in-off defaults, revocation, isolation, error handling, and recovery tested.

## Phase 2 — Visibility and rights

Build native Center and Lens surfaces following the active GoreeCloud UI authority; add Boundary integration reviews, Rights request orchestration, and one deliberately scoped real product adapter. Show unsupported, pending, failed, partial, and completed states accurately.

**Acceptance:** Product adapter and native UI tests establish both behavior and accessible presentation. Acceptance requires more than mockups or green builds.

## Phase 3 — Retention and enforcement

Implement source-owned access/export/correction/deletion flows, retention evaluation, exception handling, policy holds, bounded retries, and privacy receipts. Verify on synthetic nonproduction fixtures before any live destructive changes.

**Acceptance:** Results are verified per product and per subject; no queued request is misrepresented as actual deletion.

## Phase 4 — Additional native platforms

Introduce each new OS deliverable independently with its installer/service lifecycle, native UX, secure storage and OS permissions, accessibility, update/rollback, and runtime evidence. Keep unsupported OS targets labeled pending.

**Acceptance:** Platform-specific build, install, device/VM runtime, permissions and recovery verification.

## Phase 5 — Optional advanced families

Investigate Veil privacy/tracker controls only using legitimate platform capabilities, off by default. Develop Family roles only with clear guardianship, consent, legal and anti-surveillance boundaries.

**Acceptance:** Threat model, opt-in and withdrawal, per-platform feasibility and behavior tests.

## Prioritization

Build **Choices + Lens + native service** before broad automated enforcement. Avoid a "privacy score" until any metric is evidence-based, comprehensible, and validated. Add integrations incrementally and keep data at its owning source.

## Explicitly undecided

Programming language/framework, first target OS, database, API schema, initial adapter product, regulatory jurisdictions, licensing, exact family names, milestones/dates, and release status. These need evidence and governance review; this plan does not imply completion.
