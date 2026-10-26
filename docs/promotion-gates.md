# GoreeShield — Safe Promotion Gates

**Checkpoint:** 2026-10-26. Engineering review guidance, not an approved release.

## Branch topology

The active feature work is a series of dependent draft pull requests starting at foundation PR #1 and proceeding to signer-continuity PR #27. Each draft's base is its predecessor. Merging or rebasing out of order may obscure the tested diff and which invariants are actually present on `main`.

Before promoting a layer, confirm its actual base/head commits match the reviewed code. Review and merge one dependency at a time, let GitHub recalculate remaining diffs, and rerun CI on the resulting target. Never convert uncertain outcomes to success through documentation.

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
