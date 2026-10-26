# GoreeShield

**GoreeShield** is GoreeCloud's proposed privacy platform for its self-hosted ecosystem. Its mission is to help people understand, govern, and verify how their personal information is collected, used, shared, retained, and deleted.

> **Status:** Early engineering proof of concept on the Linux feature branch. Synthetic local Choices functionality, a read-only Unix socket service, and automated tests exist. This is **not** a production service, a complete native app, a released platform, a live GoreeCloud integration, or evidence of legal compliance.

## Proposed feature families

| Family | Purpose |
| --- | --- |
| **GoreeShield Center** | Central privacy experience and status |
| **GoreeShield Choices** | Granular opt-in, preferences, withdrawal, and consent receipts |
| **GoreeShield Lens** | Processing inventory, purposes, recipients, and data-flow visibility |
| **GoreeShield Boundary** | Optional integrations, permissions, and sharing controls |
| **GoreeShield Rights** | Data access, correction, portability, and erasure requests |
| **GoreeShield Lifecycle** | Minimization, retention, holds, and verified deletion |
| **GoreeShield Veil** | Optional tracking protection where native platform capabilities permit |
| **GoreeShield Family** | Privacy-respecting household controls, without covert monitoring |

Family names are proposals, not confirmed independent products.

## Boundaries

- GoreeShield owns privacy choices and data-governance coordination.
- **Wardveil** owns malware, threats, vulnerabilities, security monitoring, and incident response.
- **GoreeCloud Vault** owns passwords, credentials, keys, and secrets.
- **GoreeCloud Identity** owns identity, authentication, and SSO.
- **GoreeCloud UI (GC UI)** is GoreeCloud's approved design authority.

GoreeShield should integrate through least-privilege, versioned interfaces, not duplicate these systems.

## Design commitments

Privacy-invasive optional features start **off** until the user knowingly opts in. Choices must be granular, reversible, and reflected in actual processing. Data stays with its owning product where possible. Do not collect raw personal payloads centrally by default. Verify actual outcomes before reporting access, correction, or deletion complete.

The planned server and apps must be genuinely native to each declared target operating system; browser wrappers are not native deliverables. Claims of support require target-specific evidence.

## Planning documents

- [Vision and scope](docs/vision.md)
- [Feature families](docs/feature-families.md)
- [Technical architecture](docs/architecture.md)
- [Privacy requirements](docs/privacy-model.md)
- [Roadmap](docs/roadmap.md)
- [Linux local Choices prototype](docs/prototype-linux.md)
- [Read-only native Unix socket service](docs/local-service.md)
- [Initial threat model](docs/threat-model.md)
- [Synthetic downstream consent-enforcement experiment](docs/synthetic-adapter.md)
- [Synthetic adapter contract v1 — choice revisions and idempotency](docs/adapter-contract-v1.md)
- [Legacy consent migration and Identity trust boundary](docs/identity-boundary.md)
- [Synthetic tenant/subject-scoped consent isolation](docs/scoped-consent.md)
- [Source-owned synthetic outcome reconciliation](docs/source-reconciliation.md)
- [Persistent synthetic source journal](docs/durable-source.md)
- [Linux cross-process journal and integrity verification](docs/journal-process-verification.md)
- [Journal crash-recovery assessment and staged-file tests](docs/journal-crash-recovery-assessment.md)
- [Deterministic journal commit checkpoints](docs/journal-commit-checkpoints.md)
- [Linux test subprocess interruption](docs/journal-process-interruption.md)
- [Journal initialization marker and missing-history safeguards](docs/journal-presence-marker.md)
- [Ambiguous source acknowledgement recovery](docs/uncertain-source-acknowledgements.md)

This **public** repository must not contain credentials, private keys, production data, private infrastructure details, or private incident information. Project planning documents are not a substitute for GoreeCloud's authoritative governance. Licensing and first supported platforms are to be decided before implementation.

## Development proof of concept

The `feat/linux-local-choices-prototype` branch contains a Linux-only Go prototype with `goreeshield`, `goreeshieldd`, and `goreeshieldctl`. It uses synthetic purposes, a local state file, and a same-user Unix socket. The daemon is read-only and has no TCP listener, no deployment and no GoreeCloud Identity or real-product integration. The included systemd user-service unit is an example, not an installed service. See [local service guidance](docs/local-service.md).

```sh
go test -race ./...
go vet ./...
GOOS=linux GOARCH=amd64 go build ./cmd/...
```

A passing source build does not establish representative-device installation, runtime acceptance, recovery, platform support or Stable status.

