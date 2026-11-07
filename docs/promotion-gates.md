# GoreeShield — Safe Promotion Gates

**Checkpoint:** 2026-11-07. Engineering review guidance, not an approved release.

## Branch topology

I require dependency-ordered review of the active draft chain. It starts with foundation PR #1, includes the local Choices prototype (#6) and signer-continuity work (#27), and now extends through serialization (#29), schema read validation (#30), fail-closed schema mutations (#32), and corrupted-state mutation refusal ([draft #34](https://github.com/GoreeCloud/goreeshield/pull/34)). These are representative milestones, **not a complete list of every intermediate draft**. Draft #34 targets #32's feature branch, **not `main`**; its Linux prototype checks passed at head `3f77af277e78dba3079df42f0ab826eb52f4aef3` ([CI run](https://github.com/GoreeCloud/goreeshield/actions/runs/42404075775)). Green stacked-head tests do **not** imply any of the layers have reached `main` or provide production enforcement. Merging or rebasing out of order may obscure the tested diff and its invariants.

Before promoting a layer, confirm its actual base/head commits match the reviewed code. Review and merge one dependency at a time, let GitHub recalculate remaining diffs, and rerun CI on the resulting target. Never convert uncertain outcomes to success through documentation.

## GoreeShield privacy decision promotion matrix

I require an owning source service to enforce privacy decisions independently. GoreeShield is the **privacy** authority; Wardveil is the **cybersecurity** authority; Mesh coordinates authorized transport and must not turn either system's status into a universal permission. Before allowing any real personal-data flow, independently demonstrate these negative and outcome tests:

| Scenario | Required outcome | Evidence boundary |
| --- | --- | --- |
| Missing, withdrawn or expired optional consent | Deny the optional purpose-specific processing or disclosure | Verify both GoreeShield's decision **and source-owned enforcement**, not an in-memory fixture |
| Wrong tenant, person, purpose, recipient or action | Deny; do not disclose cross-subject data | Authenticated actual identity and scoped, versioned decisions |
| Unknown schema, invalid signature, replay or stale revision | Deny until a compatible verified decision is obtained | Validate at reads **and** writes; test mutation rejection and persisted revision integrity |
| Privacy provider timeout, restart or uncertain result | No automatic consent resurrection or positive transport acknowledgment | Re-read current decision and source outcome after recovery |
| Authorized essential processing without optional consent requirement | Continue only under its documented approved basis | Avoid blanket denial and invented consent requirements |
| Denial telemetry to Wardveil | Do not classify ordinary user withdrawal as an attack | No raw preferences, private content or personal identifiers in security events by default |
| Mesh reports successful delivery | Do not interpret it as fulfilled withdrawal, deletion, export or rights request | Source system must provide independently observed operation outcome |

Use only fictional subjects and synthetic data in the current drafts. These cases describe **acceptance requirements**, not working real-data integrations or guarantees. Actual consent notices, data inventories, policy/legal bases, key lifecycle, retention and user-accessible GC UI flows require independent review.

## Promotion checklist

| Gate | Evidence needed | Current status |
| --- | --- | --- |
| Linux build and static analysis | Declared Go toolchain, `go vet ./...`, target build | Prototype CI evidence only |
| Races and behavior | `go test -race ./...` including denial, withdrawal and replay | Verified on draft tip; rerun after integration |
| Restart safety | Missing/corrupt/partial journal history and cross-process recovery tests | Synthetic fixtures, no production SLA |
| Authorization | Independently verified GoreeCloud Identity, tenant/subject grants and revocation | **Blocking** — fixtures only |
| Source enforcement | Actual product adapter with source-observed outcomes and fail-closed behavior | **Blocking** — synthetic only |
| Signer and secrets | Pinned trust, rotation/revocation, backups and incident recovery | **Blocking** — test continuity is not deployed key management |
| Native operations | Install, upgrade, backup, restore, permissions and supported OS | **Blocking** — no released service |
| Privacy evidence | Data inventory, minimization, rights/retention and truthful receipts | **Blocking** — no compliance certification |
| GC UI | Real native interface, accessibility and device acceptance | **Blocking** — no accepted client |

## Local verification cautions

- Linux Unix-socket tests require filesystem/socket permissions and a **short temporary directory**. A deeply nested `TMPDIR` can exceed the Unix socket path limit. An environment failure is not automatically a product failure.
- Passing a stacked tip does not prove `main` contains the functionality. The default branch currently contains no Go implementation.
- Never include secrets, real personal data, production infrastructure or private incident evidence in issues, PRs or test artifacts.

## Decision record

For every merged layer, record the PR and exact commits, CI run/date, scope, residual risks, rollback, owner decision and next blocked gate. Leave unverified gates blocked. No release declaration until real source/identity integrations and native operations obtain independent acceptance evidence.
