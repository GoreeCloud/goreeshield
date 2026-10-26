# GoreeShield — Linux Source-Head Process Boundary

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Linux test-process transport and verification prototype; not a source provider release |
| Base | [Challenged source head contract](source-head-challenge.md) |
| Source | `internal/checkpoint/unix_linux.go` |

## Objective

PR #23 verified signed current-head challenges against in-memory source fixtures. This milestone moves the boundary into a **separate native Linux test process** and a private Unix-domain socket. The process keeps an ephemeral test Ed25519 private key; GoreeShield's client does not import, generate, store or receive the secret key.

The source-process fixture signs a canonical nonce-bound head response. GoreeShield's verification path still requires a separately pinned public key, issuer, exact scope digest, journal digest and current sequence.

## Transport and restrictions

`checkpoint.UnixHeadProvider` implements the existing `HeadProvider` interface. It accepts only an absolute, clean Unix socket path with restrictive filesystem mode, a matching filesystem owner, and a peer UID matching `ExpectedUID` returned by Linux `SO_PEERCRED`. It uses a caller context with a three-second upper bound, encodes a bounded JSON challenge, rejects malformed/unexpected response fields or trailing data, and supplies the response to the existing Ed25519 verifier.

The test-owned subprocess generates its key *inside the child*, binds a private socket under a temporary directory, and responds to challenge requests. The test parent receives the ephemeral public key and signed checkpoint over its test pipe. This **test-only public-key handoff is not authenticated independent production trust provisioning**. The fixture deliberately uses the same Linux UID, despite being another process.

## Development tests

The Linux subprocess test verifies:
- A real separate process responds with a signature that validates against the fixture's public key, scope, journal and nonce.
- A mismatched configured expected peer UID is rejected.
- An accessible-to-others socket file is rejected.
- A stopped provider cannot yield a success response.

Existing verifier tests additionally cover an old signed checkpoint, reused challenge, altered statement, wrong signer, cross-scope substitution, missing provider and concurrent journal writes.

## Security and release limitations

This branch does **not** create a production daemon, start a persistent service, install systemd units, connect a live GoreeCloud application, configure GoreeCloud Identity, provision production public-key trust, deploy signing keys to a separate UID/host, or implement durable source-owned monotonic history.

`SO_PEERCRED` identifies an OS user for one local socket connection; it does not authenticate the source's semantic claims or protect against a compromised process under the allowed UID. Filesystem checks are subject to same-UID path races. A caller that accepts a public key from an unauthenticated peer cannot establish independent trust. Production needs an independently provisioned source UID and key, audited key rotation/revocation, authenticated source-owned history, scoped authorization, bounded concurrency and lifecycle controls.

No private production keys, real personal records, live privacy-rights fulfillment, independent signed receipt or production availability guarantee is provided. Keep all prototype PRs draft until security review and live source integration.
