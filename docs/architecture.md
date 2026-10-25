# GoreeShield — Proposed Native Architecture

| Field | Value |
| --- | --- |
| Type | Technical design |
| Status | Proposal; not implemented |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Source | `GoreeCloud/goreeshield/docs/architecture.md` |

## Principle

Keep personal data with its owning product whenever possible. GoreeShield stores only minimal privacy governance metadata, explicit user choices, scoped work records, and bounded receipts. Integrations use authenticated, least-privilege APIs rather than broad database access.

## Core components

| Component | Role |
| --- | --- |
| **Native GoreeShield Service** | Self-hosted API, lifecycle, authorization checks, and orchestration |
| **Processing Registry** | Declared product data categories, purposes, recipients, retention, and notice versions |
| **Choice Ledger** | Per-subject and per-purpose consent decisions, revocation, and evidence |
| **Policy Evaluator** | Distinguish necessary from optional processing, enforce applicable choices |
| **Rights Orchestrator** | Authorized access, correction, portability, and erasure workflows |
| **Lifecycle Scheduler** | Evaluate retention, exceptions, legal holds, and cleanup outcomes |
| **Privacy Receipts** | Minimal, protected records of actions and verified results |
| **Product Adapters** | Source-owned enforcement and status reporting |
| **Native Clients** | Accessible controls and truthful operation status following GC UI |

## Conceptual integrations

GoreeCloud Identity provides verified subject identity and scoped authorization claims; GoreeCloud Vault retains ownership of secrets; Wardveil may provide narrowly relevant, redacted security status. GoreeCloud UI governs approved design and experience. GoreeShield does not duplicate the work of these systems.

Each source application remains authoritative for its own personal records and must execute its own authorized corrections or deletion. One GoreeShield operation can have many per-product outcomes. Neither HTTP success nor a queued job proves deletion.

## Request states

`draft → verified → evaluated → authorized → dispatched → awaiting evidence → completed / partially completed / denied / failed`

Require clear cancellation, retry, unsupported-adapter, legal-hold, timeout, and exception handling. Correlate requests with idempotent identifiers and verify final source acknowledgements before representing work as complete.

## Initial data model (conceptual)

- `processing_activity`: product, purpose, category, recipient class, retention policy, disclosure version.
- `privacy_choice`: authorized subject, purpose, product, affirmative choice, notice version, timestamp, provenance, effective state.
- `privacy_request`: subject, operation, scoped eligibility, verification/policy evidence, overall state.
- `request_target`: source product, scoped action, result, exception, verification evidence.
- `privacy_receipt`: minimally necessary event metadata, responsible system, event result, retention class.

No raw private content should be copied into these tables by default. Personal identifiers, access logging, cryptographic design, database, and backup retention must be established after a concrete threat model.

## Native platform requirement

Implement an actual OS-native service and actual OS-targeted native clients for each declared supported platform. A website, PWA, WebView, or Electron wrapper cannot stand in for a native application. Linux, Windows, macOS, Android, iOS, and iPadOS support must **not** be claimed until each relevant target is built, installed, and tested. A supplementary web interface may exist without being represented as native.

## Decisions still open

Initial native OS, languages, storage, packages, adapter versioning, endpoint security, update/rollback process, backups, exact UX, first integrated product, and applicable jurisdiction rules. Do not silently convert these proposals into deployed architecture.
