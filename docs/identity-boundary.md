# GoreeShield — Identity Boundary and Legacy Choice Safety

| Field | Value |
| --- | --- |
| Type | Engineering security contract and migration note |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Experimental local code only; no production Identity integration |
| Canonical source | `GoreeCloud/goreeshield/docs/identity-boundary.md` |

## 1. Consent migration safety correction

A prior experimental choice model added a **global** positive revision but did not bind each choice record to the revision when the person actually made that decision. An old JSON grant with no choice revision could become enabled after the user edited a completely different purpose, because the shared global revision became positive.

The corrected experimental model includes `Choice.Revision`, written on each explicit grant and withdrawal. A choice is allowed **only** if its own revision is positive, not later than the current global revision, the notice version is current, and that exact purpose has an affirmative recorded decision.

Existing JSON files that contain a global revision but lack per-choice revisions are supported for reading without silently becoming authorized. Their old grants stay **off** until the person affirmatively grants the affected purpose again. Editing or withdrawing an unrelated purpose does not revive the legacy grant.

This is an intentional fail-closed migration. The schema stays version 1 for this development prototype, with an optional `revision` member on each choice. Real upgrades must separately test backup/restore, version pinning, rollback, legacy UX messaging, recovery, and authorized user notices before any production migration. The current code does not provide a signed consent history or protect state against a malicious same-user process.

## 2. Provisional GoreeCloud Identity trust boundary

`internal/authz/Guard` and `Verifier` define only **what needs to be established** by an eventual trusted identity integration:

- The caller credential is verified by an actually configured provider, not inferred from request fields.
- Issuer and service audience match pinned expected values.
- The verified subject matches the exact subject being accessed; no implicit household/administrator delegation.
- The exact required scope is present, without substring or wildcard matching.
- Credential time limits have not expired and are not in the future.
- A missing verifier, empty credential, canceled request, or invalid claims fail closed.

No GoreeCloud Identity verifier, OIDC issuer, JWT/JWKS implementation, user account, token endpoint, actual signing key, or deployment credential is provided. The verification tests use `synthetic-fixture` and deliberately synthetic issuer/audience/subject identifiers. **Those literals are not credentials for any real service.**

## 3. Synthetic authorization wrapper

`internal/synthetic/GuardedProcessor` requires a successful `Guard.Check` and a matching configured local synthetic subject before calling the local in-memory processor. Tests demonstrate that mismatched subject claims and missing credentials never increment the synthetic counter.

The underlying prototype still exposes an ungated `Processor` and its original local CLI for development. This wrapper is **not** a systemic access-control migration, cannot isolate different people under one Linux OS account, and must never be wired to a remote endpoint as if it represented complete authentication or authorization.

## 4. Production acceptance gates

Before connecting any actual GoreeCloud user or product:

1. Verify the existing GoreeCloud Identity authority, supported token/session types, real issuer and audience, key trust/rotation, token lifetime and revocation behavior.
2. Implement an identity-provider-specific verifier behind the abstract interface, with algorithm allowlists, signature validation, key selection, token-type validation, correct scopes and service-to-service caller binding.
3. Threat-model subject and tenant isolation, owner/delegated authority, household privacy, token replay, partial failure, stale cache, rate limiting and secret/log exposure.
4. Design source-owned data records and truly isolated choice revisions per subject and purpose. A single Linux user and one global revision are not sufficient.
5. Make every real product adapter enforce the effective choice at the actual processing boundary. A dashboard, middleware guard or cached permit cannot replace source enforcement.
6. Define revocation acknowledgements, durable idempotency, signed source evidence where appropriate, retention policy, incident review and independent verification for rights actions.
7. Test actual native service packaging, authorization boundaries, upgrades and rollback without exposing production data or claiming a Stable release.

The requirement to authenticate, authorize, and bind a claim to a person is **not satisfied merely because this repository contains an interface and fixture tests**.

## 5. Verification scope

CI should execute strict Go formatting, race-detector tests, static analysis and native Linux builds. Tests in `internal/choices`, `internal/authz` and `internal/synthetic` cover the local migration invariant, claim mismatch denial and synthetic wrapper ordering. These checks do **not** verify GoreeCloud Identity integration, legal compliance, service installation or live source operations.
