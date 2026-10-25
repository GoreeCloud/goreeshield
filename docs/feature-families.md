# GoreeShield — Proposed Family Naming Registry

| Field | Value |
| --- | --- |
| Type | Product terminology proposal |
| Status | Proposed; not finalized or deployed |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Source | `GoreeCloud/goreeshield/docs/feature-families.md` |

GoreeShield is the umbrella brand. Family labels group distinct user-facing capabilities, not necessarily independently installed apps, services, or repositories.

| Family | Responsibility | Priority |
| --- | --- | --- |
| **Center** | Overview, privacy settings, clear status, and explanations | V1 |
| **Choices** | Granular optional processing decisions, notice versions, opt-in/withdrawal receipts | V1 |
| **Lens** | Data processing catalog: purposes, categories, sources, recipients, retention | V1 |
| **Boundary** | External sharing, third-party integrations, disclosure and permission review | V2 |
| **Rights** | Access, correction, export/portability, and erasure requests with verification | V2 |
| **Lifecycle** | Data minimization, retention, holds, cleanup and source-verified deletion | V2 |
| **Veil** | Optional, supported-platform native tracking controls (not malware scanning) | Research |
| **Family** | Limited, explicit household privacy settings with per-person boundaries | Later |

## Product overlap guardrails

- **Choices** manages decisions about purposes; **Boundary** manages optional sharing routes and recipients.
- **Lens** stores minimal inventory metadata; it is not a content-indexing or surveillance engine.
- **Rights** tracks requests and evidence; **Lifecycle** evaluates retention and authorized cleanup policies.
- **Veil** must not duplicate Wardveil's cyber-threat defenses. Device- and browser-level restrictions must be validated for each OS.
- **Family** relies on Identity roles. It cannot silently grant one household member access to another's private content.
- **Center** is a presentation layer, not a second storage authority.

## Naming policy

Use `GoreeShield <Family>` in product-facing language, with plain descriptive labels alongside brands (for example, **GoreeShield Choices — Privacy preferences**). Keep implementation identifiers stable and unambiguous; do not lock API paths, classes, or package names to unapproved labels. All family names above remain proposed pending founder decision and real platform feasibility.
