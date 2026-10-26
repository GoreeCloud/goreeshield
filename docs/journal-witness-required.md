# GoreeShield — Witness-required Journal Marker

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Status | Linux-only synthetic recovery prototype; not production-ready |
| Base | [Local checksum witness](journal-checksum-witness.md) |
| Source | `internal/reconcile/disk_witness_required_linux.go` |

## Risk addressed

The previous journal checksum witness could be deleted without any local indication, because an absent witness was always interpreted as a legacy journal. This branch adds a versioned initialization marker: v1 denotes legacy or incomplete initialization, while v2 records that a **successfully synchronized witness is now mandatory**.

After a complete journal write, the source synchronizes the journal, then synchronizes the witness, and finally upgrades the existing private marker to v2 using a synchronized temporary file and atomic replacement. New operations are not acknowledged until that final step succeeds. A v2 marker with an absent witness returns `ErrWitnessMissing`. `DiskSource.Apply`, `Inspect`, and `Count` propagate the error and refuse to infer or repeat an action.

## Compatibility and recovery boundaries

Existing v1 markers and old unmarked journals remain readable if their witness is absent. On the next successful write, the journal is given a verified witness and the marker is promoted to v2. Missing source journal history remains rejected under either marker version. Corrupted or overly permissive marker files still fail closed.

If a process is interrupted after the journal or witness is replaced but before v2 marker promotion, a v1 marker can remain. A subsequent operation may require operator inspection due to journal/witness mismatch. There is no automatic repair or replay. These are **separate local files**, not an atomic three-object transaction.

## Regression tests

The tests verify a successful synthetic source publication produces a v2 marker, removing its witness causes execution, inspection and counting to fail closed, and v1 legacy state without a witness remains readable until a subsequent write promotes it to v2.

This is only **accidental single-file loss detection**. Someone with sufficient same-UID access can replace or delete the journal, marker and witness together, or restore a matching older tuple. Neither SHA-256 nor the marker provides authenticated rollback protection, independent attestation, or a trusted monotonic counter. Real source systems require source-owned durable transactions, independent identity-bound proofs, approved recovery procedures and power-loss testing.

No production deployment, real user data or privacy-rights fulfillment is part of this branch.
