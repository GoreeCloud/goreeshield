# GoreeShield — Local Journal Checksum Witness

Date: October 25, 2026
Status: Linux synthetic-fixture consistency test; not production-ready.

The private witness stores a SHA-256 digest of the published journal bytes. The writer updates the witness only after syncing the journal. Reads reject a mismatched journal with `ErrJournalRollback` or an invalid witness with `ErrJournalWitness`.

The witness is not cryptographic authentication. It shares the same Linux UID with the journal, and both files can be replaced together by the same actor. A missing witness remains accepted for compatibility with older journals. Neither physical power-loss recovery nor genuine external-source effects are proved.

## Commit protocol

The source first writes and synchronizes a private temporary journal, validates the marker and existing source state, replaces the journal, and synchronizes its parent directory. It then writes the updated fingerprint to a private temporary witness file, synchronizes and atomically replaces the witness, and synchronizes the parent directory again. Success is acknowledged only after both files have been published.

Because these are two **separate files**, a failure between their updates returns `ErrCommitUncertain` and can leave a mismatch. Such mismatch is deliberately treated as unresolved: no new write or automatic source operation is allowed until independently reviewed. The mismatch could be caused by a partially completed local commit, not necessarily an attack.

## Linux regression tests

- Replacing a valid journal with an older valid JSON snapshot, while leaving the newer witness intact, yields `ErrJournalRollback` on counting, inspection and applying.
- Unsafe permissions or corrupted witness bytes yield `ErrJournalWitness`.
- Older journals without a witness are readable for migration; their next successful journal write creates the witness.
- If an existing witness remains stale because a test process was interrupted after journal rename, inspection and retry fail closed instead of accepting an ambiguous action.

The prior subprocess interruption test now treats a post-rename, pre-witness update as an unresolved discrepancy. This is stronger fail-closed behavior, not power-loss durability.

## Remaining engineering gates

A missing witness is currently accepted for legacy compatibility. Removing the witness together with journal rollback can bypass this prototype comparison. A real implementation needs an independently authenticated source checkpoint or monotonic anchor, explicit migration and backup integrity controls, source-owned transactional idempotency, and tested native service recovery. No automatic witness re-baselining or silent journal repair is implemented.
