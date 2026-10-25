# GoreeShield Linux Local Choices Prototype

| Field | Value |
| --- | --- |
| Type | Engineering implementation note |
| Status | Implemented as a prototype in feature branch; not production-ready |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Source | `GoreeCloud/goreeshield/docs/prototype-linux.md` |

## Current implementation boundary

This prototype is a **local Linux command-line program**, not a deployed network service or complete native application. It uses the Go standard library and runs as the invoking local operating-system account. It contains **only synthetic demonstration purposes**, does not call the network, does not integrate GoreeCloud Identity, does not connect real products, and does not collect personal content. Linux is a **development target**, not a claimed supported release platform.

No browser wrapper is involved. Production-grade identity, authorization, confidential-source integrations, backup and recovery, packaging, update lifecycle, legal rights workflows, and GUI remain future milestones.

## Commands

```sh
go test -race ./...
go vet ./...
go build ./cmd/goreeshield
./goreeshield status
./goreeshield notice --purpose demo-local-metrics
./goreeshield grant --purpose demo-local-metrics --notice-version 1 --acknowledge
./goreeshield check --purpose demo-local-metrics
./goreeshield withdraw --purpose demo-local-metrics
```

The example grant is a **recorded local example choice**, not the activation of data processing. A user should read the full notice first; the `--acknowledge` flag is an explicit local command choice, not identity verification or a lawful consent certificate.

## Data handling

- Default storage: `$XDG_STATE_HOME/goreeshield/choices.json` or `$HOME/.local/state/goreeshield/choices.json`.
- Local state directory is private (0700); the choice and lock files are private (0600); symlink state files, unsafe permissions, malformed data, and unrecognized schemas fail closed.
- Updates serialize with an OS file lock and use atomic rename and file/directory sync; state stores only the last decision per synthetic purpose, its notice version and timestamp.
- Missing, unknown, withdrawn or stale-notice choices never permit an optional purpose.
- This is not an append-only audit ledger and has no guarantee of protection against a compromised local OS user, unsafe ancestors, backup exposure, or disk tampering.

## Acceptance performed during initial development

Go 1.23 local Linux run: `go test ./...`, `go vet ./...`, and `go build ./cmd/goreeshield` succeeded using synthetic test data. This evidence covers the local source assembled for this branch; ongoing CI and repository readback are tracked separately. No production service runtime, native installer, accessibility tests, or representative target-device release verification has occurred.

## Next engineering tasks

Complete threat model and policy/identity contract before connecting real products. Add OS-managed service lifecycle and packaging only after scoping the first native target. Preserve the distinction between local proof of concept and supported GoreeShield service.
