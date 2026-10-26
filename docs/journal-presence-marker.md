# GoreeShield — Initialized Journal Presence Checks

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Linux-local synthetic recovery experiment; not a production control |
| Branch | `feat/journal-presence-marker` |
| Sources | `internal/reconcile/disk_marker_linux.go`, `disk_storage_linux.go` |

## Problem

Previously, a journal removed after storing operation IDs could be mistaken for a new empty journal. Such reset loses source-owned synthetic operation history and permits previously seen IDs to be considered new. A guard is needed for **accidental source file loss**.

## Implemented protocol

A small, privately owned marker (`*.initialized`) is created and synchronized in the same scoped local directory **before the first journal rename**. Reads check its type, ownership, permissions, single hard link, and exact versioned contents. If it exists but the journal does not, `readSourceLedger` returns `ErrJournalMissing` and `DiskSource.Apply`, `Inspect`, and `Count` fail closed via their existing source lock. Corrupt or overly permissive markers are refused using `ErrJournalMarker`.

An entirely new scope, with no journal and no marker, still starts empty. Old valid journals with no marker are read for backward compatibility; the next ordinary source write creates the marker before publication.

## Regression coverage

The dedicated tests cover:
- Lost journal after initialization: counting, inspection, and applying the prior operation all refuse to treat missing state as empty.
- Damaged marker contents and loosened permissions: the source rejects unsafe state.
- Legacy unmarked journal: it is readable; a later source operation adds the marker and remains inspectable.
- First write interrupted at the **private** pre-rename callback checkpoint: a marker left without a journal causes readback to fail closed, rather than being misclassified as fresh state.

The existing CI checks include committed Go formatting, `go test -race ./...`, `go vet ./...` and native Linux Go builds. Record final CI status from the specific final branch commit; earlier intermediate formatting failures do not establish final status.

## Limitations and production gates

**The marker is not cryptographic evidence or rollback protection.** A process with sufficient access to remove both files can still erase the evidence of initialization. A valid but replaced old journal is not detected. Linux same-UID isolation, path-ancestry races, physical power loss and distributed source processing are not addressed here.

An interruption between syncing the marker and renaming the journal can leave an intentionally fail-closed state requiring an operator-reviewed recovery procedure; the fixture does not implement automatic recovery, for good reason. Legacy journals without markers retain a migration window where accidental loss may not be distinguishable from first use.

No real personal data, GoreeCloud Identity verifier, deployed Linux service, production source application, independent privacy receipt, or actual privacy-rights fulfillment is included. Preserve draft status until recovery, migration and security review are complete.
