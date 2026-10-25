# GoreeShield — Native Linux Service Proof of Concept

| Field | Value |
| --- | --- |
| Document type | Engineering implementation and acceptance note |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Development proof of concept; no production or Stable acceptance |
| Repository | `GoreeCloud/goreeshield` |

## Scope and boundaries

The prototype now has three Linux-native Go executables:

- `goreeshield` — read/write **local synthetic** privacy choices (no real data processing).
- `goreeshieldd` — a same-user Unix socket service exposing only processing-purpose notices, status and individual decision checks.
- `goreeshieldctl` — a read-only client of the native Unix socket service.

No TCP listener, web wrapper, remote API, GoreeCloud Identity session, Vault credential, Wardveil action, real product adapter, or personal information collection is implemented. A Linux program and unit template alone **do not** prove a supported or production-ready Linux service. The `systemd --user` unit file is an example for authorized nonproduction evaluation only and is **not installed or started automatically**.

## Local development

```sh
go test -race ./...
go vet ./...
go build ./cmd/...

# In a dedicated test user account, start the service in one terminal:
go run ./cmd/goreeshieldd

# In another terminal for the SAME OS user:
go run ./cmd/goreeshieldctl status
go run ./cmd/goreeshieldctl notice --purpose demo-local-metrics
go run ./cmd/goreeshieldctl check --purpose demo-local-metrics

# Choice changes remain local and explicit:
go run ./cmd/goreeshield grant --purpose demo-local-metrics --notice-version 1 --acknowledge
go run ./cmd/goreeshieldctl check --purpose demo-local-metrics
go run ./cmd/goreeshield withdraw --purpose demo-local-metrics
```

Use only synthetic data. Do not execute commands against a privileged or production account without verifying paths and isolation. Acknowledgement demonstrates a local decision, not identity-proofed consent or permitted processing.

## Security design of this increment

1. The server's socket resides in the same private Linux state directory as `choices.json`; it is chmod 0600.
2. Linux `SO_PEERCRED` verifies the connecting process UID matches the daemon's effective UID.
3. Requests are capped at 1024 bytes, must be newline-delimited JSON, and are bounded by connection deadlines.
4. The API implements read-only `status`, `notice`, and `check`. Unknown/write operations are rejected.
5. Underlying stored choices remain fail-closed and reflect local withdrawal on the next check.
6. The listener closes and unlinks its socket on ordinary process shutdown.
7. On unexpected existing paths (including crash-stale sockets), startup refuses to overwrite them; any cleanup requires operator validation.

This single-user boundary cannot isolate multiple processes running under the *same* OS account, defend against a compromised account, replace external authentication, or guarantee end-to-end enforcement in source products.

## Packaging and validation

`packaging/systemd/goreeshieldd.service` provides an optional example of an OS-managed user service. It assumes an independently installed `goreeshieldd` binary at `%h/.local/bin/goreeshieldd`. No installer, package signature, service installation, service sandbox acceptance, update or rollback path, production data recovery or actual representative-device runtime has been verified.

GitHub CI tests Linux Go source with race detection and builds CLI/service packages; static analysis and formatting should run on *committed* code. A passing CI run is not a substitute for a live daemon installation audit.

## Remaining acceptance gates

Establish GoreeCloud Identity-backed authorization for any multi-user or remotely accessed service; native Linux service packaging and lifecycle verification; processing adapter contracts, idempotency and revocation; permission hardening, state recovery, upgrade migrations, accessibility and GoreeCloud UI work. Keep issues #3, #4 and #5 open pending production-grade scope and evidence.
