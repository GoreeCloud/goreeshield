# GoreeShield — Source Signer Continuity Boundary

**Status:** Linux-only synthetic development fixture; not a production key-management service.

## New enforcement

The source-owned `Responder` now requires an independently provisioned `PinnedPublicKey` before issuing a signed current-head response. It compares the active Ed25519 signer's public key against that pin and refuses missing or substituted keys with `ErrSignerKeyMismatch`. Existing checks still bind the source issuer, key ID, scope, journal digest, sequence and fresh challenge.

This pin is a separate input from the mutable source-head index and the key loader. Passing a newly generated key's own public key as the pin without independent authorization would defeat the intended trust boundary.

## Linux development tests

Unit tests verify refusal of a missing pin, swapped signer, missing private signer, wrong issuer/key ID and malformed challenge.

The subprocess test uses a **test-only** temporary Ed25519 seed file with restrictive ownership and mode checks. It starts the owning-source test process, verifies a signature against the pinned public key, advances the durable head, restarts the process using the same test seed, and verifies that the same key can authenticate the new head. Replacing the test seed while retaining the old pin must fail both the source-side key check and the independent verifier. Tests also reject a broadly readable or symlinked seed fixture.

The private-key file loader lives entirely in a `*_test.go` file. It is not a supported production credential store and must not be reused as one. No private keys, test seeds or credentials are committed to GitHub.

## Production prerequisites

A real owning source must obtain `crypto.Signer` from a separately administered service or protected key manager, and obtain its public-key pin from an authenticated, reviewable trust-provisioning channel. Key rotation and revocation need an explicit, auditable policy; restart must neither silently generate a new trusted identity nor silently trust a changed signing key.

The CI child and test parent still run as the same Linux UID. Test-owned file permissions cannot establish independent production key custody, resist an authorized same-UID actor, guarantee key availability, or secure the whole host. No source production daemon, GoreeCloud Identity authorization, actual transaction outcome, rights receipt, HSM, power-loss test or deployment is part of this PR.

Keep this PR draft and unmerged pending security review and a genuinely independent source-owner integration.
