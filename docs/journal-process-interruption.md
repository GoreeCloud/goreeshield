# GoreeShield — Linux Child-Process Journal Interruption Tests

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Experimental process-interruption test; not power-loss or production acceptance |
| Parent | [Deterministic journal checkpoints](journal-commit-checkpoints.md) |
| Source | `internal/reconcile/disk_process_interruption_linux_test.go` |

## Objective

A callback returning an error after a journal checkpoint is less realistic than a process exiting while an operation is in flight. This experiment uses the **Go test executable itself** as an independent Linux child process, limited to a temporary directory with synthetic tenant, subject, choice and operation ID fixtures.

The parent creates a known empty authoritative journal and affirmative synthetic consent. Each child attempts a direct, purely local journal candidate write using the **private** test-only checkpoint callback already introduced in the source writer. When the child prints the checkpoint marker, the test parent interrupts the child; then a separate parent-side source instance inspects the remaining authoritative state.

## Two interruption cases

| Checkpoint | Expected authoritative state after child termination | Recovery rule |
| --- | --- | --- |
| `before_rename` | Original zero-count journal remains; incomplete candidate may remain as an orphan temp | Never promote staging files. A new, authorized synthetic operation with that ID may be assessed against current consent |
| `after_rename`, before directory sync | Newly renamed journal is visible in this running test environment, with one recorded synthetic action | **Read back** the exact operation ID and request. Retrying it must not increment the synthetic counter |

Each case reuses the existing `DiskSource` and checks the eventual counter remains exactly one after a retry. The test child cannot touch production user files: the parent supplies a freshly created temporary scoped directory and fixed synthetic identifiers.

The writer callback is unexported and normal journal writes use `nil` for it. This code does not install or invoke a real GoreeCloud source product.

## Interpretation and exclusions

The child process is intentionally terminated only **after it emitted its checkpoint marker**. This provides test evidence about process termination and currently visible files on a CI Linux filesystem. It is not a power-loss test, an interrupted kernel-level rename, durable media flush test, disk-controller cache test, cross-host replica test, or external exactly-once processing proof.

In particular, **a renamed record that can be read in the running machine is not necessarily guaranteed durable after sudden power failure**. Post-rename directory synchronization did not complete in this child. A real product must use source-owned recovery, durable transactional idempotency and independently verified receipts.

All journal contents and process identities here are synthetic. There is no real personal-data operation, GoreeCloud Identity verifier, signed rights receipt, OS service installation or production readiness claim.

## Remaining acceptance gates

Develop OS-native test environments that can safely exercise actual filesystem failures and power-loss recovery, verify source-owned transactional outcomes under crashes, specify durable journal retention and backup behavior, and integrate true Identity-bound privacy checks into an actual GoreeCloud-owned product before advancing release status.
