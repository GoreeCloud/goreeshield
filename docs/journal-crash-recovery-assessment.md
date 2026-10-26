# GoreeShield — Journal Crash-Recovery Assessment

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Assessment only; no verified power-loss or forced-termination test |
| Baseline | [Cross-process verification](journal-process-verification.md) |

## Verified baseline

The existing Linux synthetic journal has passing CI evidence for concurrent independent processes, file-lock cooperation, replay prevention, stable source readback, and rejection of invalid journal records. The baseline is [draft PR #13](https://github.com/GoreeCloud/goreeshield/pull/13), not a production release.

## Interruption windows to validate

Before temporary-file synchronization, a crash may leave only a partial temporary file; the committed journal must remain authoritative. After temporary-file sync but before rename, the temporary file must not be promoted into an authoritative outcome. After rename but before the parent-directory synchronization, an operation may be visible even if the caller receives no acknowledgement; the caller must inspect the source record before deciding whether to retry. After directory synchronization, a fresh process should read the committed operation exactly once.

These are **test objectives**, not claims that the current tool run has executed fault injection at any of these stages.

## Safety requirements

- Make the crash checkpoint controllable only by local test fixtures, not exposed through a public service endpoint.
- Test only temporary directories with synthetic tenant and subject identifiers. Do not terminate user processes or modify real production state.
- Every ambiguous outcome remains unresolved until readback of the owning source journal.
- If corruption or unsupported version is encountered, preserve existing files, stop writes, and surface an actionable error rather than silently resetting state.
- Do not infer power-loss durability or real external exactly-once effects from ordinary Go tests or a clean process restart.

## Verified local staging invariants

Two development tests exercise recovery-adjacent cases without terminating a process at a writer checkpoint:

- `TestStaleStagingFileDoesNotOverrideCommittedSource` writes a malformed file with the writer's staging prefix after a valid source commit. A fresh `DiskSource` still reads the exact committed observation; retrying the operation ID does not increment the synthetic counter again.
- `TestStagingFileCannotRepairCorruptedCommittedJournal` leaves a valid-looking staged copy alongside a deliberately corrupted authoritative journal. A fresh source **refuses to read or accept new operations** instead of silently promoting untrusted staged bytes.

These tests verify that stray temporary filenames are never treated as authoritative. They do not test a real kill at a rename or sync checkpoint, power loss, on-disk torn-write recovery, or production transaction guarantees.

## Current limitation

This assessment does not itself implement interrupted-write injection or a test that forcibly terminates a source process at a known disk checkpoint. Source-writer edits and process-termination test changes were blocked in the current integration session. The production acceptance gate remains open.

The next authorized test should use a controlled native Linux test environment and record separate evidence for crash checkpoints, failure recovery, file integrity, error semantics, retained permissions, and real lifecycle behavior before changing release status.
