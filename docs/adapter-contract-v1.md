# GoreeShield — Synthetic Adapter Contract v1

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | **Experimental internal fixture; not a production API** |
| Canonical source | `GoreeCloud/goreeshield/docs/adapter-contract-v1.md` |

## Contract intent

A privacy choice displayed in GoreeShield is **not** automatically enforced in an owning product. This experiment models the minimum distinct concepts needed for verifiable, future source-side enforcement: a versioned request envelope, exact choice revision, notice version, repeat-operation identity, and bounded synthetic outcomes.

The code in `internal/contract` and `internal/synthetic` does not accept personal data or make network calls. It has no GoreeCloud Identity authorization, production signatures, persistent idempotency table, adapters, source-specific evidence, or live rights workflows.

## Request envelope, version 1

| Field | Required meaning |
| --- | --- |
| `Version` | Exact internal contract version; only `1` recognized |
| `OperationID` | Nonpersonal synthetic identifier, 3–64 characters, limited to lowercase ASCII, digits, hyphen, underscore |
| `PurposeID` | Registered synthetic optional purpose |
| `NoticeVersion` | Current declared notice version; stale versions rejected |
| `ChoiceRevision` | Exact positive revision of the local user’s stored choices |

The `ChoiceRevision` increments on every accepted local grant or withdrawal, including re-grants. The implementation uses a **single global local revision**. Any change to any purpose invalidates prepared requests for all purposes. This is intentionally conservative for the experiment and **not** a recommended production multi-user data model.

A stored version-1 choice file without a revision is recognized as **legacy, not authoritative**: its old grants fail closed until the person grants again explicitly. Overflow fails closed. Local state schema version remains `1` for compatibility with existing proof-of-concept files; `revision` is a new optional JSON field with restrictive absence handling.

## Response semantics

| Status | Meaning |
| --- | --- |
| `applied` | One synthetic in-memory counter increment occurred for this request |
| `denied` | No synthetic action; missing, withdrawn or mismatching consent revision |
| `duplicate` | This exact synthetic operation ID and request was seen before; no new action. `PriorStatus` records the historical response, **not** fresh authorization |
| `conflict` | Operation ID was reused with different fields; no new action |
| Error | Malformed envelope, unavailable state, or full bounded in-memory idempotency table; no action |

Repeat requests are serialized inside the *same* `Processor` object and executed at most once. This guarantee ends at process restart and is **not durable**, distributed, identity-bound or cryptographic. The table refuses operations after 1,024 unique operation IDs; it never evicts old IDs and accidentally permits replays.

The synthetic action and revision check occur under a cooperating Linux local choice-file lock. This means an already-executing action finishes before a withdrawal acquires that lock. Once withdrawal completes, stale prepared requests cannot execute; a subsequent re-grant receives a new revision, preventing the earlier grant revision from being reused.

## Explicit future integration requirements

- GoreeCloud Identity must authenticate source and subject identity, delegation, tenant and scope. A revision alone is **not** an authorization credential.
- A source-owned adapter must enforce the choice at each *actual* processing boundary. Cache invalidation, outbox retries, eventual consistency, background processing and third-party propagation require separate designs.
- Persistent idempotency needs a database transaction boundary, scoped key, expiry policy, duplicate-result semantics and crash recovery.
- Adopt monotonic per-subject/per-purpose fencing or equivalent signed authorization and revocation semantics after threat review. No HMAC or signature is implied by this experiment.
- The owning product must distinguish accepted, queued, failed, partially applied and independently verified completed. A local synthetic `applied` response must **never** be displayed as verified privacy rights fulfillment.
- Rate limiting, privacy of logs/receipts, legal holds, retention, consent provenance and notice localization remain open.

## Verification

Tests cover legacy grant default-deny, revision monotonicity, stale revision after withdrawal and re-grant, duplicate retry, changed-payload ID conflict, concurrent same-ID delivery, invalid envelope and default-deny. CI checks formatting, Go race tests, static analysis and native Linux build. No installation, release or real adapter is asserted.
