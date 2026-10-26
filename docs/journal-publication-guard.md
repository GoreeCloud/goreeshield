# GoreeShield — Journal Publication Guard and Recovery Review

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Linux-only synthetic recovery guard; not production-ready |
| Base | [Journal presence marker](journal-presence-marker.md) |
| Source | `internal/reconcile/disk_marker_linux.go` |
| Tests | `internal/reconcile/disk_publication_guard_linux_test.go` |

## Why this change matters

The journal presence marker in the earlier branch stopped `DiskSource.Apply`, `Inspect` and `Count` from interpreting a missing previously initialized journal as an empty history. However, the private lower-level writer could still accept a valid marker and directly replace a missing or corrupted journal. It could therefore defeat the recovery gate if invoked from a future source adapter or maintenance path.

This increment closes that gap without adding a privileged repair command. When a journal marker already exists, the writer validates the authoritative journal through `readSourceLedger` before publishing a new candidate. If an initialized journal is missing or damaged, the write is refused, and the original state is left untouched. The same check applies if marker initialization encounters an existing valid marker.

## Verified development test intentions

1. Initialize a journal, remove its authoritative file, then invoke the lower-level writer directly. The writer must return `ErrJournalMissing` and must not publish a new empty journal.
2. Initialize a journal, replace its contents with malformed JSON, and invoke the lower-level writer directly. The writer must return an error and preserve the damaged bytes instead of overwriting them.
3. Interrupt first publication at the private `before_rename` checkpoint after the marker is written. A direct retry must not silently treat the marked-but-missing journal as new.

These tests remain synthetic Linux local-fixture checks. Consult CI for the actual final result; successful execution does not prove physical power-loss durability or secure adversarial rollback detection.

## Operator review protocol (not an automated repair)

If `ErrJournalMissing`, `ErrJournalMarker`, or an invalid-journal error appears, **stop new source writes for the affected scope**. Record the exact error and relevant application/host version without exposing personal content, then preserve the source journal, marker and related scoped state under controlled, least-privilege incident procedures. Establish whether the event was a failed first publication, accidental file loss, storage rollback, inconsistent restoration, or unauthorized change. Independently verify the owning source's actual operation outcomes and idempotency history before deciding on a recovery strategy.

Do **not** delete the marker, silently recreate an empty journal, promote orphaned temporary files, or replay externally visible operations based solely on an absent local record. Recovery decisions and any restoration of history require an explicit operator-reviewed, source-owned procedure and appropriately authorized evidence. This prototype does not implement such a restoration mechanism.

## Boundaries

No installed native service or production system was modified. This fixture uses one Linux UID and synthetic records. Marker and file ownership checks do not defend against a same-UID actor who can replace both valid files, against authorized-but-malicious rollback, or against adversarial filesystem ancestry races. There is no cryptographic ledger anchor, trusted clock, real GoreeCloud Identity verification, externally owned transaction, authenticated source receipt, data-erasure fulfillment, or compliance attestation.

The milestone remains a draft PR stacked on the journal-marker work. Real power-loss testing and source-owned recovery remain separate release-blocking gates.
