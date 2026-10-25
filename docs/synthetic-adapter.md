# GoreeShield — Synthetic Downstream Consent Enforcement

| Field | Value |
| --- | --- |
| Document type | Engineering verification and API-boundary note |
| Status | Linux-only synthetic proof of concept; not a production integration |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Source | `GoreeCloud/goreeshield/docs/synthetic-adapter.md` |

## Why this increment exists

A dashboard showing **granted** or **withdrawn** is not proof that a downstream processor obeys the setting. This increment introduces a deliberately narrow demonstration: a local, synthetic optional processor that must verify a current purpose-specific grant before incrementing an in-memory counter. It never collects, exports or transmits user content.

## Components

- `internal/choices/serialized_linux.go` — a local, exclusive file-lock-protected conditional action gate. Missing purpose, missing grant, mismatched notice version, revoked choice, unreadable state and malformed input do **not** cause an action.
- `internal/synthetic/adapter_linux.go` — a synthetic adapter that has only an in-memory counter as its action and validates the current local choice for every attempted synthetic event.
- `cmd/goreeshieldsim` — one-shot native Linux CLI to attempt a synthetic operation using the same state as `goreeshield` and `goreeshieldd`.
- Unit tests — default denial, purpose isolation, affirmative grants, withdrawal, malformed-state refusal, missing sinks and the one-shot CLI flow. Concurrent withdrawal ordering is an important additional acceptance test.

## Local demonstration (nonproduction OS account)

```sh
go test -race ./...
go vet ./...
GOOS=linux GOARCH=amd64 go build ./cmd/...

# Both commands use the SAME Linux user and local choice state.
go run ./cmd/goreeshieldsim --purpose demo-local-metrics
go run ./cmd/goreeshield notice --purpose demo-local-metrics
go run ./cmd/goreeshield grant --purpose demo-local-metrics --notice-version 1 --acknowledge
go run ./cmd/goreeshieldsim --purpose demo-local-metrics
go run ./cmd/goreeshield withdraw --purpose demo-local-metrics
go run ./cmd/goreeshieldsim --purpose demo-local-metrics
```

The expected abstract result is `DENIED`, then `SYNTHETIC ACTION`, then `DENIED`. The action is only an in-memory counter during the current process; it does not start metrics, surveillance, or network processing.

## Isolation, ordering and failure behavior

`Store.WithAllowed` evaluates the grant under the existing private per-user file lock and holds that lock through the short synthetic action. This creates a serialization point against **cooperating** local choice writers. It does not solve remote or cross-machine revocation races, same-user malicious processes, adapter bypass, or asynchronous downstream work. A long-running action holding a lock would delay local withdrawal, so external IO and recursive Store calls are explicitly forbidden in the callback. Do not generalize this method to network-connected processors.

A synthetic operation completed before withdrawal is not undone by withdrawal; future attempts after withdrawal are denied. Read failures fail closed with an error and no synthetic action. The one-shot counter is not persisted, is not a durable or signed receipt, and is not a processing log.

## Production contract requirements — not implemented

1. GoreeCloud Identity must attest caller, subject, audience and scope for an authorized request. The local OS username is not a multi-user identity contract.
2. The owning product must enforce the choice at the actual processing boundary, with explicit purpose and notice version and no cached permission that silently survives revocation.
3. Requests must have bounded lifetime, replay defenses, idempotency keys, ordered revisions or fencing tokens, source-specific receipts, and explicit failed/partial/stale states.
4. The source and GoreeShield must reconcile the actual enforcement outcome before representing revocation propagation as complete.
5. Third-party sharing and remote adapter behavior need separate data-flow, retention and legal-hold review.
6. Independent concurrency, fault-injection, recovery, security, OS packaging, target-native service lifecycle and accessibility testing remain required.

This synthetic adapter does not constitute actual cross-application enforcement or verified privacy compliance.
