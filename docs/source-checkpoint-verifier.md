# GoreeShield — Source-Owned Checkpoint Verification v1

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Status | Verification-only synthetic development contract |
| Location | `internal/checkpoint`, `internal/reconcile/source_checkpoint_linux.go` |
| Scope | Local Linux fixture; no live source provider, credentials or deployment |

## Purpose

The current synthetic journal, initialization marker, and checksum witness are stored under one Linux user. A party able to roll back those files together can evade their local consistency checks. This proposal defines the **consumer/verifier** side of a future source-owned checkpoint: an owning application signs a journal digest and an increasing checkpoint sequence using an Ed25519 private key that does not reside in GoreeShield.

The package contains **no signing private key** or trusted public-key store. Tests generate ephemeral fixture keys at runtime only.

## Canonical v1 contract

Each signed statement contains a domain-separated version, an ASCII issuer and key ID, an opaque scope digest, a strictly positive source sequence, and the SHA-256 digest of the exact journal bytes. Its `SigningPayload` encoding uses an explicit domain tag and big-endian length prefixes. Signatures cover the entire encoded statement.

`checkpoint.Verify` requires a separately pinned issuer, key ID and Ed25519 public key, plus an expected scope, expected journal checksum and a minimum accepted source sequence. It refuses malformed statements, wrong signers, bad signatures, scope substitution, journal changes and checkpoints older than that independently supplied minimum.

The `checkpoint.ScopeDigest` function binds exact tenant and subject strings using domain separation and length-prefix encoding. This digest is **not anonymization**: short or guessable identifiers may be inferred by dictionary attacks.

## Linux integration fixture

`VerifySyntheticJournalCheckpoint` obtains the scoped store, acquires the cooperating source-journal lock, validates the marker and local witness, hashes the current journal bytes, and verifies the source statement against that digest and exact scope. It is read-only. Tests sign with an ephemeral source fixture key, confirm that a matching statement is accepted, and verify that a later synthetic journal write invalidates the earlier signed state.

## Mandatory trust requirements before production

- The owning GoreeCloud application must generate and protect its private signing key separately. GoreeShield must have an authenticated, independently provisioned trust anchor and an audited key rotation/revocation policy.
- A **current sequence floor** must come from an independently trusted, durable owning-source head. A caller using the local journal to choose this number cannot prevent replay of an older signed snapshot.
- A valid signature is not itself proof of consent, deletion, processing purpose, privacy-rights completion, or legal compliance. Transaction semantics and verified source outcomes need separate evidence.
- An offline or unreachable source means the checkpoint cannot be refreshed; do not silently treat the locally cached signed statement as current.
- Key compromise, replacement of the pinned public key, denial of service, hardware power loss, multi-host consensus, and actual live-source integration remain unsolved.

## Tests and status

The verification package has tests for valid signatures, lower signed sequence, cross-scope and journal substitution, wrong issuer/key, altered payload, truncated signature, malformed schema and missing trusted sequence. The Linux integration test binds a signed fixture statement to the current source journal and rejects it after a subsequent journal write.

This is an **interface and verification proof of concept**, not deployment of a new independent attestation authority. All earlier PRs remain draft, and a real GoreeCloud Identity-bound source-provider integration remains a separate acceptance gate.
