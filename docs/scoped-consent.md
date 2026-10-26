# GoreeShield — Synthetic Subject and Tenant Isolation

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Development-only Linux fixture; no real multitenant isolation or Identity connection |
| Canonical source | `GoreeCloud/goreeshield/docs/scoped-consent.md` |

## Objective

Demonstrate that the privacy choices, revisions, synthetic events and in-memory idempotency records associated with one **verified synthetic tenant/subject pair** are not automatically reused for another pair. The previous single-user prototype had only a Linux OS-user-wide choice file. This experimental extension is opt-in through separate fixture types and does not modify or migrate the original CLI or daemon.

## Components

- `authz.Claims.Tenant` and `Guard.CheckTenant`: require issuer, audience, exact subject, exact tenant, scope and active credential time window from a configured trusted verifier. Existing `Guard.Check` remains a single-subject, tenant-unaware compatibility fixture and is **not acceptable** for future multitenant services.
- `scoped.Directory`: derive a distinct private local choice file for every synthetic tenant and subject, using a domain-separated SHA-256 digest of the identifiers as a filename. Validate the restricted fixture identifiers (3–64 lowercase ASCII letters, digits, underscore or hyphen) and reject symlinked, publicly readable or wrong-owner root directories.
- `synthetic.ScopedProcessor`: authorize first; then resolve the scoped choice file and retain a separate synthetic `Processor` and ephemeral idempotency map for each pair.
- Tests: scope collisions, traversal, unsafe directories, exact tenant/subject checks, same operation ID used independently in different tenants, independent withdrawal, and no storage selection if the verifier is unavailable.

## Trust and threat limits

This is **logical separation for synthetic fixtures**, not a production security control:

1. All files and processes continue to share **one Linux operating-system user**. Compromised same-UID processes can access all scoped records. Use properly isolated OS identities, database row permissions, or robust keyed encryption when a production threat model requires them.
2. A deterministic filename hash is not encryption, a secret, or a guarantee of unlinkability. The identifier space is small and guessable.
3. This implementation currently uses `MkdirAll`, `Lstat`, and normal path operations rather than a fully race-free directory-fd/openat2 sandbox. It checks symlinked ancestors and ownership but **does not** claim defense against a concurrent malicious same-UID filesystem actor.
4. Files share the `.choices.lock` in the scoped root: this serializes cooperating local processing across pairs and can create contention. It is not a horizontally scalable or distributed lock.
5. The verifier is an interface only. The synthetic tests supply a fixture identity; no OIDC discovery, JWT/JWKS verification, production secret, authenticated session, user account or service endpoint has been connected.
6. Subject and tenant identifiers are intentionally restricted synthetic ASCII fixtures. Real external subjects may have different legal characters and opaque forms; do not use this validation or storage representation as an actual Identity subject schema.
7. The in-memory processor map, counters, and operation-ID history are lost across process restarts, and the original unscoped Linux CLI/service still exist. No global enforcement, durable idempotency, signed receipt, source-owned processing or verified rights fulfillment exists.

## Processing invariants demonstrated

A synthetic request must have a verified tenant and subject and the exact `privacy.synthetic.execute` scope before a choice file is even resolved. A request evaluated under tenant A cannot execute against tenant B's store merely by changing the tenant or subject string.

A choice granted for one pair remains absent for every other pair until explicitly granted there. Revoking a choice in tenant A does not revoke an unrelated tenant B decision. The same operation ID may be used in both pairs without cross-pair deduplication because each maintains its own synthetic processor table.

Every processor still checks the consent revision at the actual local synthetic action boundary and never treats a historical duplicate result as fresh permission.

## Next production gates

Choose and review the actual GoreeCloud Identity token/session contract, signature/key rotation, delegation, tenancy and audience trust. Move the persisted choices to a transactional per-person and per-purpose store with independent authorization and durable idempotency. Enforce access control in the real source product; verify revocation propagation with signed or independently reconcilable outcomes where appropriate. Complete OS-native service installation and shutdown tests, data recovery, key management, privacy review, GC UI-native applications and representative platform acceptance.

**Do not describe this experiment as a multitenant hosted service, a production integration, secure anonymization, or verified compliance.**
