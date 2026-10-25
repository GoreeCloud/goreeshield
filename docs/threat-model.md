# GoreeShield — Initial Threat Model and Trust Boundaries

| Field | Value |
| --- | --- |
| Document type | Engineering threat-model record |
| Status | Initial proposal; validation limited to local synthetic Linux prototype |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Canonical source | `GoreeCloud/goreeshield/docs/threat-model.md` |

## 1. Protected objectives

Protect accurate privacy decisions, per-person authorization, purpose and notice integrity, authorized data rights actions, and truthful outcomes. Avoid creating unnecessary centralized personal-data copies. A privacy tool must not become a privileged new route to another user's private information or a bypass for the owning product's permission model.

## 2. Current prototype boundary

The initial code is a **single-user local Linux CLI**, not a network service. The invoking operating-system user is the only practical trust boundary. It has no GoreeCloud Identity integration, remote adapters, privileged daemon, user database, real network processing, or operational service data. Two synthetic processing-purpose declarations are hard-coded for demonstration.

The local consent file is **not tamper-proof** and must not be treated as a production authorization token. A hostile process running as the same OS user can change it. Local file mode, no-follow opens, and locking are defense-in-depth against accidents and some filesystem attacks, not a substitute for authenticated subject identity, secure key management, full filesystem integrity, or source-system enforcement.

## 3. Data flow and classification

| Data | Current handler | Classification and limits |
| --- | --- | --- |
| Synthetic purpose metadata | Compiled source | Public nonpersonal demonstration data |
| Explicit local choice | Per-user Linux state file | User preference metadata; private file permissions |
| Last decision time and notice version | Per-user Linux state file | Minimal receipt metadata; not append-only evidence |
| User identities and authentication | Not implemented | Not collected |
| Real product records or exported data | Not implemented | Never pass to prototype |
| Network messages, telemetry, third-party destinations | Not implemented | None expected |

## 4. Initial threats and response expectations

| Threat | Potential harm | Prototype control | Production-level gap |
| --- | --- | --- | --- |
| Default allow, missing or stale consent | Processing without informed choice | Fail-closed policy evaluation and notice version check | Product enforcement and policy distribution |
| Silent opt-in after upgrade | Increased data access | No auto-grant path; explicit acknowledgement flag | Fresh install/upgrade testing with real adapters |
| Consent replay or local file tampering | Forged affirmative choice | Strict schema and file permissions | Authenticated, tamper-evident source-bound decisions and revocation order |
| Cross-person authorization | One person's choices affect another | No shared or remote users in prototype | Identity-backed subject binding; per-object checks |
| Symlink or improperly permissioned file | Unauthorized read/write or unsafe path | No-follow state reads, private files, directory checks | Hardened path resolution and privilege separation |
| Concurrent update races | Lost withdrawal or inconsistent state | Exclusive file lock and atomic replacement | Transactional storage, version/conflict semantics |
| Invalid or corrupt on-disk data | Unreliable authorizations | Strict validation; errors prevent processing | Recovery and operational incident response |
| Compromised product adapter | Fraudulent completion, data theft | No real adapters | Product-scoped credentials, signed results, attestation and circuit breakers |
| Deletion request marked complete too early | False user assurance | No deletion functionality | Independently verified per-source completion receipts |
| Logs and troubleshooting output | Exposure of private activity | Minimal synthetic CLI output | Defined logging classification, retention, and redaction |
| Legal hold or backup erasure mismatch | Invalid deletion claims | Not implemented | Source-specific retention/hold policy and tested recovery |
| Unauthorized household monitoring | Privacy harm inside families | Family features not implemented | Granular legitimate roles, consent and age/guardian rules |

## 5. Prioritized engineering gates

### Before a remote API or service is reachable

- Bind each caller and subject to verified GoreeCloud Identity claims; define step-up and token-revocation handling.
- Design narrow service/process privileges, authenticated service-to-service transport, and deny-by-default endpoint authorization.
- Add request freshness, replay defenses, idempotency, subject/purpose binding, and verifiable action receipts.
- Minimize security-event payloads shared with Wardveil; do not transfer credentials from Vault.
- Threat-model secrets injection, service deployment, native operating-system permissions, and supply-chain updates.

### Before real privacy-rights execution

- Preflight exact user and object scope, source authority, necessary processing, retention and legal holds.
- Require source-owned operations and track accepted, pending, partial, failed, denied, and verified-completed separately.
- Use synthetic test data and a nonproduction restore target for deletion and recovery tests.
- Provide clear notices about data already shared, backup lifecycle limitations, and what withdrawal does not undo.

### Before supported-platform or Stable claims

- Complete target-native packaging, installation, lifecycle, runtime and recovery checks.
- Verify upgrade cannot activate optional processing and can handle versioned notices.
- Test permission boundaries, accessibility, GoreeCloud UI integration, data retention and error recovery.
- Perform a separate privacy/security review of the final implementation, not just the proposal.

## 6. Evidence and outstanding decisions

Local Go 1.23 tests, static analysis and build passed in the development environment. A repository-side Linux CI run also passed on the feature branch. These support only the explicitly tested synthetic prototype behaviors; they do not verify operational identity integration, deployment, remote security, privacy law compliance, or deletion outcomes.

Open decisions include native service OS, trust/authorization models, real product adapters, stored data lifetimes, backup handling, and a formal data-flow inventory. Track them under existing GoreeShield GitHub issues rather than treating this document as completed security acceptance.
