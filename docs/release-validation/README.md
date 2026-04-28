# Release Validation Evidence

Status: historical evidence and audit trail.
Current release process lives in [RELEASING.md](../../RELEASING.md).
The stable major release gate lives in [stable-major-release-readiness.md](../stable-major-release-readiness.md).

Files in this branch preserve command results, release-gate notes, and one-time validation context.
They are not current release approval by themselves.
Before making a release decision, rerun the relevant commands against the candidate being tagged.

## Records

- [2026-04-24-v1.0.0-blocker-evidence.md](./2026-04-24-v1.0.0-blocker-evidence.md)
  Historical evidence gathered while validating the stable-major-release blocker list and the `v0.3.0` release gate.
  It records that commit-SHA fallback evidence was accepted at the time because the tag was not fetchable yet.

- [2026-04-28-v0.3.0-external-module-check.md](./2026-04-28-v0.3.0-external-module-check.md)
  Follow-up evidence that `v0.3.0` is now fetchable from a fresh external Go module and compiles against the documented public surface.

- [rb004-external-check-sha.log](./rb004-external-check-sha.log)
  Raw command log supporting the external module SHA fallback check.

## Reading Rule

Treat dated records here as snapshots.
If a record mentions tag availability, command output, local ports, external consumer paths, or product approval, read it as true for that run unless a newer validation record says otherwise.
