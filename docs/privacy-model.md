# GoreeShield — Privacy and Security Requirements

| Field | Value |
| --- | --- |
| Type | Proposed product requirements |
| Status | Draft; no compliance or implementation claim |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Source | `GoreeCloud/goreeshield/docs/privacy-model.md` |

## Privacy behavior

1. All optional privacy-invasive capabilities start **disabled** on installation, migration, and upgrades. No prechecked or bundled opt-in.
2. Inform people what data is processed, for what purpose, by whom, where it goes, the retention and sharing effects, and what withdrawal means before they enable it.
3. Record affirmative, scoped choices with a policy/notice version and enable withdrawal without unnecessary friction.
4. Withdrawal stops **future** optional processing; separately disclose actions still required for data already processed or shared.
5. Distinguish necessary product functions from optional purposes. Consent is not a substitute for other lawful grounds, secure implementation, or another person's privacy rights.
6. Verify identity and authorization for rights requests; evaluate each requested action and its exact data subject and source scope.
7. Do not claim deletion, export, or correction completed until the source confirms the result with adequate evidence.
8. Handle legal holds, legitimate record preservation, inaccessible adapters, partial outcomes, and errors honestly.
9. Minimize centralized personal-data storage, log contents, analytics, and privacy receipts; define bounded retention.
10. Protect household member boundaries. No covert surveillance or unauthorized parent-to-adult access.
11. Never collect or expose credentials, secret payloads, or production personal data in public code, examples, fixtures, or logs.
12. Measure claims against actual implementation and target OS tests, not UI copy or planned specifications.

## Risk and abuse scenarios

Unauthorized rights requests; forged consent receipts; consent replay; broken object-level authorization; cross-family data leaks; malicious product adapters; fake deletion results; accidental overbroad retention purge; private data in diagnostics; silent re-enablement of tracking after upgrades; centralized governance database compromise.

Each scenario requires proportionate mitigations and security/privacy review before production use.

## Acceptance tests for first service

- Optional purpose remains off on fresh install, upgrade, and failed sync.
- Choice updates require verified identity and narrowly scoped authorization.
- Withdrawal takes effect and subsequent events cannot bypass revocation.
- A user cannot read or change another person's choices or workflow records.
- Duplicate, stale, forged, delayed, and failed adapter responses cannot create false completion receipts.
- Requests remain pending or partial until actual source verification succeeds.
- Retention and exception behavior uses synthetic data and authorized nonproduction recovery environments.
- Logs do not contain unnecessary private content or identifiers.

## Scope

This specification is a technical starting point aligned with GoreeCloud's privacy-first direction, not an assertion of regulatory certification or complete data-rights support. Jurisdiction-specific requirements and product-specific data categories must be verified before implementation.

## November 7, 2026 — bounded local replay guard (draft)

The synthetic Linux Choices development chain introduces one **non-authoritative**, per-purpose ordering check: a new local `State.Grant` must carry a decision timestamp **strictly later** than that purpose's last stored decision timestamp. Stale or equal-time grants fail without changing a previous withdrawal or its revision. An unrelated purpose remains independent. `State.Withdraw` continues to apply even if the local clock goes backward; it now also preserves the previous per-purpose `ChangedAt` value as a **logical ordering floor** instead of recording a backward-moving timestamp. This floor is not an attested physical event time. Negative tests exercise stale re-grant after withdrawal, equal timestamps, no partial state mutation, fresh re-grant, repeated skewed withdrawal and revision integrity.

This is deliberately **not** replay-resistant consent. Timestamps can be forged or arrive out of order, clocks can skew, and the local synthetic state lacks trusted operation IDs, authenticated subjects, durable owner fencing or verified downstream withdrawal. A re-grant timestamp newer than a skewed withdrawal may still be accepted. Production integration requires source-authenticated, monotonic operation/revision fencing and confirmation of actual owner-side effects. No real optional processing or product integration is authorized by these local checks.