An additional stacked development branch, `feat/synthetic-adapter-gating`, demonstrates a purpose-specific local enforcement check using an ephemeral in-memory synthetic adapter and a `goreeshieldsim` CLI. This is **not** real product-source enforcement or a verified remote revocation protocol. See [synthetic adapter scope](docs/synthetic-adapter.md).

A further stacked branch, `feat/choice-revisions-idempotency`, adds monotonic local consent revisions and an internal versioned synthetic operation envelope with bounded in-memory duplicate handling. **It is not a remote adapter API, durable authorization token or privacy-rights completion proof.** The [adapter contract v1 experiment](docs/adapter-contract-v1.md) documents exact limits and production prerequisites.

## Consent migration safeguard

The `fix/legacy-consent-invariant-20261025` branch adds per-choice revision binding to close a fail-closed migration gap. Previous grants that lack a **per-choice** revision stay disabled even if another purpose changes; the owner must explicitly re-grant that exact purpose. The branch also introduces a **default-deny verifier interface**, synthetic claim-validation tests and an identity-gated test fixture. **No real GoreeCloud Identity token verifier, credential exchange, multi-user isolation or production integration is shipped.** See [the security and migration note](docs/identity-boundary.md).

## Subject and tenant isolation experiment

The optional `feat/scoped-choice-isolation-20261025` branch models separately stored synthetic consent decisions for each **verified fixture tenant and subject**. The actual GoreeCloud Identity integration is not wired; all records remain under one Linux user and the initial unscoped CLI/service still exist. Hash-based filenames do not provide secrecy or same-user security isolation. See [scoped consent design](docs/scoped-consent.md) for current tests and production acceptance gates.

## Source observation and reconciliation experiment

The `feat/source-receipts` development branch adds a local, in-memory **mock source** and a separate coordinator. A submitted synthetic operation stays `pending_source_readback`; a separately authorized inspection must read the source's exact record before reporting a synthetic observation. Missing and conflicting outcomes remain unresolved or explicitly conflicted. No real source product is connected, no personal data is processed, and the in-memory observations are **not** durable receipts or verified privacy rights fulfillment. See [source reconciliation contract](docs/source-reconciliation.md).

## Local durable source experiment

The `feat/durable-source-journal` branch adds a strictly local `DiskSource` fixture. A synthetic counter and operation ID are saved together using a private Linux JSON journal, advisory file locking, atomic rename and source readback. Test objects can reopen the state and reject repeated IDs, invalid records and unsafe file permissions. This does **not** make real external processing exactly-once, does not provide tamper-proof receipts or source-independent verification, and has no live GoreeCloud Identity integration or production data. See [persistent source journal scope](docs/durable-source.md).

## Cross-process recovery testing

The `feat/durable-journal-process-tests` branch expands the local synthetic source journal with real Go test subprocesses exercising the same operation ID, additional journal/lock hardlink refusal, and corrupted-record validation. It is **not** a real source application or a completed power-loss/crash-recovery test. See the [process verification limitations](docs/journal-process-verification.md).

## Journal interruption assessment

The stacked `feat/journal-crash-recovery` branch adds local tests ensuring that orphaned journal staging files cannot override a committed synthetic observation and are **not** automatically promoted to repair a corrupted authoritative journal. Controlled process termination at the journal writer's `fsync`/rename checkpoints has **not** been verified. See the [crash-recovery assessment](docs/journal-crash-recovery-assessment.md). No real source product or production data is involved.

## Local journal write checkpoints

The `feat/journal-write-fault-invariants` branch distinguishes the **pre-rename failure** path from an **uncertain post-rename source commit**, returning a typed `ErrCommitUncertain` for failures while reopening or syncing the journal's parent directory. An internal checkpoint callback supports deterministic development tests without changing normal calls. Tests exercise journal visibility, temporary-file cleanup, exact source readback and duplicate replay. **These tests do not simulate process termination, power loss or real external effects.** See [journal commit checkpoints](docs/journal-commit-checkpoints.md).

## Local writer-process interruption

The `test/journal-linux-process-interrupt-20261025` branch adds isolated Linux child-process interruption checks at the private `before_rename` and `after_rename` journal checkpoints. Recovery is evaluated through fresh source readback and exact operation-ID replay behavior in temporary synthetic files. This is **not** a power-loss, external-source transaction, signed receipt or production readiness test. See [process interruption scope](docs/journal-process-interruption.md).

## Missing synthetic journal safeguard

The `feat/journal-presence-marker` development branch stores a private initialization marker before publishing the first local source journal. After initialization, a missing or invalid journal can no longer silently reset operation history when the marker survives. Tests cover journal loss, marker corruption and permissions, unmarked legacy reads and first-write interruption. This only detects **some accidental local losses**, not malicious deletion, rollback, physical power loss or production rights fulfillment. See [journal presence marker](docs/journal-presence-marker.md).
