# GoreeShield — Durable Source-Head Responder

**Status:** Linux synthetic development fixture; no production signing service.

PR #25 persists a monotonic source head, while PR #24 responds using a fixed in-memory statement. The new `internal/sourcehead.Responder` reads the latest durable source-owned checkpoint on each nonce challenge and signs its canonical payload with an externally supplied `crypto.Signer`. It does not generate, store, or export signing keys.

The responder validates nonempty scope and nonce, checks its configured issuer and key ID against the persisted statement, verifies the signing result against the signer's public key, and fails closed if the key or source history is unavailable.

Tests cover sequence advances, old signed checkpoints, invalid requests and missing signers. A Linux subprocess fixture serves a private Unix socket using an ephemeral key retained in child memory. The head sequence survives a provider restart, but the test signing key does not; responses signed by a new key must be rejected under the previously pinned key.

**Limitations:** Test parent and child share one UID and the same fixture directory. The public-key handoff is not authenticated trust provisioning. This is not persistent signer custody, production key rotation, a real source application, GoreeCloud Identity authorization, a tamper-resistant source head, or evidence of actual privacy-rights fulfillment.

Production release requires independently controlled service identity, protected signing-key continuity and revocation, scoped caller authorization, validated source-owned transactions, real durability/fault testing, native service lifecycle and independent security review. Remain draft and unmerged.
