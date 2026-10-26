# GoreeShield — Local Journal Commit Checkpoints

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Linux development checks passed; not power-loss verified or production-ready |
| Base | [Ambiguous source acknowledgements](uncertain-source-acknowledgements.md) |
| Source | `internal/reconcile/disk_storage_linux.go` |
| Tests | `internal/reconcile/disk_checkpoint_linux_test.go` |

## Objective

A source journal writer must distinguish a failure before publishing an operation from a failure after the operation may already be visible. Treating both paths as ordinary failures risks repeating a real-world action after an unacknowledged commit.

This increment keeps the original `writeSourceLedger` interface intact and adds a private `writeSourceLedgerWithCheckpoint` path for deterministic local tests. Production callers provide no callback, and there is no network-accessible or configurable fault-injection API.

## Checkpoint semantics

| Point | Local behavior | Interpretation |
| --- | --- | --- |
| Before rename | Temporary candidate is removed; prior journal is unchanged | Caller can identify a pre-publication failure |
| After rename, before parent directory sync | New file is visible in the running fixture; writer returns `ErrCommitUncertain` | **Read back** the exact source-owned operation; do not assume the write failed |
| Parent-directory open fails after rename | Writer returns `ErrCommitUncertain` | Readback required |
| Parent-directory sync fails after rename | Writer returns `ErrCommitUncertain` | Readback required |
| Successful directory sync | Normal success from the local writer | Synthetic journal committed under the fixture's assumptions |

Tests use a bounded synthetic journal with an exact operation ID, registered demo purpose, notice version, and explicit consent revision. The post-rename test verifies the typed error via `errors.Is`, loads the source-owned observation with a fresh object, and proves an identical replay does not add another synthetic counter increment. The pre-rename test ensures the original record remains and the writer does not leave a temporary file behind.

## Security and operational limits

These are **controlled callback failures in an ordinary live Go process**. They are not SIGKILL tests, power-loss simulation, filesystem fault injection, crash recovery across reboots, or guarantees about storage hardware. Returning an error at a checkpoint does not reproduce a process being forcibly terminated.

The local journal remains one same-UID synthetic fixture using advisory locks and JSON atomic replacement. Its operation count is not an external side effect. No true GoreeCloud Identity verifier, source-system integration, signed receipt, rights fulfillment, or deployed Linux service is established here.

The next test gate is representative Linux fault-injection at actual rename, sync and power-loss boundaries, with a controlled lifecycle and independently checked persistence behavior. A separate product-owned transactional contract remains mandatory before any release or real user processing.

**Do not equate this check with independent evidence that privacy operations completed.**
