# GoreeShield — Source-Owned Synthetic Outcome and Reconciliation Contract

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Type | Engineering specification and nonproduction test evidence |
| Status | Linux-only, in-memory synthetic proof of concept |
| Canonical source | `GoreeCloud/goreeshield/docs/source-reconciliation.md` |

## Objective

GoreeShield must **never equate a submitted request with an independently verified result**. A privacy choice may be accepted locally while a source product has not responded, an acknowledgement is lost, or the source itself has not performed the requested action.

This increment models that distinction through a synthetic source and a separate coordinator:

- `internal/reconcile/MockSource` owns an ephemeral map of operation IDs and their actual **synthetic counter** outcomes. It checks current purpose-specific revision under the local Linux consent lock before counting an action.
- `internal/reconcile/Coordinator.Dispatch` checks exact tenant, subject and `privacy.synthetic.execute` scope, sends a versioned operation to the mock source, and returns **`pending_source_readback`** even when submission succeeds.
- `internal/reconcile/Coordinator.Reconcile` separately checks the exact `privacy.synthetic.inspect` scope, retrieves the source observation by tenant, subject and operation ID, and compares its entire request to the expected envelope before reporting a source-observed result.
- Neither operation produces or consumes personal information. There are no network transports, production tokens, externally connected products, databases or deletion operations.

## Protocol status semantics

| Status | Meaning | Claim allowed |
| --- | --- | --- |
| `pending_source_readback` | Submission returned without error; no inspection yet | Request submitted to **mock source only** |
| `unresolved` | Missing observation, unavailable source, or failed request (with an error) | Nothing about downstream completion |
| `source_observed_applied` | Separate inspection finds the exact operation recorded as applied | A synthetic in-memory counter action was observed |
| `source_observed_denied` | Separate inspection finds the exact operation recorded as denied | The synthetic source recorded denial |
| `conflict` | Same ID maps to different request content | No completion assertion |

Only an affirmative independent source inspection upgrades a pending observation. Even `source_observed_applied` is **not** a cryptographically signed receipt, source-independent attestation, durable proof, legal compliance determination, or verified fulfillment of data rights.

The mock source uses a separate operation journal from the coordinator. A duplicate operation ID with the *same* request never increments the source counter again; an ID reused with different request fields returns an error and cannot be represented as applied for that altered request. Its records are scoped to a synthetic tenant/subject pair, not globally shared across households.

## Failure modes

- **Lost acknowledgement:** The coordinator may know only that submission was attempted. A later separately scoped source inspection discovers whether the mock source recorded the operation. Retrying the identical operation ID does not repeat a synthetic action.
- **Missing observation:** Reconciliation stays `unresolved`. Absence is neither denial nor successful application.
- **Withdrawal:** The source checks the current local consent revision at the actual synthetic action boundary. Previously executed actions are not reversed by withdrawal; fresh attempts after withdrawal are denied.
- **Re-grant:** A prior revision cannot be reused after withdrawal/re-grant. A new affirmative request must carry the new revision.
- **Unauthorized caller:** Both dispatch and inspection require exact tenant, subject, issuer, audience, time window and purpose-specific scope from an explicitly configured verifier. No production verifier is present, so default configuration denies.
- **Simultaneous retries:** Source-owned records are serialized in-process; duplicate delivery results in one counter increment at most for each scope and operation ID.
- **History exhaustion:** A source stops accepting new operation IDs after 1,024 entries per synthetic scope. Old IDs are not evicted and replayed as new.
- **Cancellation / invalid inputs / unavailable source:** They remain unconfirmed. The coordinator cannot infer completion from an error.

## Security and operational limitations

The mock source and coordinator run **in the same Linux process** when instantiated for tests. Both are memory-only, and all synthetic consent files remain accessible to a compromised same-UID process. The mock source is not an actual source application or durable evidence authority. The scope-specific operation maps and counts are lost when a process exits; thus exactly-once behavior is **not guaranteed across restarts or independent replicas**.

Current tenant/subject paths use deterministic hashes for local convenience, **not encryption or secure anonymization**. Filesystem checks are not hardened against all same-user races. The previous unscoped daemon and CLI remain available and do not implement this reconciliation contract.

Do not expose these fixture interfaces as a network service, treat the in-memory map as a privacy audit ledger, or report a privacy rights request as fulfilled on the strength of these outcomes.

## Acceptance gates before real integration

1. **GoreeCloud Identity:** Implement and independently test a true credential verifier, subject/tenant claims, issuer/audience binding, token expiry and revocation, delegation policy, scopes and service authentication.
2. **Owning product:** Enforce privacy choices at the real processing boundary and provide a source-owned, immutable transaction or event identifier linked to the exact operation.
3. **Persistence:** Use a durable idempotency/outbox mechanism with transactional source effects and revision fencing across crashes, retries, replicas and recovery.
4. **Evidence:** Define which source attestations count as accepted, queued, applied, verified, partial, failed or revoked, along with proof freshness, authorization, authenticity and retention.
5. **Security:** Threat-model forged receipts, cross-tenant disclosure, rollback, clock skew, tampering, exhaustion, delayed acknowledgements, replay and unsafe audit logging.
6. **Release:** Verify OS-native packaging, lifecycle, target-specific UI and accessibility, upgrades, rollback and representative-device security testing.

## Development verification

GitHub Actions should check committed `gofmt`, `go test -race ./...`, `go vet ./...`, and native Linux builds. Tests require one observed action after duplicate retries, no cross-tenant leakage, exact request matching, withdrawal denial, separate read privileges, unresolved missing observations and a bounded source journal.

These checks can validate **synthetic protocol behavior only**, not source-product interoperability or actual privacy compliance.
