# GoreeShield — Challenged Source Head Verification v1

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Status | Synthetic verification contract; not a live source attestation service |
| Base | [Source-owned checkpoint verifier](source-checkpoint-verifier.md) |
| Code | `internal/checkpoint/head.go`, `internal/reconcile/source_head_linux.go` |

## Problem and objective

A correctly signed checkpoint can still be old. PR #22 therefore needed an independently trusted minimum source sequence. Supplying that minimum as a caller parameter is useful for testing, but it is **not a retrieval or authentication protocol**.

This milestone defines a source-owned, challenge-response head interface. The verifier generates 32 random bytes for each query and requests a signed source-head statement bound to that nonce and the requested scope. A replay of a signed response to another challenge fails, even if its journal digest and sequence were otherwise plausible.

## Contract

The `checkpoint.HeadProvider` abstraction accepts a `HeadRequest` with a scope digest and per-request nonce and returns a `HeadProof`: the source's versioned statement, the echoed nonce, and an Ed25519 signature covering a dedicated head-response domain, nonce, and canonical statement encoding.

The verifier checks an existing checkpoint signature against a separately pinned source issuer, key ID and public key. It then verifies the challenged head's nonce, signer, scope, exact journal digest, and source sequence. The old checkpoint is accepted only when its sequence **equals** the newly authenticated head sequence. The source must maintain a trustworthy monotonic current head; a signature cannot establish this on its own.

Offline, absent, canceled, or otherwise untrusted providers fail closed. There is no automatic fallback to a cached source head.

## Linux synthetic journal integration

`VerifySyntheticJournalWithFreshHead` checks local journal integrity and reads its digest while holding GoreeShield's existing journal lock. It releases that lock before invoking the independent head provider to avoid blocking a remote service or deadlocking with callbacks. After verification, it reacquires the lock and checks the journal digest a second time. Changes during the provider call invalidate the result.

The returned success is **point-in-time verification**, not a transactional promise that the source cannot change immediately afterward. Separate consistency and transaction mechanisms are required to make consequential source actions.

## Adversarial tests

- Accept a signature and fresh source head produced by an ephemeral, synthetic test signer.
- Reject a source head with a newer sequence than the previous signed checkpoint.
- Reject a response containing a previously signed challenge nonce.
- Reject cross-scope and different-journal heads, invalid signer keys, altered local checkpoints, and an unavailable or canceled head provider.
- In the Linux fixture, detect a concurrent journal update performed inside the provider call; the journal lock is not held across that callback.

## Trust and deployment boundaries

No production source endpoint, transport protocol, certificate policy, API token, key manager, external sequence registry, or GoreeCloud Identity verifier has been implemented. Test signers use ephemeral in-memory keys and synthetic journals. An actual source provider must authenticate callers appropriately, protect signing keys independently, maintain a real monotonic source head, restrict per-scope disclosure, and provide bounded/authorized access. Do not place signing private keys in GoreeShield.

This design does not guarantee source availability, resist compromise of the source signing key, prevent a malicious source from lying about its head, establish privacy-rights fulfillment, or prove hardware power-loss durability. It is not production-ready or independent legal evidence. All earlier proof-of-concept PRs remain open and unmerged.
