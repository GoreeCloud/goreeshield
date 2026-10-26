# GoreeShield — Durable Source-Owned Head Index (Linux Fixture)

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Status | Linux development fixture; not a source-owned production authority |
| Depends on | [Unix source-head process boundary](linux-source-head-process-boundary.md) |
| Code | `internal/checkpoint/disk_head_linux.go` |
| Tests | `disk_head_linux_test.go`, `disk_head_process_linux_test.go` |

## Objective

The source-head process fixture in PR #24 signs a fixed, ephemeral in-memory statement. It has no persistent monotonic history after a restart. This milestone provides a **separate, file-backed index** for the future *owning source*, distinct from GoreeShield's local consent journal, marker and witness.

The `checkpoint.DiskHeadStore` stores one canonical checkpoint `Statement` per opaque SHA-256 scope digest, with a private directory, 0600 files, source-side advisory `flock`, exact version validation, bounded JSON decoding, atomic replacement and file/directory synchronization. It does **not** generate, persist, use or expose an Ed25519 private signing key.

## State transitions

- `Initialize` is an explicit first-use action that requires sequence **1**. A private initialization marker is synchronized before its first head is published. Retrying first use after that marker exists refuses to reset history.
- `Read` returns the last recorded source statement. Missing never-initialized state is `ErrHeadAbsent`; a missing **previously initialized** file is `ErrHeadCorrupt`.
- `Advance` requires an existing, validated source head; its issuer and key ID cannot change in this initial contract. Only the next consecutive sequence may be published. An exact repeat of the current statement is idempotent.
- Older sequences, gaps, same-sequence conflicts, changed source identity and malformed or unsafe history are rejected without overwriting authoritative bytes.
- A publication failure after file rename may have changed the visible record; `ErrHeadCommitUncertain` requires source-owner readback. Do not interpret an error as proof of no commit.

## Linux tests

The development fixtures test readback through new store instances, sequence monotonicity, idempotent retries, issuer/record conflicts, deleted initialized history, invalid file content, broad file permissions and interruption between marker and first-head publication.

An additional test launches **two independent Linux test subprocesses** against the same private source directory. They both attempt the same next sequence, under the advisory lock; the resulting source head remains one valid monotonic state when read by a fresh object.

## Important exclusions

The directory is owned by the same Linux UID in CI. It is not protected from a malicious same-UID actor, symlink tricks in trusted ancestor directories, replay of the whole valid directory, virtual-machine rollback, or physical power loss. The marker is accidental-loss detection, not signed or independent tamper evidence. The code does not create a live source endpoint or a persistent daemon.

The source-head index does **not** solve production signing-key persistence and rotation across a provider restart. A process restarting with a newly generated key is not the same trusted signer; a real authority must retain signing-key custody independently and securely. The index also does not implement live GoreeCloud Identity authentication, source transaction receipts, actual consent enforcement or independent privacy-rights proof.

A production design needs separately provisioned service identity, protected source key management, an external tamper-resistant monotonic anchor or audited immutable history, native service lifecycle, explicit authorization, disaster-recovery testing and a security review. Keep the PR draft and unmerged until those acceptance gates are met.
