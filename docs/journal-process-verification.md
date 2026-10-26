# GoreeShield — Cross-Process Journal Recovery and Integrity Tests

| Field | Value |
| --- | --- |
| Owner | GoreeCloud |
| Date | October 25, 2026 |
| Status | Experimental Linux process-level verification; not a production acceptance test |
| Source | `GoreeCloud/goreeshield/docs/journal-process-verification.md` |
| Base | [Persistent synthetic source journal](durable-source.md) |

## Objective

A source journal that appears to work after constructing a second Go object is **not the same thing** as tested recovery across independent operating-system processes. This increment adds real Linux test subprocesses to exercise the existing per-scope `DiskSource` and its advisory `flock` coordination.

Only synthetic fixture identifiers, local consent, a demonstration counter and operation metadata are involved. No actual GoreeCloud Identity account, token, personal information, network endpoint, production filesystem or source product is used.

## What was implemented

- `internal/reconcile/disk_multiprocess_linux_test.go` invokes **four independent child test processes**. Each opens the same private fixture directory, submits the exact same operation ID, and reads the resulting synthetic source record. The parent separately opens a new `DiskSource` and checks that exactly one synthetic counter increment remains.
- `internal/reconcile/disk_storage_linux.go` refuses source journal and lock files that have multiple hard links (`Nlink != 1`) as well as existing unsafe type, owner and permission conditions. `O_NOFOLLOW` remains applied to both file opens. These measures constrain a set of simple link-based hazards but are not a complete defense against hostile same-user races.
- `internal/reconcile/disk_integrity_linux_test.go` verifies rejection of multiply linked journal and lock files, count/record inconsistency, unknown schemas, negative counters, trailing JSON and unknown JSON fields.
- The original CI gate is retained: committed formatting, Go race-detector tests, static analysis and Linux builds.

## Correct interpretation

Successful subprocess tests provide stronger **local cooperating-process evidence** than restarting an object in the same process. The lock and atomic state replacement ensure that the same synthetic operation does not increment the test counter twice when multiple cooperating Linux processes deliver it concurrently.

They still do **not** demonstrate real source effects or cross-host exactly-once delivery. A system restart after power loss, torn filesystem write, malicious same-UID actor, external side effects, distributed replicas, true user/tenant security isolation, signed source evidence, key rotation and full service lifecycle are all outside this experiment.

A cryptographic digest in the scoped filename is not encryption or anonymization. The ledger contains synthetic fixture identifiers and no real user content.

## Operational prerequisites still open

Real GoreeCloud Identity verifier and scopes; source-owned transactional adapter in an actual product; durable idempotency in the source's transactional datastore; systemd installation/upgrade/recovery tests on a representative Linux host; crash/fault injection around `fsync` and rename; storage backup and rollback protocol; tamper-evident and privacy-minimal source receipts; deployment review; UI design under GoreeCloud UI authority.

**Never interpret a green Go workflow as production security acceptance or legal privacy compliance.**
