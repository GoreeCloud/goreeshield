# GoreeShield — Persistent Synthetic Source Journal

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Linux-only, local durability experiment; not production-ready |
| Repository | `GoreeCloud/goreeshield` |
| Source | `internal/reconcile/disk_source_linux.go` and `disk_storage_linux.go` |

## Purpose

The previous mock source stored an optional processing counter and its operation IDs only in process memory. Restarting the process lost both, so a returned acknowledgement or retry could not be reconciled with previously accepted local operations. The new `DiskSource` fixture stores **the synthetic counter and its source-owned operation record in one local JSON document**.

It is intentionally distinct from any actual GoreeCloud product. No personal information, remote requests, legal rights actions, real telemetry, or production credentials are used.

## Protocol

`DiskSource` implements the existing `reconcile.Source` interface. `Coordinator.Dispatch` still reports `pending_source_readback`; `Coordinator.Reconcile` independently queries the source journal with the separate synthetic `privacy.synthetic.inspect` permission. Merely creating or storing a record is not independently verified privacy fulfillment.

For each validated synthetic tenant/subject, the source derives a sibling ledger filename from its existing scoped choice path. No raw tenant or subject identifier is used in the filename; the hash is **not encryption, anonymity, or strong security isolation**.

The journal stores `version=1`, an integer demonstration action count, and up to **1,024** operation records with their exact versioned request and either `applied` or `denied` status. Source inspection requires an exact request match in the coordinator before an observed status can be reported.

## Transaction and replay rules

- A private lock file is opened without symlink following and locked with Linux `flock`. This serializes cooperating processes targeting the **same** scoped source journal.
- New synthetic actions check the current purpose-specific consent revision under the existing choice lock. The recorded counter increment and exact source operation ID are committed **together**, within that lock, through an atomic file rename followed by directory synchronization.
- Identical operation-ID retries reuse the persisted result without another counter increment. Changed content under an existing ID produces a conflict.
- An absent or mismatched grant produces a persistent `denied` record. Later re-granting does not revive that earlier operation ID.
- On source write failure, the coordinator must not claim success. If the file rename occurred before a reported synchronization failure, a retry reopens the persisted record rather than assuming it did not happen.
- Private regular-file checks, limited JSON parsing, full-input validation, operation ID checks, schema consistency and source-count reconciliation fail closed for malformed or unsafe journal input.
- A full journal rejects new IDs rather than evicting old IDs and permitting replay. There is **no retention / truncation / migration policy** for a real deployment.

These are **synthetic local state transitions**, not exactly-once execution for external effects. They do not implement a transactional outbox coupled to a different database or application.

## Tests and known gaps

Development tests cover a new source object reopening a previous journal, repeated calls, concurrent use of independent fixture instances, persisted default denial, malformed contents, overly permissive files, symlinked lock rejection, and coordinator readback.

Security and operational limits remain significant:

1. The entire experiment is confined to a single Linux UID. A malicious same-UID process can read, modify, delete, or replace source-state files and cannot be securely isolated by this fixture.
2. Filesystem ancestry checks and normal path operations are not race-free against malicious concurrent filesystem changes.
3. The two locks are advisory and work only for cooperating writers. No distributed transaction, remote consensus, queue, lease, or multi-host lock exists.
4. The persisted counter has no external side effect. Atomic file replacement does not guarantee exactly-once processing in a real source product, nor independent attestation or a tamper-evident receipt.
5. The current tests do not establish real process crash recovery under power loss, cross-process integration on production machines, rollback-safe migration, backup consistency, or OS-native installed service lifecycle.
6. The `DiskSource` does not authenticate callers itself. The synthetic `Coordinator` checks its fixture claims before invoking the source; real source-owned authorization still needs to be implemented and audited.
7. This is not a production source adapter and must never be exposed as a remote multiuser endpoint without an independently reviewed security design.

## Next gates

Implement the actual GoreeCloud Identity verifier, subject and tenant isolation at the real data owner, transactionally durable idempotency in each source product, authenticated and independently verifiable source observations, bounded and encrypted retention where required, health and recovery procedures, production consent migration safeguards, and native service and client acceptance.

**No production deployment, actual rights fulfillment, real-source receipt, or Stable release is established by this work.**
